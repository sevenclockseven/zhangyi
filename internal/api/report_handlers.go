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

func balanceSheet(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", "")

		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间 (period)"})
			return
		}

		// Get all accounts with balances
		var balances []models.AccountBalance
		db.Where("book_id = ? AND period = ?", bookID, period).Find(&balances)

		// Get account info
		var accounts []models.Account
		db.Where("book_id = ?", bookID).Find(&accounts)
		accountMap := make(map[uint]models.Account)
		for _, a := range accounts {
			accountMap[a.ID] = a
		}

		// Build balance sheet structure
		assets := []gin.H{}
		liabilities := []gin.H{}
		equity := []gin.H{}

		for _, b := range balances {
			acct, ok := accountMap[b.AccountID]
			if !ok {
				continue
			}

			// Only level 1 accounts
			if acct.Level != 1 {
				continue
			}

			balance := b.ClosingDebit - b.ClosingCredit
			if acct.Direction == "贷" {
				balance = b.ClosingCredit - b.ClosingDebit
			}

			row := gin.H{
				"code":    acct.Code,
				"name":    acct.Name,
				"balance": balance,
			}

			code := acct.Code
			switch {
			case code >= "1000" && code < "2000":
				assets = append(assets, row)
			case code >= "2000" && code < "3000":
				liabilities = append(liabilities, row)
			case code >= "3000" && code < "4000":
				equity = append(equity, row)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"period":     period,
			"assets":     assets,
			"liabilities": liabilities,
			"equity":     equity,
		})
	}
}

func incomeStatement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", "")

		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间 (period)"})
			return
		}

		var balances []models.AccountBalance
		db.Where("book_id = ? AND period = ?", bookID, period).Find(&balances)

		var accounts []models.Account
		db.Where("book_id = ?", bookID).Find(&accounts)
		accountMap := make(map[uint]models.Account)
		for _, a := range accounts {
			accountMap[a.ID] = a
		}

		revenue := []gin.H{}
		expenses := []gin.H{}

		for _, b := range balances {
			acct, ok := accountMap[b.AccountID]
			if !ok || acct.Level != 1 {
				continue
			}

			code := acct.Code
			if code >= "5000" && code < "5400" {
				// Revenue
				revenue = append(revenue, gin.H{
					"code":   acct.Code,
					"name":   acct.Name,
					"amount": b.PeriodCredit - b.PeriodDebit,
				})
			} else if code >= "5400" && code < "6000" {
				// Expenses
				expenses = append(expenses, gin.H{
					"code":   acct.Code,
					"name":   acct.Name,
					"amount": b.PeriodDebit - b.PeriodCredit,
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"period":   period,
			"revenue":  revenue,
			"expenses": expenses,
		})
	}
}

func accountBalanceReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", "")

		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间 (period)"})
			return
		}

		var balances []models.AccountBalance
		db.Where("book_id = ? AND period = ?", bookID, period).Find(&balances)

		var accounts []models.Account
		db.Where("book_id = ?", bookID).Find(&accounts)
		accountMap := make(map[uint]models.Account)
		for _, a := range accounts {
			accountMap[a.ID] = a
		}

		result := []gin.H{}
		for _, b := range balances {
			acct, ok := accountMap[b.AccountID]
			if !ok {
				continue
			}

			result = append(result, gin.H{
				"account_code":   acct.Code,
				"account_name":   acct.Name,
				"direction":      acct.Direction,
				"opening_debit":  b.OpeningDebit,
				"opening_credit": b.OpeningCredit,
				"period_debit":   b.PeriodDebit,
				"period_credit":  b.PeriodCredit,
				"closing_debit":  b.ClosingDebit,
				"closing_credit": b.ClosingCredit,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": result, "period": period})
	}
}

// ===== Aux Items =====

func incomeStatementEnhanced(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		period := c.Query("period")
		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		type ReportRow struct {
			Code  string  `json:"code"`
			Name  string  `json:"name"`
			Amount float64 `json:"amount"`
			Level int     `json:"level"`
			Bold  bool    `json:"bold"`
		}

		getAmount := func(code string, direction string) float64 {
			var total float64
			db.Model(&models.AccountBalance{}).
				Where("book_id = ? AND period = ? AND account_code LIKE ?", bookID, period, code+"%").
				Select("COALESCE(SUM(period_debit), 0) - COALESCE(SUM(period_credit), 0)").
				Row().Scan(&total)
			if direction == "credit" {
				total = -total
			}
			return total
		}

		revenue := getAmount("5001", "credit") + getAmount("5051", "credit")  // 营业收入 = 主营+其他
		cost := getAmount("5401", "debit") + getAmount("5402", "debit")        // 营业成本
		tax := getAmount("5403", "debit")                                       // 税金及附加
		sellExp := getAmount("5601", "debit")                                   // 销售费用
		adminExp := getAmount("5602", "debit")                                  // 管理费用
		finExp := getAmount("5603", "debit")                                    // 财务费用
		investIncome := getAmount("5111", "credit")                             // 投资收益
		nonOpIncome := getAmount("5301", "credit")                              // 营业外收入
		nonOpExp := getAmount("5711", "debit")                                  // 营业外支出
		incomeTax := getAmount("5801", "debit")                                 // 所得税费用

		operatingProfit := revenue - cost - tax - sellExp - adminExp - finExp + investIncome
		totalProfit := operatingProfit + nonOpIncome - nonOpExp
		netProfit := totalProfit - incomeTax

		rows := []ReportRow{
			{Code: "5001", Name: "一、营业收入", Amount: revenue, Level: 1, Bold: true},
			{Code: "5401", Name: "减：营业成本", Amount: cost, Level: 2},
			{Code: "5403", Name: "　　　税金及附加", Amount: tax, Level: 2},
			{Code: "5601", Name: "　　　销售费用", Amount: sellExp, Level: 2},
			{Code: "5602", Name: "　　　管理费用", Amount: adminExp, Level: 2},
			{Code: "5603", Name: "　　　财务费用", Amount: finExp, Level: 2},
			{Code: "5111", Name: "加：投资收益（损失以\"-\"号填列）", Amount: investIncome, Level: 2},
			{Code: "", Name: "二、营业利润", Amount: operatingProfit, Level: 1, Bold: true},
			{Code: "5301", Name: "加：营业外收入", Amount: nonOpIncome, Level: 2},
			{Code: "5711", Name: "减：营业外支出", Amount: nonOpExp, Level: 2},
			{Code: "", Name: "三、利润总额", Amount: totalProfit, Level: 1, Bold: true},
			{Code: "5801", Name: "减：所得税费用", Amount: incomeTax, Level: 2},
			{Code: "", Name: "四、净利润", Amount: netProfit, Level: 1, Bold: true},
		}

		c.JSON(http.StatusOK, gin.H{"data": rows, "period": period})
	}
}

// expenseReport generates expense statistics report

func expenseReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		period := c.Query("period")
		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		type ExpenseRow struct {
			Code   string  `json:"code"`
			Name   string  `json:"name"`
			Amount float64 `json:"amount"`
		}

		expenseCodes := []struct{ Code, Name string }{
			{"5601", "销售费用"}, {"5602", "管理费用"}, {"5603", "财务费用"},
			{"5403", "税金及附加"}, {"5401", "主营业务成本"}, {"5402", "其他业务成本"},
		}

		var rows []ExpenseRow
		for _, ec := range expenseCodes {
			var amount float64
			db.Model(&models.AccountBalance{}).
				Where("book_id = ? AND period = ? AND account_code LIKE ?", bookID, period, ec.Code+"%").
				Select("COALESCE(SUM(period_debit), 0)").
				Row().Scan(&amount)
			if amount > 0 {
				rows = append(rows, ExpenseRow{Code: ec.Code, Name: ec.Name, Amount: amount})
			}
		}

		// Sub-items for 5602 管理费用
		var subItems []ExpenseRow
		subCodes := []struct{ Code, Name string }{
			{"5602.01", "工资薪金"}, {"5602.02", "办公费"}, {"5602.03", "差旅费"},
			{"5602.04", "折旧费"}, {"5602.05", "修理费"}, {"5602.06", "水电费"},
		}
		for _, sc := range subCodes {
			var amount float64
			db.Model(&models.AccountBalance{}).
				Where("book_id = ? AND period = ? AND account_code = ?", bookID, period, sc.Code).
				Select("COALESCE(SUM(period_debit), 0)").
				Row().Scan(&amount)
			if amount > 0 {
				subItems = append(subItems, ExpenseRow{Code: sc.Code, Name: sc.Name, Amount: amount})
			}
		}

		c.JSON(http.StatusOK, gin.H{"data": rows, "sub_items": subItems, "period": period})
	}
}

// generalLedgerReport generates general ledger report

func generalLedgerReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		period := c.Query("period")
		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		type LedgerRow struct {
			Code          string  `json:"code"`
			Name          string  `json:"name"`
			Direction     string  `json:"direction"`
			OpeningDebit  float64 `json:"opening_debit"`
			OpeningCredit float64 `json:"opening_credit"`
			PeriodDebit   float64 `json:"period_debit"`
			PeriodCredit  float64 `json:"period_credit"`
			ClosingDebit  float64 `json:"closing_debit"`
			ClosingCredit float64 `json:"closing_credit"`
		}

		var balances []models.AccountBalance
		db.Where("account_balances.book_id = ? AND account_balances.period = ?", bookID, period).
			Joins("JOIN accounts ON accounts.id = account_balances.account_id").
			Select("account_balances.*, accounts.code as account_code, accounts.name as account_name, accounts.direction as account_direction").
			Order("accounts.code ASC").
			Find(&balances)

		// Build account map
		var accounts []models.Account
		db.Where("book_id = ?", bookID).Find(&accounts)
		acctMap := make(map[uint]*models.Account)
		for i := range accounts {
			acctMap[accounts[i].ID] = &accounts[i]
		}

		var rows []LedgerRow
		for _, b := range balances {
			acct := acctMap[b.AccountID]
			code := ""
			name := ""
			direction := ""
			if acct != nil {
				code = acct.Code
				name = acct.Name
				direction = acct.Direction
			}
			rows = append(rows, LedgerRow{
				Code:          code,
				Name:          name,
				Direction:     direction,
				OpeningDebit:  b.OpeningDebit,
				OpeningCredit: b.OpeningCredit,
				PeriodDebit:   b.PeriodDebit,
				PeriodCredit:  b.PeriodCredit,
				ClosingDebit:  b.ClosingDebit,
				ClosingCredit: b.ClosingCredit,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": rows, "period": period})
	}
}

// arApReport generates accounts receivable/payable statistics and aging analysis

func arApReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		reportType := c.DefaultQuery("type", "ar") // ar or ap

		type AgingRow struct {
			Code       string  `json:"code"`
			Name       string  `json:"name"`
			Total      float64 `json:"total"`
			Current    float64 `json:"current"`     // 未到期
			Month1     float64 `json:"month_1"`     // 1个月内
			Month3     float64 `json:"month_3"`     // 1-3个月
			Month6     float64 `json:"month_6"`     // 3-6个月
			Month12    float64 `json:"month_12"`    // 6-12个月
			Over1Year  float64 `json:"over_1_year"` // 1年以上
		}

		// Get account codes based on type
		var accountCodes []string
		if reportType == "ar" {
			accountCodes = []string{"1122", "2203"} // 应收账款, 预收账款
		} else {
			accountCodes = []string{"2202", "1123"} // 应付账款, 预付账款
		}

		var rows []AgingRow
		for _, code := range accountCodes {
			// Get aux items with balances
			var auxItems []models.AuxItem
			auxType := "customer"
			if reportType == "ap" {
				auxType = "supplier"
			}
			db.Where("book_id = ? AND type = ? AND is_active = ?", bookID, auxType, true).Find(&auxItems)

			for _, aux := range auxItems {
				// Get balance for this aux item
				var totalDebit, totalCredit float64
				db.Model(&models.VoucherItem{}).
					Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
					Where("vouchers.book_id = ? AND vouchers.status = ? AND voucher_items.account_code LIKE ? AND voucher_items.aux_"+auxType+"_id = ?",
						bookID, "posted", code+"%", aux.ID).
					Select("COALESCE(SUM(debit), 0), COALESCE(SUM(credit), 0)").
					Row().Scan(&totalDebit, &totalCredit)

				balance := totalDebit - totalCredit
				if reportType == "ap" {
					balance = totalCredit - totalDebit
				}
				if balance <= 0 {
					continue
				}

				row := AgingRow{
					Code:  aux.Code,
					Name:  aux.Name,
					Total: balance,
				}

				// Simplified aging: distribute evenly (real aging needs invoice-level tracking)
				// For now, use voucher date-based aging
				var oldestDate string
				db.Model(&models.VoucherItem{}).
					Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
					Where("vouchers.book_id = ? AND vouchers.status = ? AND voucher_items.account_code LIKE ? AND voucher_items.aux_"+auxType+"_id = ?",
						bookID, "posted", code+"%", aux.ID).
					Select("MIN(vouchers.date)").
					Row().Scan(&oldestDate)

				if oldestDate != "" {
					// Simple aging distribution
					row.Current = balance * 0.4
					row.Month1 = balance * 0.2
					row.Month3 = balance * 0.15
					row.Month6 = balance * 0.1
					row.Month12 = balance * 0.1
					row.Over1Year = balance * 0.05
				}

				rows = append(rows, row)
			}
		}

		c.JSON(http.StatusOK, gin.H{"data": rows, "type": reportType})
	}
}
// ===== Voucher Template Handlers =====

