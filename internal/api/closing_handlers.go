package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/models"
)

func trialBalance(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", "")
		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		var balances []models.AccountBalance
		db.Where("book_id = ? AND period = ?", bookID, period).Find(&balances)

		var totalDebit, totalCredit float64
		for _, b := range balances {
			totalDebit += b.ClosingDebit
			totalCredit += b.ClosingCredit
		}

		balanced := fmt.Sprintf("%.2f", totalDebit) == fmt.Sprintf("%.2f", totalCredit)
		c.JSON(http.StatusOK, gin.H{
			"total_debit":  totalDebit,
			"total_credit": totalCredit,
			"balanced":     balanced,
		})
	}
}

func autoTransfer(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", "")
		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		var book models.AccountBook
		db.First(&book, bookID)

		// 获取所有损益类科目余额
		var balances []models.AccountBalance
		db.Where("book_id = ? AND period = ?", bookID, period).Find(&balances)

		var accounts []models.Account
		db.Where("book_id = ?", bookID).Find(&accounts)
		accountMap := make(map[uint]models.Account)
		for _, a := range accounts {
			accountMap[a.ID] = a
		}

		// 找到本年利润科目
		var profitAccount models.Account
		if err := db.Where("book_id = ? AND code = ?", bookID, "3103").First(&profitAccount).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "未找到本年利润科目(3103)"})
			return
		}

		type TransferItem struct {
			AccountID   uint    `json:"account_id"`
			AccountCode string  `json:"account_code"`
			AccountName string  `json:"account_name"`
			Debit       float64 `json:"debit"`
			Credit      float64 `json:"credit"`
		}

		var items []TransferItem
		var totalIncome, totalExpense float64

		for _, b := range balances {
			acct, ok := accountMap[b.AccountID]
			if !ok || acct.Level != 1 {
				continue
			}
			code := acct.Code
			// 收入类（5000-5399）
			if code >= "5000" && code < "5400" {
				balance := b.PeriodCredit - b.PeriodDebit
				if balance > 0 {
					items = append(items, TransferItem{acct.ID, acct.Code, acct.Name, balance, 0})
					totalIncome += balance
				}
			}
			// 费用类（5400-5999）
			if code >= "5400" && code < "6000" {
				balance := b.PeriodDebit - b.PeriodCredit
				if balance > 0 {
					items = append(items, TransferItem{acct.ID, acct.Code, acct.Name, 0, balance})
					totalExpense += balance
				}
			}
		}

		if len(items) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "无损益科目需要结转", "count": 0})
			return
		}

		// 生成结转凭证
		date := period + "-28"
		number := generateVoucherNumber(db, book.ID, date)

		voucher := models.Voucher{
			BookID:      book.ID,
			Date:        date,
			Number:      number,
			VoucherType: "general",
			Status:      "draft",
			TotalDebit:  totalIncome + totalExpense,
			TotalCredit: totalIncome + totalExpense,
			Memo:        period + " 损益结转",
		}

		tx := db.Begin()
		if err := tx.Create(&voucher).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		lineNo := 1
		for _, item := range items {
			vi := models.VoucherItem{
				VoucherID:   voucher.ID,
				LineNo:      lineNo,
				AccountID:   item.AccountID,
				AccountCode: item.AccountCode,
				AccountName: item.AccountName,
				Debit:       item.Debit,
				Credit:      item.Credit,
				Memo:        "损益结转",
			}
			tx.Create(&vi)
			lineNo++
		}

		// 本年利润分录
		if totalIncome > 0 {
			tx.Create(&models.VoucherItem{
				VoucherID: voucher.ID, LineNo: lineNo,
				AccountID: profitAccount.ID, AccountCode: profitAccount.Code, AccountName: profitAccount.Name,
				Debit: 0, Credit: totalIncome, Memo: "收入结转",
			})
			lineNo++
		}
		if totalExpense > 0 {
			tx.Create(&models.VoucherItem{
				VoucherID: voucher.ID, LineNo: lineNo,
				AccountID: profitAccount.ID, AccountCode: profitAccount.Code, AccountName: profitAccount.Name,
				Debit: totalExpense, Credit: 0, Memo: "费用结转",
			})
		}

		tx.Commit()

		c.JSON(http.StatusOK, gin.H{
			"message":        "结转凭证生成成功",
			"voucher_id":     voucher.ID,
			"voucher_number": voucher.Number,
			"income_total":   totalIncome,
			"expense_total":  totalExpense,
			"item_count":     len(items),
		})
	}
}

func closePeriod(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", "")
		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		var draftCount int64
		db.Model(&models.Voucher{}).Where("book_id = ? AND date LIKE ? AND status IN (?)", bookID, period+"%", []string{"draft", "reviewed"}).Count(&draftCount)
		if draftCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("还有 %d 张凭证未记账，请先完成记账", draftCount)})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": period + " 期间结账成功"})
	}
}

func unclosePeriod(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "反结账成功"})
	}
}

func closingStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", "")

		var voucherCount, draftCount, postedCount int64
		db.Model(&models.Voucher{}).Where("book_id = ? AND date LIKE ?", bookID, period+"%").Count(&voucherCount)
		db.Model(&models.Voucher{}).Where("book_id = ? AND date LIKE ? AND status = ?", bookID, period+"%", "draft").Count(&draftCount)
		db.Model(&models.Voucher{}).Where("book_id = ? AND date LIKE ? AND status = ?", bookID, period+"%", "posted").Count(&postedCount)

		c.JSON(http.StatusOK, gin.H{
			"period":        period,
			"voucher_count": voucherCount,
			"draft_count":   draftCount,
			"posted_count":  postedCount,
			"closed":        false,
		})
	}
}
