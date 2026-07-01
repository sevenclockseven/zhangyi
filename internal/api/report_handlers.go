package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
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

		c.JSON(http.StatusOK, gin.H{
			"period":      period,
			"assets":      assets,
			"liabilities": liabilities,
			"equity":      equity,
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

		// 1. 查询所有科目和余额
		var accounts []models.Account
		db.Where("book_id = ?", bookID).Order("code ASC").Find(&accounts)

		var balances []models.AccountBalance
		db.Where("book_id = ? AND period = ?", bookID, period).Find(&balances)

		// 2. 构建 balance map: account_id -> balance
		balanceMap := make(map[uint]models.AccountBalance)
		for _, b := range balances {
			balanceMap[b.AccountID] = b
		}

		// 3. 构建科目节点 map: code -> tree node
		nodeMap := make(map[string]*gin.H)
		for _, acct := range accounts {
			b, hasBalance := balanceMap[acct.ID]
			node := gin.H{
				"account_code":   acct.Code,
				"account_name":   acct.Name,
				"direction":      acct.Direction,
				"level":          acct.Level,
				"parent_code":    acct.ParentCode,
				"opening_debit":  0.0,
				"opening_credit": 0.0,
				"period_debit":   0.0,
				"period_credit":  0.0,
				"closing_debit":  0.0,
				"closing_credit": 0.0,
				"children":       []gin.H{},
			}
			if hasBalance {
				node["opening_debit"] = b.OpeningDebit
				node["opening_credit"] = b.OpeningCredit
				node["period_debit"] = b.PeriodDebit
				node["period_credit"] = b.PeriodCredit
				node["closing_debit"] = b.ClosingDebit
				node["closing_credit"] = b.ClosingCredit
			}
			nodeMap[acct.Code] = &node
		}

		// 4. 组装树：将子节点挂到父节点
		roots := []gin.H{}
		for code, node := range nodeMap {
			parentCode := (*node)["parent_code"].(string)
			if parentCode == "" {
				roots = append(roots, *node)
			} else if parent, ok := nodeMap[parentCode]; ok {
				children := (*parent)["children"].([]gin.H)
				children = append(children, *node)
				(*parent)["children"] = children
			} else {
				// 父节点不存在，作为根节点
				roots = append(roots, *node)
			}
			_ = code
		}

		// 5. 自底向上汇总：父节点金额 = 子节点合计
		var sumUp func(node *gin.H)
		sumUp = func(node *gin.H) {
			children := (*node)["children"].([]gin.H)
			if len(children) == 0 {
				return
			}
			for i := range children {
				sumUp(&children[i])
			}
			var sOpeningDebit, sOpeningCredit, sPeriodDebit, sPeriodCredit, sClosingDebit, sClosingCredit float64
			for _, ch := range children {
				sOpeningDebit += ch["opening_debit"].(float64)
				sOpeningCredit += ch["opening_credit"].(float64)
				sPeriodDebit += ch["period_debit"].(float64)
				sPeriodCredit += ch["period_credit"].(float64)
				sClosingDebit += ch["closing_debit"].(float64)
				sClosingCredit += ch["closing_credit"].(float64)
			}
			(*node)["opening_debit"] = sOpeningDebit
			(*node)["opening_credit"] = sOpeningCredit
			(*node)["period_debit"] = sPeriodDebit
			(*node)["period_credit"] = sPeriodCredit
			(*node)["closing_debit"] = sClosingDebit
			(*node)["closing_credit"] = sClosingCredit
		}

		for i := range roots {
			sumUp(&roots[i])
		}

		// 6. 清理 children 为空的非叶子节点（保留字段让 el-table 识别）
		// el-table 树形模式需要 children 字段存在即可
		// 7. 排序：根节点按编码排序，子节点也递归排序
		var sortTree func(nodes []gin.H)
		sortTree = func(nodes []gin.H) {
			sort.Slice(nodes, func(i, j int) bool {
				return nodes[i]["account_code"].(string) < nodes[j]["account_code"].(string)
			})
			for i := range nodes {
				children := nodes[i]["children"].([]gin.H)
				if len(children) > 1 {
					sortTree(children)
				}
			}
		}
		sortTree(roots)


		c.JSON(http.StatusOK, gin.H{"data": roots, "period": period})
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

		getAmount := func(code string, getDebit bool) float64 {
			var total float64
			subQuery := db.Model(&models.Account{}).Select("id").Where("book_id = ? AND code LIKE ?", uint(bookID), code+"%")
			if getDebit {
				db.Model(&models.AccountBalance{}).
					Where("book_id = ? AND period = ? AND account_id IN (?)", uint(bookID), period, subQuery).
					Select("COALESCE(SUM(period_debit), 0)").
					Row().Scan(&total)
			} else {
				db.Model(&models.AccountBalance{}).
					Where("book_id = ? AND period = ? AND account_id IN (?)", uint(bookID), period, subQuery).
					Select("COALESCE(SUM(period_credit), 0)").
					Row().Scan(&total)
			}
			return total
		}

		revenue := getAmount("5001", false) + getAmount("5051", false)   // 营业收入 = 主营+其他
		cost := getAmount("5401", true) + getAmount("5402", true)         // 营业成本
		tax := getAmount("5403", true)                                       // 税金及附加
		sellExp := getAmount("5601", true)                                   // 销售费用
		adminExp := getAmount("5602", true)                                  // 管理费用
		finExp := getAmount("5603", true)                                    // 财务费用
		investIncome := getAmount("5111", false)                             // 投资收益
		nonOpIncome := getAmount("5301", false)                              // 营业外收入
		nonOpExp := getAmount("5711", true)                                  // 营业外支出
		incomeTax := getAmount("5801", true)                                 // 所得税费用

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
			expSub := db.Model(&models.Account{}).Select("id").Where("book_id = ? AND code LIKE ?", uint(bookID), ec.Code+"%")
			db.Model(&models.AccountBalance{}).
				Where("book_id = ? AND period = ? AND account_id IN (?)", uint(bookID), period, expSub).
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
			expSub2 := db.Model(&models.Account{}).Select("id").Where("book_id = ? AND code = ?", uint(bookID), sc.Code)
			db.Model(&models.AccountBalance{}).
				Where("book_id = ? AND period = ? AND account_id IN (?)", uint(bookID), period, expSub2).
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
		now := time.Now()

		type AgingRow struct {
			Code      string  `json:"code"`
			Name      string  `json:"name"`
			Total     float64 `json:"total"`
			Current   float64 `json:"current"`     // 30天内
			Month1    float64 `json:"month_1"`     // 1-3个月
			Month3    float64 `json:"month_3"`     // 3-6个月
			Month6    float64 `json:"month_6"`     // 6-12个月
			Month12   float64 `json:"month_12"`    // 12-24个月
			Over1Year float64 `json:"over_1_year"` // 2年以上
		}

		var accountCodes []string
		auxType := "customer"
		if reportType == "ar" {
			accountCodes = []string{"1122", "2203"} // 应收账款, 预收账款
		} else {
			accountCodes = []string{"2202", "1123"} // 应付账款, 预付账款
			auxType = "supplier"
		}

		// 收集所有辅助项的账龄数据
		type AgingEntry struct {
			AuxID   uint
			AuxCode string
			AuxName string
			Date    string
			Amount  float64 // 正数=应收/应付余额
		}
		var entries []AgingEntry

		for _, code := range accountCodes {
			var auxItems []models.AuxItem
			db.Where("book_id = ? AND type = ? AND is_active = ?", bookID, auxType, true).Find(&auxItems)

			for _, aux := range auxItems {
				// 查询该辅助项下所有已记账的凭证明细
				type VoucherDetail struct {
					Date   string
					Debit  float64
					Credit float64
				}
				var details []VoucherDetail
				db.Model(&models.VoucherItem{}).
					Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
					Where("vouchers.book_id = ? AND vouchers.status = ? AND voucher_items.account_code LIKE ? AND voucher_items.aux_"+auxType+"_id = ?",
						bookID, "posted", code+"%", aux.ID).
					Select("vouchers.date, voucher_items.debit, voucher_items.credit").
					Order("vouchers.date ASC").
					Find(&details)

				// 逐笔计算每条记录对账龄的贡献
				for _, d := range details {
					var amount float64
					if reportType == "ar" {
						amount = d.Debit - d.Credit // 应收：借方-贷方
					} else {
						amount = d.Credit - d.Debit // 应付：贷方-借方
					}
					if amount <= 0 {
						continue // 负数表示已结算，不计入账龄
					}
					entries = append(entries, AgingEntry{
						AuxID:   aux.ID,
						AuxCode: aux.Code,
						AuxName: aux.Name,
						Date:    d.Date,
						Amount:  amount,
					})
				}
			}
		}

		// 按辅助项聚合，逐笔归入账龄桶
		auxMap := make(map[uint]*AgingRow)
		for _, e := range entries {
			row, exists := auxMap[e.AuxID]
			if !exists {
				row = &AgingRow{Code: e.AuxCode, Name: e.AuxName}
				auxMap[e.AuxID] = row
			}
			row.Total += e.Amount

			// 解析日期
			entryDate, err := time.Parse("2006-01-02", e.Date)
			if err != nil {
				entryDate, err = time.Parse("2006-01", e.Date)
				if err != nil {
					row.Current += e.Amount
					continue
				}
			}
			ageDays := int(now.Sub(entryDate).Hours() / 24)

			switch {
			case ageDays <= 30:
				row.Current += e.Amount
			case ageDays <= 90:
				row.Month1 += e.Amount
			case ageDays <= 180:
				row.Month3 += e.Amount
			case ageDays <= 365:
				row.Month6 += e.Amount
			case ageDays <= 730:
				row.Month12 += e.Amount
			default:
				row.Over1Year += e.Amount
			}
		}

		// 转为数组并四舍五入
		var rows []AgingRow
		for _, row := range auxMap {
			row.Total = math.Round(row.Total*100) / 100
			row.Current = math.Round(row.Current*100) / 100
			row.Month1 = math.Round(row.Month1*100) / 100
			row.Month3 = math.Round(row.Month3*100) / 100
			row.Month6 = math.Round(row.Month6*100) / 100
			row.Month12 = math.Round(row.Month12*100) / 100
			row.Over1Year = math.Round(row.Over1Year*100) / 100
			if row.Total > 0 {
				rows = append(rows, *row)
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

		// Cash account codes: 1001 (库存现金), 1002 (银行存款), 1012 (其他货币资金)
		cashCodes := []string{"1001", "1002", "1012"}

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
			NetAmount  float64
		}

		var results []CFResult
		db.Model(&models.VoucherItem{}).
			Select("voucher_items.cash_flow_id, COALESCE(SUM(voucher_items.debit - voucher_items.credit), 0) as net_amount").
			Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
			Where("vouchers.book_id = ? AND vouchers.status = ? AND vouchers.date LIKE ? AND voucher_items.cash_flow_id IS NOT NULL AND voucher_items.account_code IN (?)",
				bookID, "posted", period+"%", cashCodes).
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
			"operating": 0,
			"investing": 0,
			"financing": 0,
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

		reportIncrease := categoryTotals["operating"] + categoryTotals["investing"] + categoryTotals["financing"]

		// === P0: 未标记凭证检测 ===
		type UntaggedItem struct {
			VoucherID   uint    `json:"voucher_id"`
			VoucherDate string  `json:"voucher_date"`
			VoucherNo   string  `json:"voucher_no"`
			AccountCode string  `json:"account_code"`
			AccountName string  `json:"account_name"`
			Debit       float64 `json:"debit"`
			Credit      float64 `json:"credit"`
			Memo        string  `json:"memo"`
		}
		var untagged []UntaggedItem
		db.Model(&models.VoucherItem{}).
			Select("voucher_items.voucher_id, vouchers.date as voucher_date, vouchers.no as voucher_no, "+
				"voucher_items.account_code, voucher_items.account_name, "+
				"voucher_items.debit, voucher_items.credit, voucher_items.memo").
			Joins("JOIN vouchers ON vouchers.id = voucher_items.voucher_id").
			Where("vouchers.book_id = ? AND vouchers.status = ? AND vouchers.date LIKE ? "+
				"AND voucher_items.account_code IN (?) AND voucher_items.cash_flow_id IS NULL",
				bookID, "posted", period+"%", cashCodes).
			Order("vouchers.date").
			Scan(&untagged)

		// === P1+P2: 期初/期末现金余额 + 勾稽校验 ===
		// Get account IDs for cash codes
		var cashAccountIDs []uint
		db.Model(&models.Account{}).
			Select("id").
			Where("book_id = ? AND code IN (?)", bookID, cashCodes).
			Scan(&cashAccountIDs)

		type BalanceResult struct {
			OpeningDebit  float64
			OpeningCredit float64
			ClosingDebit  float64
			ClosingCredit float64
		}

		var currentBalance BalanceResult
		if len(cashAccountIDs) > 0 {
			db.Model(&models.AccountBalance{}).
				Select("SUM(opening_debit) as opening_debit, SUM(opening_credit) as opening_credit, "+
					"SUM(closing_debit) as closing_debit, SUM(closing_credit) as closing_credit").
				Where("book_id = ? AND period = ? AND account_id IN (?)", bookID, period, cashAccountIDs).
				Scan(&currentBalance)
		}

		openingCash := currentBalance.OpeningDebit - currentBalance.OpeningCredit
		closingCash := currentBalance.ClosingDebit - currentBalance.ClosingCredit
		actualIncrease := closingCash - openingCash
		reconciled := math.Abs(actualIncrease-reportIncrease) < 0.01

		c.JSON(http.StatusOK, gin.H{
			"data": flowItems,
			"summary": gin.H{
				"operating_total": categoryTotals["operating"],
				"investing_total": categoryTotals["investing"],
				"financing_total": categoryTotals["financing"],
				"cash_increase":   reportIncrease,
			},
			"balance": gin.H{
				"opening_cash":   openingCash,
				"closing_cash":   closingCash,
				"actual_increase": actualIncrease,
				"reconciled":     reconciled,
			},
			"warnings": gin.H{
				"untagged_count": len(untagged),
				"untagged_items": untagged,
			},
			"period": period,
		})
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

// monthlyTrend returns monthly revenue/expense/profit for a given year
func monthlyTrend(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		year := c.DefaultQuery("year", fmt.Sprintf("%d", time.Now().Year()))

		// 初始化12个月
		months := make([]string, 12)
		revenue := make([]float64, 12)
		expense := make([]float64, 12)

		for i := 0; i < 12; i++ {
			months[i] = fmt.Sprintf("%s-%02d", year, i+1)
		}

		// 查询该年所有已记账的科目余额
		type BalanceRow struct {
			Period       string
			AccountCode  string
			PeriodDebit  float64
			PeriodCredit float64
		}
		var rows []BalanceRow
		db.Model(&models.AccountBalance{}).
			Select("account_balances.period, accounts.code as account_code, account_balances.period_debit, account_balances.period_credit").
			Joins("JOIN accounts ON accounts.id = account_balances.account_id").
			Where("account_balances.book_id = ? AND account_balances.period LIKE ? AND accounts.is_active = ?",
				bookID, year+"%", true).
			Find(&rows)

		// 按月汇总
		for _, r := range rows {
			monthIdx := -1
			for i, m := range months {
				if r.Period == m {
					monthIdx = i
					break
				}
			}
			if monthIdx < 0 {
				continue
			}

			code := r.AccountCode
			if len(code) < 4 {
				continue
			}

			switch {
			case code >= "5000" && code <= "5399":
				// 收入类：贷方发生额 - 借方发生额
				revenue[monthIdx] += r.PeriodCredit - r.PeriodDebit
			case code >= "5400" && code <= "5999":
				// 费用类：借方发生额 - 贷方发生额
				expense[monthIdx] += r.PeriodDebit - r.PeriodCredit
			}
		}

		// 计算利润
		profit := make([]float64, 12)
		for i := 0; i < 12; i++ {
			profit[i] = math.Round((revenue[i]-expense[i])*100) / 100
			revenue[i] = math.Round(revenue[i]*100) / 100
			expense[i] = math.Round(expense[i]*100) / 100
		}

		// 费用构成：当年各一级费用科目的合计
		type ExpenseBreakdown struct {
			AccountName string  `json:"name"`
			Amount      float64 `json:"value"`
		}
		var breakdown []ExpenseBreakdown
		db.Model(&models.AccountBalance{}).
			Select("accounts.name as account_name, SUM(account_balances.period_debit - account_balances.period_credit) as amount").
			Joins("JOIN accounts ON accounts.id = account_balances.account_id").
			Where("account_balances.book_id = ? AND account_balances.period LIKE ? AND accounts.code >= ? AND accounts.code <= ? AND accounts.is_active = ? AND accounts.level = 1",
				bookID, year+"%", "5400", "5999", true).
			Group("accounts.name").
			Having("SUM(account_balances.period_debit - account_balances.period_credit) > 0").
			Order("amount DESC").
			Find(&breakdown)

		// 四舍五入
		for i := range breakdown {
			breakdown[i].Amount = math.Round(breakdown[i].Amount*100) / 100
		}

		c.JSON(http.StatusOK, gin.H{
			"year":             year,
			"months":           months,
			"revenue":          revenue,
			"expense":          expense,
			"profit":           profit,
			"expense_breakdown": breakdown,
		})
	}
}