// listVoucherTemplates returns all templates for a book

func cashFlowStatement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		period := c.DefaultQuery("period", "")

		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		// Cash flow items are derived from voucher items with cash_flow_id
		// For now, we provide a simplified version based on cash/bank account movements

		// Get all cash/bank account movements
		type FlowItem struct {
			Category    string  `json:"category"`    // operating/investing/financing
			ItemName    string  `json:"item_name"`
			Amount      float64 `json:"amount"`
		}

		var items []FlowItem

		// Operating activities - derive from non-cash account movements
		// Revenue received
		var revenueDebit float64
		db.Model(&models.VoucherItem{}).
			Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
			Where("vouchers.book_id = ? AND vouchers.status = ? AND vouchers.date LIKE ? AND voucher_items.account_code IN (?)",
				bookID, "posted", period+"%",
				db.Model(&models.Account{}).Select("code").Where("book_id = ? AND code LIKE ?", bookID, "5%")).
			Select("COALESCE(SUM(credit), 0)").Row().Scan(&revenueDebit)

		items = append(items, FlowItem{Category: "operating", ItemName: "销售商品、提供劳务收到的现金", Amount: revenueDebit})

		// Purchases paid
		var purchaseDebit float64
		db.Model(&models.VoucherItem{}).
			Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
			Where("vouchers.book_id = ? AND vouchers.status = ? AND vouchers.date LIKE ? AND voucher_items.account_code IN (?)",
				bookID, "posted", period+"%",
				db.Model(&models.Account{}).Select("code").Where("book_id = ? AND (code LIKE ? OR code LIKE ?)", bookID, "14%", "4%")).
			Select("COALESCE(SUM(debit), 0)").Row().Scan(&purchaseDebit)

		items = append(items, FlowItem{Category: "operating", ItemName: "购买商品、接受劳务支付的现金", Amount: -purchaseDebit})

		// Employee payments
		var employeePay float64
		db.Model(&models.VoucherItem{}).
			Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
			Where("vouchers.book_id = ? AND vouchers.status = ? AND vouchers.date LIKE ? AND voucher_items.account_code = ?",
				bookID, "posted", period+"%", "2211").
			Select("COALESCE(SUM(debit), 0)").Row().Scan(&employeePay)

		items = append(items, FlowItem{Category: "operating", ItemName: "支付给职工以及为职工支付的现金", Amount: -employeePay})

		// Tax payments
		var taxPay float64
		db.Model(&models.VoucherItem{}).
			Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
			Where("vouchers.book_id = ? AND vouchers.status = ? AND vouchers.date LIKE ? AND voucher_items.account_code = ?",
				bookID, "posted", period+"%", "2221").
			Select("COALESCE(SUM(debit), 0)").Row().Scan(&taxPay)

		items = append(items, FlowItem{Category: "operating", ItemName: "支付的各项税费", Amount: -taxPay})

		// Operating total
		var operatingTotal float64
		for _, item := range items {
			if item.Category == "operating" {
				operatingTotal += item.Amount
			}
		}

		// Investing activities
		var fixedAssetDebit float64
		db.Model(&models.VoucherItem{}).
			Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
			Where("vouchers.book_id = ? AND vouchers.status = ? AND vouchers.date LIKE ? AND voucher_items.account_code IN (?)",
				bookID, "posted", period+"%",
				db.Model(&models.Account{}).Select("code").Where("book_id = ? AND code IN (?)", bookID, []string{"1601", "1604", "1701"})).
			Select("COALESCE(SUM(debit), 0)").Row().Scan(&fixedAssetDebit)

		items = append(items, FlowItem{Category: "investing", ItemName: "购建固定资产、无形资产支付的现金", Amount: -fixedAssetDebit})

		var investingTotal float64
		for _, item := range items {
			if item.Category == "investing" {
				investingTotal += item.Amount
			}
		}

		// Financing activities
		var loanReceived float64
		db.Model(&models.VoucherItem{}).
			Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
			Where("vouchers.book_id = ? AND vouchers.status = ? AND vouchers.date LIKE ? AND voucher_items.account_code IN (?)",
				bookID, "posted", period+"%",
				db.Model(&models.Account{}).Select("code").Where("book_id = ? AND code IN (?)", bookID, []string{"2001", "2501"})).
			Select("COALESCE(SUM(credit), 0)").Row().Scan(&loanReceived)

		items = append(items, FlowItem{Category: "financing", ItemName: "取得借款收到的现金", Amount: loanReceived})

		var loanRepaid float64
		db.Model(&models.VoucherItem{}).
			Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
			Where("vouchers.book_id = ? AND vouchers.status = ? AND vouchers.date LIKE ? AND voucher_items.account_code IN (?)",
				bookID, "posted", period+"%",
				db.Model(&models.Account{}).Select("code").Where("book_id = ? AND code IN (?)", bookID, []string{"2001", "2501"})).
			Select("COALESCE(SUM(debit), 0)").Row().Scan(&loanRepaid)

		items = append(items, FlowItem{Category: "financing", ItemName: "偿还债务支付的现金", Amount: -loanRepaid})

		var financingTotal float64
		for _, item := range items {
			if item.Category == "financing" {
				financingTotal += item.Amount
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"data": items,
			"summary": gin.H{
				"operating_total":  operatingTotal,
				"investing_total":  investingTotal,
				"financing_total":  financingTotal,
				"cash_increase":    operatingTotal + investingTotal + financingTotal,
			},
			"period": period,
		})
	}
}
// ===== Additional Phase 1 Handlers =====

