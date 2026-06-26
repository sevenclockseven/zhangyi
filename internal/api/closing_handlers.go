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
		if err := db.First(&book, bookID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "账套不存在"})
			return
		}

		// 1. 获取所有损益类科目及其余额
		var accounts []models.Account
		db.Where("book_id = ? AND is_active = ?", bookID, true).Find(&accounts)
		accountMap := make(map[uint]models.Account)
		for _, a := range accounts {
			accountMap[a.ID] = a
		}

		var balances []models.AccountBalance
		db.Where("book_id = ? AND period = ?", bookID, period).Find(&balances)
		balanceMap := make(map[uint]models.AccountBalance)
		for _, b := range balances {
			balanceMap[b.AccountID] = b
		}

		// 2. 找到本年利润科目（3103小企业/3131企业）
		var profitAccount models.Account
		if err := db.Where("book_id = ? AND code = ?", bookID, "3103").First(&profitAccount).Error; err != nil {
			if err := db.Where("book_id = ? AND code = ?", bookID, "3131").First(&profitAccount).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "未找到本年利润科目(3103/3131)，请先添加"})
				return
			}
		}

		// 3. 计算损益类科目本期净额
		type TransferItem struct {
			AccountID   uint
			AccountCode string
			AccountName string
			Debit       float64
			Credit      float64
		}

		var items []TransferItem
		var totalIncome, totalExpense float64

		for _, acct := range accounts {
			if !acct.IsActive {
				continue
			}
			code := acct.Code
			// 损益类科目：5000-5999
			if code < "5000" || code >= "6000" {
				continue
			}
			b, ok := balanceMap[acct.ID]
			if !ok {
				continue
			}

			// 收入类（5000-5399）：贷方余额=收入
			if code >= "5000" && code < "5400" {
				net := b.PeriodCredit - b.PeriodDebit
				if net != 0 {
					items = append(items, TransferItem{acct.ID, acct.Code, acct.Name, net, 0})
					totalIncome += net
				}
			} else if code >= "5400" && code < "6000" {
				// 费用类（5400-5999）：借方余额=费用
				net := b.PeriodDebit - b.PeriodCredit
				if net != 0 {
					items = append(items, TransferItem{acct.ID, acct.Code, acct.Name, 0, net})
					totalExpense += net
				}
			}
		}

		if len(items) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "无损益科目需要结转", "count": 0})
			return
		}

		// 4. 生成结转凭证
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

		// 收入结转分录
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

		// 本年利润汇总
		netProfit := totalIncome - totalExpense
		if netProfit > 0 {
			// 盈利：收入>费用
			tx.Create(&models.VoucherItem{
				VoucherID: voucher.ID, LineNo: lineNo,
				AccountID: profitAccount.ID, AccountCode: profitAccount.Code, AccountName: profitAccount.Name,
				Debit: 0, Credit: totalIncome, Memo: "收入结转",
			})
			lineNo++
			tx.Create(&models.VoucherItem{
				VoucherID: voucher.ID, LineNo: lineNo,
				AccountID: profitAccount.ID, AccountCode: profitAccount.Code, AccountName: profitAccount.Name,
				Debit: totalExpense, Credit: 0, Memo: "费用结转",
			})
		} else if netProfit < 0 {
			// 亏损：费用>收入
			tx.Create(&models.VoucherItem{
				VoucherID: voucher.ID, LineNo: lineNo,
				AccountID: profitAccount.ID, AccountCode: profitAccount.Code, AccountName: profitAccount.Name,
				Debit: 0, Credit: totalIncome, Memo: "收入结转",
			})
			lineNo++
			tx.Create(&models.VoucherItem{
				VoucherID: voucher.ID, LineNo: lineNo,
				AccountID: profitAccount.ID, AccountCode: profitAccount.Code, AccountName: profitAccount.Name,
				Debit: totalExpense, Credit: 0, Memo: "费用结转",
			})
		}

		// 5. 自动记账（更新科目余额）
		if err := updateAccountBalances(tx, &voucher); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新科目余额失败: " + err.Error()})
			return
		}

		// 6. 更新凭证状态为已记账
		tx.Model(&voucher).Updates(map[string]interface{}{
			"status":    "posted",
			"posted_by": "system",
		})

		tx.Commit()

		c.JSON(http.StatusOK, gin.H{
			"message":        "结转凭证生成并记账成功",
			"voucher_id":     voucher.ID,
			"voucher_number": voucher.Number,
			"income_total":   totalIncome,
			"expense_total":  totalExpense,
			"net_profit":     netProfit,
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

		// 1. 检查是否有未记账凭证
		var draftCount int64
		db.Model(&models.Voucher{}).Where("book_id = ? AND date LIKE ? AND status IN (?)", bookID, period+"%", []string{"draft", "reviewed"}).Count(&draftCount)
		if draftCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("还有 %d 张凭证未记账，请先完成记账", draftCount)})
			return
		}

		// 2. 更新科目余额：将 period -> closing
		var balances []models.AccountBalance
		db.Where("book_id = ? AND period = ?", bookID, period).Find(&balances)
		for _, b := range balances {
			closingDebit := b.OpeningDebit + b.PeriodDebit
			closingCredit := b.OpeningCredit + b.PeriodCredit
			db.Model(&b).Updates(map[string]interface{}{
				"closing_debit":  closingDebit,
				"closing_credit": closingCredit,
			})
		}

		// 3. 创建下一期间的期初余额记录
		// Parse period to get next period
		var year, month int
		fmt.Sscanf(period, "%d-%d", &year, &month)
		nextMonth := month + 1
		nextYear := year
		if nextMonth > 12 {
			nextMonth = 1
			nextYear++
		}
		nextPeriod := fmt.Sprintf("%04d-%02d", nextYear, nextMonth)

		// Check if next period already exists
		var existingCount int64
		db.Model(&models.AccountBalance{}).Where("book_id = ? AND period = ?", bookID, nextPeriod).Count(&existingCount)
		if existingCount == 0 {
			for _, b := range balances {
				nextBalance := models.AccountBalance{
					BookID:        b.BookID,
					AccountID:     b.AccountID,
					Period:        nextPeriod,
					OpeningDebit:  b.ClosingDebit,
					OpeningCredit: b.ClosingCredit,
					AuxKey:        b.AuxKey,
				}
				db.Create(&nextBalance)
			}
		}

		// 4. 标记账套已结账
		db.Model(&models.AccountBook{}).Where("id = ?", bookID).Update("status", "closed")

		c.JSON(http.StatusOK, gin.H{
			"message":       period + " 期间结账成功",
			"period":        period,
			"next_period":   nextPeriod,
			"balance_count": len(balances),
		})
	}
}

