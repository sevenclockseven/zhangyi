package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/models"
	"github.com/sevenclockseven/zhangyi/internal/services"
)


// Template directory - can be overridden by env var

func getOpeningBalances(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		period := c.Query("period")

		if period == "" {
			// Get book start date
			var book models.AccountBook
			if err := db.First(&book, bookID).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "账套不存在"})
				return
			}
			period = book.StartDate
		}

		// Get all accounts
		var accounts []models.Account
		db.Where("book_id = ? AND is_active = ?", bookID, true).Order("code ASC").Find(&accounts)

		// Get existing opening balances
		var balances []models.OpeningBalance
		db.Where("book_id = ? AND period = ?", bookID, period).Find(&balances)

		// Create balance map
		balanceMap := make(map[uint]*models.OpeningBalance)
		for i := range balances {
			balanceMap[balances[i].AccountID] = &balances[i]
		}

		// Build result
		type BalanceRow struct {
			AccountID    uint    `json:"account_id"`
			AccountCode  string  `json:"account_code"`
			AccountName  string  `json:"account_name"`
			Direction    string  `json:"direction"`
			Level        int     `json:"level"`
			IsLeaf       bool    `json:"is_leaf"`
			OpeningDebit float64 `json:"opening_debit"`
			OpeningCredit float64 `json:"opening_credit"`
		}

		var result []BalanceRow
		for _, acct := range accounts {
			row := BalanceRow{
				AccountID:   acct.ID,
				AccountCode: acct.Code,
				AccountName: acct.Name,
				Direction:   acct.Direction,
				Level:       acct.Level,
				IsLeaf:      acct.IsLeaf,
			}
			if b, ok := balanceMap[acct.ID]; ok {
				row.OpeningDebit = b.Debit
				row.OpeningCredit = b.Credit
			}
			result = append(result, row)
		}

		c.JSON(http.StatusOK, gin.H{"data": result, "period": period})
	}
}

// saveOpeningBalances saves opening balances (batch)