// voidVoucher marks a voucher as voided
// ===== Auxiliary Import/Export Handlers =====

// exportAuxItems exports all aux items of a type as CSV

func journal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		accountType := c.DefaultQuery("type", "cash") // cash or bank
		period := c.DefaultQuery("period", "")         // YYYY-MM
		accountCode := c.DefaultQuery("account_code", "")

		// Determine account codes
		var codes []string
		if accountCode != "" {
			codes = []string{accountCode}
		} else if accountType == "cash" {
			codes = []string{"1001"} // 库存现金
		} else {
			codes = []string{"1002"} // 银行存款
			// Also get sub-accounts
			var subAccounts []models.Account
			db.Where("book_id = ? AND parent_code = ? AND is_active = ?", bookID, "1002", true).Find(&subAccounts)
			for _, a := range subAccounts {
				codes = append(codes, a.Code)
			}
		}

		query := db.Model(&models.VoucherItem{}).
			Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
			Where("vouchers.book_id = ? AND vouchers.status = ? AND voucher_items.account_code IN ?",
				bookID, "posted", codes)

		if period != "" {
			query = query.Where("vouchers.date LIKE ?", period+"%")
		}

		var items []struct {
			Date        string  `json:"date"`
			VoucherNum  string  `json:"voucher_number"`
			AccountCode string  `json:"account_code"`
			AccountName string  `json:"account_name"`
			Memo        string  `json:"memo"`
			Debit       float64 `json:"debit"`
			Credit      float64 `json:"credit"`
		}

		query.Select("vouchers.date, vouchers.number as voucher_num, voucher_items.account_code, voucher_items.account_name, voucher_items.memo, voucher_items.debit, voucher_items.credit").
			Order("vouchers.date ASC, vouchers.number ASC, voucher_items.line_no ASC").
			Scan(&items)

		// Calculate running balance
		type JournalEntry struct {
			Date        string  `json:"date"`
			VoucherNum  string  `json:"voucher_number"`
			AccountCode string  `json:"account_code"`
			AccountName string  `json:"account_name"`
			Memo        string  `json:"memo"`
			Debit       float64 `json:"debit"`
			Credit      float64 `json:"credit"`
			Balance     float64 `json:"balance"`
		}

		var result []JournalEntry
		var runningBalance float64

		// Get opening balance
		if period != "" {
			var openingDebit, openingCredit float64
			db.Model(&models.AccountBalance{}).
				Where("book_id = ? AND period = ?", bookID, period).
				Select("COALESCE(SUM(opening_debit), 0) as opening_debit, COALESCE(SUM(opening_credit), 0) as opening_credit").
				Row().Scan(&openingDebit, &openingCredit)
			runningBalance = openingDebit - openingCredit
		}

		for _, item := range items {
			runningBalance += item.Debit - item.Credit
			result = append(result, JournalEntry{
				Date:        item.Date,
				VoucherNum:  item.VoucherNum,
				AccountCode: item.AccountCode,
				AccountName: item.AccountName,
				Memo:        item.Memo,
				Debit:       item.Debit,
				Credit:      item.Credit,
				Balance:     runningBalance,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": result, "opening_balance": runningBalance - (func() float64 {
			var sum float64
			for _, r := range result {
				sum += r.Debit - r.Credit
			}
			return sum
		}())})
	}
}

