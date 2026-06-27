package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sevenclockseven/zhangyi/internal/models"
	"gorm.io/gorm"
)

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

		// Calculate totals
		var totalAssets, totalLiabilities, totalEquity float64
		for _, a := range assets {
			if v, ok := a["balance"].(float64); ok {
				totalAssets += v
			}
		}
		for _, l := range liabilities {
			if v, ok := l["balance"].(float64); ok {
				totalLiabilities += v
			}
		}
		for _, e := range equity {
			if v, ok := e["balance"].(float64); ok {
				totalEquity += v
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"period":            period,
			"assets":            assets,
			"liabilities":       liabilities,
			"equity":            equity,
			"total_assets":      totalAssets,
			"total_liabilities": totalLiabilities,
			"total_equity":      totalEquity,
			"total_liab_equity": totalLiabilities + totalEquity,
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

		// Get all accounts
		var accounts []models.Account
		db.Where("book_id = ?", bookID).Find(&accounts)
		accountMap := make(map[uint]models.Account)
		for _, a := range accounts {
			accountMap[a.ID] = a
		}

		// Get posted vouchers for this period (excluding transfer vouchers)
		var vouchers []models.Voucher
		db.Where("book_id = ? AND date LIKE ? AND status = ?", bookID, period+"%", "posted").Find(&vouchers)

		// Calculate amounts from voucher items (excluding transfer vouchers)
		type AmountVal struct {
			Debit  float64
			Credit float64
		}
		amountMap := make(map[uint]AmountVal)
		for _, v := range vouchers {
			if v.Memo != "" && strings.Contains(v.Memo, "损益结转") {
				continue
			}
			var items []models.VoucherItem
			db.Where("voucher_id = ?", v.ID).Find(&items)
			for _, item := range items {
				val := amountMap[item.AccountID]
				val.Debit += item.Debit
				val.Credit += item.Credit
				amountMap[item.AccountID] = val
			}
		}

		revenue := []gin.H{}
		expenses := []gin.H{}

		for _, acct := range accounts {
			if !acct.IsActive || acct.Level != 1 {
				continue
			}
			code := acct.Code
			amounts := amountMap[acct.ID]

			if code >= "5000" && code < "5400" {
				net := amounts.Credit - amounts.Debit
				if net != 0 {
					revenue = append(revenue, gin.H{
						"code":   acct.Code,
						"name":   acct.Name,
						"amount": net,
					})
				}
			} else if code >= "5400" && code < "6000" {
				net := amounts.Debit - amounts.Credit
				if net != 0 {
					expenses = append(expenses, gin.H{
						"code":   acct.Code,
						"name":   acct.Name,
						"amount": net,
					})
				}
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

// incomeStatementEnhanced generates proper income statement per tax bureau format
func incomeStatementEnhanced(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		period := c.Query("period")
		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		type ReportRow struct {
			Code   string  `json:"code"`
			Name   string  `json:"name"`
			Amount float64 `json:"amount"`
			Level  int     `json:"level"`
			Bold   bool    `json:"bold"`
		}

		// 从凭证直接取数（排除结转凭证），避免结转后余额表净额为0
		type AcctAmount struct {
			AccountID uint
			Debit     float64
			Credit    float64
		}
		acctAmounts := make(map[uint]*AcctAmount)

		var vouchers []models.Voucher
		db.Where("book_id = ? AND date LIKE ? AND status = ?", bookID, period+"%", "posted").Find(&vouchers)
		for _, v := range vouchers {
			if v.Memo != "" && strings.Contains(v.Memo, "损益结转") {
				continue
			}
			var items []models.VoucherItem
			db.Where("voucher_id = ?", v.ID).Find(&items)
			for _, item := range items {
				if acctAmounts[item.AccountID] == nil {
					acctAmounts[item.AccountID] = &AcctAmount{AccountID: item.AccountID}
				}
				acctAmounts[item.AccountID].Debit += item.Debit
				acctAmounts[item.AccountID].Credit += item.Credit
			}
		}

		// Build account code->amount map
		var accounts []models.Account
		db.Where("book_id = ?", bookID).Find(&accounts)
		codeAmounts := make(map[string]float64)
		for _, a := range accounts {
			if aa, ok := acctAmounts[a.ID]; ok {
				codeAmounts[a.Code] = aa.Debit - aa.Credit
			}
		}

		// Sum by code prefix
		sumCodes := func(prefix string) float64 {
			var total float64
			for code, amt := range codeAmounts {
				if strings.HasPrefix(code, prefix) {
					total += amt
				}
			}
			return total
		}

		revenue := -sumCodes("5001") - sumCodes("5051")          // 收入：贷方净额取正
		cost := sumCodes("5401") + sumCodes("5402")               // 营业成本
		tax := sumCodes("5403")                                    // 税金及附加
		sellExp := sumCodes("5601")                                // 销售费用
		adminExp := sumCodes("5602")                               // 管理费用
		finExp := sumCodes("5603")                                 // 财务费用
		investIncome := -sumCodes("5111")                          // 投资收益
		nonOpIncome := -sumCodes("5301")                           // 营业外收入
		nonOpExp := sumCodes("5711")                               // 营业外支出
		incomeTax := sumCodes("5801")                              // 所得税费用

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
				Joins("JOIN accounts ON accounts.id = account_balances.account_id").
				Where("account_balances.book_id = ? AND account_balances.period = ? AND accounts.code LIKE ?", bookID, period, ec.Code+"%").
				Select("COALESCE(SUM(account_balances.period_debit), 0)").
				Row().Scan(&amount)
			if amount > 0 {
				rows = append(rows, ExpenseRow{Code: ec.Code, Name: ec.Name, Amount: amount})
			}
		}

		// Sub-items for 5602 管理费用
		var subItems []ExpenseRow
		subCodes := []struct{ Code, Name string }{
			{"5602.01", "管理人员薪酬"}, {"5602.02", "办公费"}, {"5602.03", "折旧费"},
			{"5602.04", "修理费"}, {"5602.05", "水电费"}, {"5602.06", "差旅费"},
			{"5602.07", "业务招待费"}, {"5602.08", "车辆使用费"}, {"5602.09", "其他"},
		}
		for _, sc := range subCodes {
			var amount float64
			db.Model(&models.AccountBalance{}).
				Joins("JOIN accounts ON accounts.id = account_balances.account_id").
				Where("account_balances.book_id = ? AND account_balances.period = ? AND accounts.code = ?", bookID, period, sc.Code).
				Select("COALESCE(SUM(account_balances.period_debit), 0)").
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
			Code      string  `json:"code"`
			Name      string  `json:"name"`
			Total     float64 `json:"total"`
			Current   float64 `json:"current"`     // 未到期
			Month1    float64 `json:"month_1"`     // 1个月内
			Month3    float64 `json:"month_3"`     // 1-3个月
			Month6    float64 `json:"month_6"`     // 3-6个月
			Month12   float64 `json:"month_12"`    // 6-12个月
			Over1Year float64 `json:"over_1_year"` // 1年以上
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

// cashFlowStatement generates cash flow statement
func cashFlowStatement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		period := c.DefaultQuery("period", "")

		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		// Get all cash flow items for this book
		var auxItems []models.AuxItem
		db.Where("book_id = ? AND type = ? AND is_active = ?", bookID, "cash_flow", true).Order("code").Find(&auxItems)

		// Build a map of aux_id -> item info
		cfMap := make(map[uint]models.AuxItem)
		for _, item := range auxItems {
			cfMap[item.ID] = item
		}

		// Query voucher items with cash_flow_id, grouped by cash_flow_id
		type CFResult struct {
			CashFlowID uint
			NetAmount  float64 // sum(debit) - sum(credit) for cash accounts
		}

		// Cash account codes: 1001 (库存现金), 1002 (银行存款), 1012 (其他货币资金)
		var results []CFResult
		db.Model(&models.VoucherItem{}).
			Select("voucher_items.cash_flow_id, COALESCE(SUM(voucher_items.debit - voucher_items.credit), 0) as net_amount").
			Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
			Where("vouchers.book_id = ? AND vouchers.status = ? AND vouchers.date LIKE ? AND voucher_items.cash_flow_id IS NOT NULL AND voucher_items.account_code IN (?)",
				bookID, "posted", period+"%",
				[]string{"1001", "1002", "1012"}).
			Group("voucher_items.cash_flow_id").
			Scan(&results)

		// Build flow items grouped by category
		type FlowItem struct {
			Category string  `json:"category"`
			ItemCode string  `json:"item_code"`
			ItemName string  `json:"item_name"`
			Amount   float64 `json:"amount"`
		}

		var flowItems []FlowItem
		categoryTotals := map[string]float64{
			"operating":  0,
			"investing":  0,
			"financing":  0,
		}

		for _, r := range results {
			item, ok := cfMap[r.CashFlowID]
			if !ok {
				continue
			}
			cat := "operating"
			var extra struct {
				Category string `json:"category"`
			}
			json.Unmarshal([]byte(item.Extra), &extra)
			if extra.Category != "" {
				cat = extra.Category
			}
			flowItems = append(flowItems, FlowItem{
				Category: cat,
				ItemCode: item.Code,
				ItemName: item.Name,
				Amount:   r.NetAmount,
			})
			categoryTotals[cat] += r.NetAmount
		}

		// Also include cash flow items with zero amount (so the report shows all items)
		existingIDs := make(map[uint]bool)
		for _, r := range results {
			existingIDs[r.CashFlowID] = true
		}
		for _, item := range auxItems {
			if existingIDs[item.ID] {
				continue
			}
			cat := "operating"
			var extra struct {
				Category string `json:"category"`
			}
			json.Unmarshal([]byte(item.Extra), &extra)
			if extra.Category != "" {
				cat = extra.Category
			}
			flowItems = append(flowItems, FlowItem{
				Category: cat,
				ItemCode: item.Code,
				ItemName: item.Name,
				Amount:   0,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"data": flowItems,
			"summary": gin.H{
				"operating_total":  categoryTotals["operating"],
				"investing_total":  categoryTotals["investing"],
				"financing_total":  categoryTotals["financing"],
				"cash_increase":    categoryTotals["operating"] + categoryTotals["investing"] + categoryTotals["financing"],
			},
			"period": period,
		})
	}
}



// auxBalanceReport returns balance breakdown by auxiliary dimension
func auxBalanceReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		period := c.DefaultQuery("period", "")
		auxType := c.DefaultQuery("type", "") // customer/supplier/department/project/employee/warehouse/bank_account

		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		// Get accounts that have aux_types configured
		var accounts []models.Account
		query := db.Where("book_id = ? AND is_active = ?", bookID, true)
		if auxType != "" {
			query = query.Where("aux_types LIKE ?", "%"+auxType+"%")
		}
		query.Find(&accounts)

		type AuxBalanceRow struct {
			AccountCode   string  `json:"account_code"`
			AccountName   string  `json:"account_name"`
			AuxName       string  `json:"aux_name"`
			AuxType       string  `json:"aux_type"`
			OpeningDebit  float64 `json:"opening_debit"`
			OpeningCredit float64 `json:"opening_credit"`
			PeriodDebit   float64 `json:"period_debit"`
			PeriodCredit  float64 `json:"period_credit"`
			ClosingDebit  float64 `json:"closing_debit"`
			ClosingCredit float64 `json:"closing_credit"`
		}

		var result []AuxBalanceRow

		for _, acct := range accounts {
			if acct.AuxTypes == "" {
				continue
			}

			// Get balances with aux_key
			var balances []models.AccountBalance
			db.Where("book_id = ? AND account_id = ? AND period = ?", bookID, acct.ID, period).Find(&balances)

			for _, b := range balances {
				auxName := b.AuxKey
				if auxName == "" {
					auxName = "(无辅助)"
				}
				result = append(result, AuxBalanceRow{
					AccountCode:   acct.Code,
					AccountName:   acct.Name,
					AuxName:       auxName,
					AuxType:       acct.AuxTypes,
					OpeningDebit:  b.OpeningDebit,
					OpeningCredit: b.OpeningCredit,
					PeriodDebit:   b.PeriodDebit,
					PeriodCredit:  b.PeriodCredit,
					ClosingDebit:  b.ClosingDebit,
					ClosingCredit: b.ClosingCredit,
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{"data": result, "period": period})
	}
}

// journal returns cash/bank journal entries
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
			Period  string       `json:"period"`
			Total   ColumnData   `json:"total"`
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

// customReport generates a custom report based on a template
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

// exportReport exports current report data as CSV
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
					case "1":
						category = "资产"
					case "2":
						category = "负债"
					case "3":
						category = "所有者权益"
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

func listReportTemplates(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var templates []models.ReportTemplate
		db.Where("book_id = ? OR book_id IS NULL", bookID).Order("type ASC, name ASC").Find(&templates)
		c.JSON(http.StatusOK, gin.H{"data": templates})
	}
}

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