func saveOpeningBalances(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

		var req struct {
			Period   string `json:"period"`
			Balances []struct {
				AccountID     uint    `json:"account_id"`
				OpeningDebit  float64 `json:"opening_debit"`
				OpeningCredit float64 `json:"opening_credit"`
			} `json:"balances"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Period == "" {
			var book models.AccountBook
			db.First(&book, bookID)
			req.Period = book.StartDate
		}

		tx := db.Begin()

		for _, b := range req.Balances {
			if b.OpeningDebit == 0 && b.OpeningCredit == 0 {
				// Delete empty balance
				tx.Where("book_id = ? AND account_id = ? AND period = ?", bookID, b.AccountID, req.Period).Delete(&models.OpeningBalance{})
				continue
			}

			// Upsert
			var existing models.OpeningBalance
			result := tx.Where("book_id = ? AND account_id = ? AND period = ? AND aux_key = ?",
				bookID, b.AccountID, req.Period, "").First(&existing)

			if result.Error == gorm.ErrRecordNotFound {
				tx.Create(&models.OpeningBalance{
					BookID:    uint(bookID),
					AccountID: b.AccountID,
					Period:    req.Period,
					Debit:     b.OpeningDebit,
					Credit:    b.OpeningCredit,
					AuxKey:    "",
				})
			} else {
				tx.Model(&existing).Updates(map[string]interface{}{
					"debit":  b.OpeningDebit,
					"credit": b.OpeningCredit,
				})
			}
		}

		// Update account_balances table
		// Clear existing opening balances for this period
		tx.Where("book_id = ? AND period = ?", bookID, req.Period).Delete(&models.AccountBalance{})

		// Rebuild from opening balances + posted vouchers
		var accounts []models.Account
		tx.Where("book_id = ?", bookID).Find(&accounts)

		for _, acct := range accounts {
			var ob models.OpeningBalance
			tx.Where("book_id = ? AND account_id = ? AND period = ?", bookID, acct.ID, req.Period).First(&ob)

			// Calculate period totals from posted vouchers
			var periodDebit, periodCredit float64
			tx.Model(&models.VoucherItem{}).
				Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
				Where("vouchers.book_id = ? AND vouchers.status = ? AND vouchers.date LIKE ? AND voucher_items.account_id = ?",
					bookID, "posted", req.Period+"%", acct.ID).
				Select("COALESCE(SUM(debit), 0), COALESCE(SUM(credit), 0)").
				Row().Scan(&periodDebit, &periodCredit)

			closingDebit := ob.Debit + periodDebit
			closingCredit := ob.Credit + periodCredit

			tx.Create(&models.AccountBalance{
				BookID:        uint(bookID),
				AccountID:     acct.ID,
				Period:        req.Period,
				OpeningDebit:  ob.Debit,
				OpeningCredit: ob.Credit,
				PeriodDebit:   periodDebit,
				PeriodCredit:  periodCredit,
				ClosingDebit:  closingDebit,
				ClosingCredit: closingCredit,
			})
		}

		tx.Commit()

		c.JSON(http.StatusOK, gin.H{"message": "期初余额保存成功"})
	}
}
// ===== Opening Balance Import/Export =====

// exportOpeningBalances exports opening balances as CSV

func exportOpeningBalances(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		period := c.Query("period")

		if period == "" {
			var book models.AccountBook
			db.First(&book, bookID)
			period = book.StartDate
		}

		// Get accounts with balances
		var accounts []models.Account
		db.Where("book_id = ? AND is_active = ?", bookID, true).Order("code ASC").Find(&accounts)

		balanceMap := make(map[uint]*models.OpeningBalance)
		var balances []models.OpeningBalance
		db.Where("book_id = ? AND period = ?", bookID, period).Find(&balances)
		for i := range balances {
			balanceMap[balances[i].AccountID] = &balances[i]
		}

		var buf strings.Builder
		buf.WriteString("\xEF\xBB\xBF") // BOM
		buf.WriteString("科目编码,科目名称,方向,期初借方,期初贷方\n")

		for _, acct := range accounts {
			if !acct.IsLeaf {
				continue // Only export leaf accounts
			}
			var debit, credit float64
			if b, ok := balanceMap[acct.ID]; ok {
				debit = b.Debit
				credit = b.Credit
			}
			buf.WriteString(fmt.Sprintf("%s,%s,%s,%.2f,%.2f\n",
				quoteCSV(acct.Code), quoteCSV(acct.Name), acct.Direction, debit, credit))
		}

		filename := fmt.Sprintf("opening_balance_%s_%s.csv", period, time.Now().Format("20060102"))
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.String(http.StatusOK, buf.String())
	}
}

// importOpeningBalances imports opening balances from CSV

func importOpeningBalances(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		period := c.Query("period")

		if period == "" {
			var book models.AccountBook
			db.First(&book, bookID)
			period = book.StartDate
		}

		file, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
			return
		}
		defer file.Close()

		content, _ := io.ReadAll(file)
		text := string(content)
		if len(text) > 3 && text[:3] == "\xEF\xBB\xBF" {
			text = text[3:]
		}

		lines := strings.Split(strings.TrimSpace(text), "\n")
		if len(lines) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件内容为空"})
			return
		}

		// Build account code->id map
		var accounts []models.Account
		db.Where("book_id = ?", bookID).Find(&accounts)
		codeToID := make(map[string]uint)
		for _, a := range accounts {
			codeToID[a.Code] = a.ID
		}

		tx := db.Begin()
		var imported, skipped int

		for _, line := range lines[1:] {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fields := parseCSVLine(line)
			if len(fields) < 5 {
				skipped++
				continue
			}

			code := fields[0]
			accountID, ok := codeToID[code]
			if !ok {
				skipped++
				continue
			}

			debit, _ := strconv.ParseFloat(fields[3], 64)
			credit, _ := strconv.ParseFloat(fields[4], 64)

			if debit == 0 && credit == 0 {
				// Delete empty
				tx.Where("book_id = ? AND account_id = ? AND period = ?", bookID, accountID, period).Delete(&models.OpeningBalance{})
				continue
			}

			// Upsert
			var existing models.OpeningBalance
			result := tx.Where("book_id = ? AND account_id = ? AND period = ? AND aux_key = ?",
				bookID, accountID, period, "").First(&existing)

			if result.Error == gorm.ErrRecordNotFound {
				tx.Create(&models.OpeningBalance{
					BookID:    uint(bookID),
					AccountID: accountID,
					Period:    period,
					Debit:     debit,
					Credit:    credit,
					AuxKey:    "",
				})
			} else {
				tx.Model(&existing).Updates(map[string]interface{}{
					"debit":  debit,
					"credit": credit,
				})
			}
			imported++
		}

		// Rebuild account_balances
		tx.Where("book_id = ? AND period = ?", bookID, period).Delete(&models.AccountBalance{})
		for _, acct := range accounts {
			var ob models.OpeningBalance
			tx.Where("book_id = ? AND account_id = ? AND period = ?", bookID, acct.ID, period).First(&ob)

			var periodDebit, periodCredit float64
			tx.Model(&models.VoucherItem{}).
				Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
				Where("vouchers.book_id = ? AND vouchers.status = ? AND vouchers.date LIKE ? AND voucher_items.account_id = ?",
					bookID, "posted", period+"%", acct.ID).
				Select("COALESCE(SUM(debit), 0), COALESCE(SUM(credit), 0)").
				Row().Scan(&periodDebit, &periodCredit)

			tx.Create(&models.AccountBalance{
				BookID:        uint(bookID),
				AccountID:     acct.ID,
				Period:        period,
				OpeningDebit:  ob.Debit,
				OpeningCredit: ob.Credit,
				PeriodDebit:   periodDebit,
				PeriodCredit:  periodCredit,
				ClosingDebit:  ob.Debit + periodDebit,
				ClosingCredit: ob.Credit + periodCredit,
			})
		}

		tx.Commit()

		c.JSON(http.StatusOK, gin.H{
			"message":  fmt.Sprintf("导入完成：导入%d条，跳过%d条", imported, skipped),
			"imported": imported,
			"skipped":  skipped,
		})
	}
}
// ===== Enhanced Report Handlers =====

// incomeStatementEnhanced generates proper income statement per tax bureau format