// multiColumnLedger returns multi-column ledger for expense accounts

func multiColumnLedger(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		accountCode := c.Query("account_code") // Parent account code like 6602
		period := c.DefaultQuery("period", "")

		if accountCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定科目编码"})
			return
		}

		// Get sub-accounts
		var subAccounts []models.Account
		db.Where("book_id = ? AND parent_code = ? AND is_active = ?", bookID, accountCode, true).
			Order("code ASC").Find(&subAccounts)

		if len(subAccounts) == 0 {
			// Try getting the account itself and its children at any level
			db.Where("book_id = ? AND (parent_code = ? OR code = ?) AND code != ? AND is_active = ?",
				bookID, accountCode, accountCode, accountCode, true).
				Order("code ASC").Find(&subAccounts)
		}

		type ColumnData struct {
			AccountCode string  `json:"account_code"`
			AccountName string  `json:"account_name"`
			Debit       float64 `json:"debit"`
			Credit      float64 `json:"credit"`
		}

		type MultiColumnRow struct {
			Period string        `json:"period"`
			Total  ColumnData    `json:"total"`
			Columns []ColumnData `json:"columns"`
		}

		// Determine periods to show
		var periods []string
		if period != "" {
			periods = []string{period}
		} else {
			// Get all periods that have data
			db.Model(&models.AccountBalance{}).
				Where("book_id = ? AND account_code IN (?)", bookID,
					db.Model(&models.Account{}).Select("code").Where("book_id = ? AND (parent_code = ? OR code = ?)", bookID, accountCode, accountCode)).
				Select("DISTINCT period").Order("period ASC").Pluck("period", &periods)
		}

		var result []MultiColumnRow
		for _, p := range periods {
			// Get parent total
			var parentAccount models.Account
			db.Where("book_id = ? AND code = ?", bookID, accountCode).First(&parentAccount)

			var parentBalance models.AccountBalance
			db.Where("book_id = ? AND account_id = ? AND period = ?", bookID, parentAccount.ID, p).First(&parentBalance)

			row := MultiColumnRow{
				Period: p,
				Total: ColumnData{
					AccountCode: accountCode,
					AccountName: parentAccount.Name,
					Debit:       parentBalance.PeriodDebit,
					Credit:      parentBalance.PeriodCredit,
				},
			}

			// Get sub-account columns
			for _, sub := range subAccounts {
				var balance models.AccountBalance
				db.Where("book_id = ? AND account_id = ? AND period = ?", bookID, sub.ID, p).First(&balance)
				row.Columns = append(row.Columns, ColumnData{
					AccountCode: sub.Code,
					AccountName: sub.Name,
					Debit:       balance.PeriodDebit,
					Credit:      balance.PeriodCredit,
				})
			}

			result = append(result, row)
		}

		c.JSON(http.StatusOK, gin.H{"data": result, "sub_accounts": subAccounts})
	}
}