func unclosePeriod(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", "")
		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		// 1. 计算下一期间
		var year, month int
		fmt.Sscanf(period, "%d-%d", &year, &month)
		nextMonth := month + 1
		nextYear := year
		if nextMonth > 12 {
			nextMonth = 1
			nextYear++
		}
		nextPeriod := fmt.Sprintf("%04d-%02d", nextYear, nextMonth)

		// 2. 删除下一期间的期初余额记录
		db.Where("book_id = ? AND period = ?", bookID, nextPeriod).Delete(&models.AccountBalance{})

		// 3. 恢复账套状态
		db.Model(&models.AccountBook{}).Where("id = ?", bookID).Update("status", "active")

		c.JSON(http.StatusOK, gin.H{
			"message":     period + " 反结账成功",
			"next_period": nextPeriod,
		})
	}
}

func closingStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", "")

		var book models.AccountBook
		db.First(&book, bookID)

		var voucherCount, draftCount, postedCount int64
		db.Model(&models.Voucher{}).Where("book_id = ? AND date LIKE ?", bookID, period+"%").Count(&voucherCount)
		db.Model(&models.Voucher{}).Where("book_id = ? AND date LIKE ? AND status = ?", bookID, period+"%", "draft").Count(&draftCount)
		db.Model(&models.Voucher{}).Where("book_id = ? AND date LIKE ? AND status = ?", bookID, period+"%", "posted").Count(&postedCount)

		// Check if next period has opening balances (indicates closed)
		var year, month int
		fmt.Sscanf(period, "%d-%d", &year, &month)
		nextMonth := month + 1
		nextYear := year
		if nextMonth > 12 {
			nextMonth = 1
			nextYear++
		}
		nextPeriod := fmt.Sprintf("%04d-%02d", nextYear, nextMonth)
		var nextCount int64
		db.Model(&models.AccountBalance{}).Where("book_id = ? AND period = ?", bookID, nextPeriod).Count(&nextCount)

		// Get latest posted voucher date as current_period if period is empty
		currentPeriod := period
		if currentPeriod == "" {
			var latestVoucher models.Voucher
			if err := db.Where("book_id = ? AND status = ?", bookID, "posted").Order("date DESC").First(&latestVoucher).Error; err == nil && len(latestVoucher.Date) >= 7 {
				currentPeriod = latestVoucher.Date[:7]
			} else {
				// Fallback to book start_date
				currentPeriod = book.StartDate
			}
		}

		isClosed := book.Status == "closed" || nextCount > 0

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"current_period":  currentPeriod,
				"is_closed":       isClosed,
				"unposted_count":  draftCount,
				"closed_at":       "",
				"period":          currentPeriod,
				"voucher_count":   voucherCount,
				"draft_count":     draftCount,
				"posted_count":    postedCount,
				"book_status":     book.Status,
			},
		})
	}
}