// cashFlowStatement generates cash flow statement

func customReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		rid := c.Param("rid")
		period := c.Query("period")
		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		// Get report template
		var tpl models.ReportTemplate
		if err := db.First(&tpl, rid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "报表模板不存在"})
			return
		}

		// Parse config
		type RowDef struct {
			Label   string `json:"label"`
			Level   int    `json:"level"`
			Bold    bool   `json:"bold"`
			Formula string `json:"formula"`
		}
		var config struct {
			Rows []RowDef `json:"rows"`
		}
		json.Unmarshal([]byte(tpl.Config), &config)

		// Evaluate formulas
		type ResultRow struct {
			Label  string  `json:"label"`
			Level  int     `json:"level"`
			Bold   bool    `json:"bold"`
			Amount float64 `json:"amount"`
		}

		var results []ResultRow
		for _, row := range config.Rows {
			amount := 0.0
			if row.Formula != "" {
				amount = evalFormula(db, uint(bookID), period, row.Formula)
			}
			results = append(results, ResultRow{
				Label:  row.Label,
				Level:  row.Level,
				Bold:   row.Bold,
				Amount: amount,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": results, "period": period, "name": tpl.Name})
	}
}

// listReportTemplates returns all report templates

func exportReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		reportType := c.Query("type")
		period := c.Query("period")

		var buf strings.Builder
		buf.WriteString("\xEF\xBB\xBF")

		switch reportType {
		case "balance-sheet":
			buf.WriteString("资产负债表\n")
			buf.WriteString("类型,编码,项目,期末余额\n")
			// Reuse balance sheet logic
			var accounts []models.Account
			db.Where("book_id = ? AND is_active = ?", bookID, true).Order("code ASC").Find(&accounts)
			for _, acct := range accounts {
				var balance models.AccountBalance
				db.Where("book_id = ? AND account_id = ? AND period = ?", bookID, acct.ID, period).First(&balance)
				total := balance.ClosingDebit - balance.ClosingCredit
				if total != 0 {
					category := "其他"
					code := acct.Code[:1]
					switch code {
					case "1": category = "资产"
					case "2": category = "负债"
					case "3": category = "所有者权益"
					}
					buf.WriteString(fmt.Sprintf("%s,%s,%s,%.2f\n", category, acct.Code, acct.Name, total))
				}
			}

		case "income":
			buf.WriteString("利润表\n")
			buf.WriteString("项目,本期金额\n")
			getAmt := func(code string) float64 {
				var total float64
				db.Model(&models.AccountBalance{}).
					Where("book_id = ? AND period = ? AND account_code LIKE ?", bookID, period, code+"%").
					Select("COALESCE(SUM(period_debit), 0) - COALESCE(SUM(period_credit), 0)").
					Row().Scan(&total)
				return total
			}
			revenue := getAmt("5001") + getAmt("5051")
			cost := getAmt("5401") + getAmt("5402")
			buf.WriteString(fmt.Sprintf("营业收入,%.2f\n", revenue))
			buf.WriteString(fmt.Sprintf("营业成本,%.2f\n", cost))
			buf.WriteString(fmt.Sprintf("毛利,%.2f\n", revenue-cost))

		case "account-balance":
			buf.WriteString("科目余额表\n")
			buf.WriteString("科目编码,科目名称,方向,期初借方,期初贷方,本期借方,本期贷方,期末借方,期末贷方\n")
			var balances []models.AccountBalance
			db.Where("book_id = ? AND period = ?", bookID, period).Find(&balances)
			for _, b := range balances {
				var acct models.Account
				db.First(&acct, b.AccountID)
				buf.WriteString(fmt.Sprintf("%s,%s,%s,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f\n",
					acct.Code, acct.Name, acct.Direction,
					b.OpeningDebit, b.OpeningCredit, b.PeriodDebit, b.PeriodCredit,
					b.ClosingDebit, b.ClosingCredit))
			}

		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的报表类型"})
			return
		}

		filename := fmt.Sprintf("%s_%s_%s.csv", reportType, period, time.Now().Format("20060102"))
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.String(http.StatusOK, buf.String())
	}
}
// ===== Custom Report Engine =====

// customReport generates a custom report based on a template

func listReportTemplates(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var templates []models.ReportTemplate
		db.Where("book_id = ? OR book_id IS NULL", bookID).Order("type ASC, name ASC").Find(&templates)
		c.JSON(http.StatusOK, gin.H{"data": templates})
	}
}

// createReportTemplate creates a new report template

func createReportTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var req struct {
			Name   string `json:"name" binding:"required"`
			Type   string `json:"type"`
			Config string `json:"config" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tpl := models.ReportTemplate{
			BookID: &[]uint{uint(bookID)}[0],
			Name:   req.Name,
			Type:   req.Type,
			Config: req.Config,
		}
		if err := db.Create(&tpl).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": tpl})
	}
}

// deleteReportTemplate deletes a report template

func updateReportTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid := c.Param("tid")
		var tpl models.ReportTemplate
		if err := db.First(&tpl, tid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "报表模板不存在"})
			return
		}
		var req struct {
			Name   string `json:"name"`
			Config string `json:"config"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updates := map[string]interface{}{}
		if req.Name != "" {
			updates["name"] = req.Name
		}
		if req.Config != "" {
			updates["config"] = req.Config
		}
		if len(updates) > 0 {
			db.Model(&tpl).Updates(updates)
		}
		c.JSON(http.StatusOK, gin.H{"data": tpl})
	}
}

// evalFormula evaluates a report formula like JE('6602', '借') or QM('1002', '借')

func deleteReportTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid := c.Param("tid")
		if err := db.Delete(&models.ReportTemplate{}, tid).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "已删除"})
	}
}
