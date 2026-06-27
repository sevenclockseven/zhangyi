package api

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/sevenclockseven/zhangyi/internal/models"
	"gorm.io/gorm"
)

// Template directory - can be overridden by env var
func templateDir() string {
	if d := os.Getenv("TEMPLATE_DIR"); d != "" {
		return d
	}
	return "templates"
}

func generateID(db *gorm.DB) uint {
	var maxCode string
	db.Model(&models.AccountBook{}).Select("COALESCE(MAX(code), 'BK000000')").Row().Scan(&maxCode)
	// Extract number from BK000001 format
	num := 0
	if len(maxCode) > 2 {
		fmt.Sscanf(maxCode[2:], "%d", &num)
	}
	return uint(num + 1)
}

func generateVoucherNumber(db *gorm.DB, bookID uint, date string) string {
	// Format: 记-YYYY-MM-001
	parts := strings.Split(date, "-")
	period := parts[0] + "-" + parts[1] // e.g. "2026-06"
	// Also match old format without hyphen: 记-YYYYMM-001
	oldPeriod := parts[0] + parts[1] // e.g. "202606"

	// Get max number from both formats
	var maxNew, maxOld string
	db.Model(&models.Voucher{}).
		Where("book_id = ? AND number LIKE ?", bookID, "记-"+period+"-%").
		Select("COALESCE(MAX(number), '')").
		Row().Scan(&maxNew)
	db.Model(&models.Voucher{}).
		Where("book_id = ? AND number LIKE ?", bookID, "记-"+oldPeriod+"-%").
		Select("COALESCE(MAX(number), '')").
		Row().Scan(&maxOld)

	// Use the higher sequence
	maxSeq := 0
	for _, num := range []string{maxNew, maxOld} {
		if num == "" {
			continue
		}
		// Extract last segment after final hyphen
		idx := strings.LastIndex(num, "-")
		if idx < 0 {
			continue
		}
		seq := 0
		fmt.Sscanf(num[idx+1:], "%d", &seq)
		if seq > maxSeq {
			maxSeq = seq
		}
	}

	return fmt.Sprintf("记-%s-%03d", period, maxSeq+1)
}

// AccountTree represents a tree node
type AccountTree struct {
	models.Account
	Children []AccountTree `json:"children,omitempty"`
}

func buildAccountTree(accounts []models.Account, parentCode string) []AccountTree {
	var result []AccountTree
	for _, a := range accounts {
		if a.ParentCode == parentCode {
			node := AccountTree{
				Account:  a,
				Children: buildAccountTree(accounts, a.Code),
			}
			result = append(result, node)
		}
	}
	return result
}

// updateAccountBalances updates account balances when a voucher is posted
func updateAccountBalances(db *gorm.DB, voucher *models.Voucher) error {
	var items []models.VoucherItem
	db.Where("voucher_id = ?", voucher.ID).Find(&items)

	// Determine period from voucher date
	parts := strings.Split(voucher.Date, "-")
	period := parts[0] + "-" + parts[1]

	for _, item := range items {
		// Find or create balance record
		var balance models.AccountBalance
		result := db.Where("book_id = ? AND account_id = ? AND period = ? AND aux_key = ?",
			voucher.BookID, item.AccountID, period, "").
			First(&balance)

		if result.Error == gorm.ErrRecordNotFound {
			balance = models.AccountBalance{
				BookID:    voucher.BookID,
				AccountID: item.AccountID,
				Period:    period,
				AuxKey:    "",
			}
			db.Create(&balance)
		}

		// Update period amounts
		db.Model(&balance).Updates(map[string]interface{}{
			"period_debit":   gorm.Expr("period_debit + ?", item.Debit),
			"period_credit":  gorm.Expr("period_credit + ?", item.Credit),
			"ytd_debit":      gorm.Expr("ytd_debit + ?", item.Debit),
			"ytd_credit":     gorm.Expr("ytd_credit + ?", item.Credit),
			"closing_debit":  gorm.Expr("closing_debit + ?", item.Debit),
			"closing_credit": gorm.Expr("closing_credit + ?", item.Credit),
		})
	}

	return nil
}

// reverseAccountBalances reverses account balances when a voucher is unposted
func reverseAccountBalances(db *gorm.DB, voucher *models.Voucher) error {
	var items []models.VoucherItem
	db.Where("voucher_id = ?", voucher.ID).Find(&items)

	parts := strings.Split(voucher.Date, "-")
	period := parts[0] + "-" + parts[1]

	for _, item := range items {
		var balance models.AccountBalance
		if err := db.Where("book_id = ? AND account_id = ? AND period = ? AND aux_key = ?",
			voucher.BookID, item.AccountID, period, "").
			First(&balance).Error; err != nil {
			continue
		}

		db.Model(&balance).Updates(map[string]interface{}{
			"period_debit":   gorm.Expr("period_debit - ?", item.Debit),
			"period_credit":  gorm.Expr("period_credit - ?", item.Credit),
			"ytd_debit":      gorm.Expr("ytd_debit - ?", item.Debit),
			"ytd_credit":     gorm.Expr("ytd_credit - ?", item.Credit),
			"closing_debit":  gorm.Expr("closing_debit - ?", item.Debit),
			"closing_credit": gorm.Expr("closing_credit - ?", item.Credit),
		})
	}

	return nil
}

func parseExtra(extraStr string) map[string]string {
	result := make(map[string]string)
	if extraStr == "" {
		return result
	}
	json.Unmarshal([]byte(extraStr), &result)
	return result
}

func quoteCSV(s string) string {
	if s == "" {
		return ""
	}
	if strings.ContainsAny(s, ",\"\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

func boolStatus(b bool) string {
	if b {
		return "启用"
	}
	return "停用"
}

func parseCSVLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuote := false
	for _, r := range line {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == ',' && !inQuote:
			fields = append(fields, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	fields = append(fields, strings.TrimSpace(current.String()))
	return fields
}

// evalFormula evaluates a report formula like JE('6602', '借') or QM('1002', '借')
func evalFormula(db *gorm.DB, bookID uint, period string, formula string) float64 {
	// Parse formula: FUNC('code', 'direction')
	formula = strings.TrimSpace(formula)

	// Handle simple arithmetic: formula1 - formula2 or formula1 + formula2
	if strings.Contains(formula, " - ") {
		parts := strings.SplitN(formula, " - ", 2)
		return evalFormula(db, bookID, period, parts[0]) - evalFormula(db, bookID, period, parts[1])
	}
	if strings.Contains(formula, " + ") {
		parts := strings.SplitN(formula, " + ", 2)
		return evalFormula(db, bookID, period, parts[0]) + evalFormula(db, bookID, period, parts[1])
	}

	// Parse function call: FUNC('code', 'direction')
	re := regexp.MustCompile(`(\w+)\('([^']+)'\s*,\s*'([^']+)'\)`)
	matches := re.FindStringSubmatch(formula)
	if len(matches) < 4 {
		return 0
	}

	funcName := matches[1]
	code := matches[2]
	direction := matches[3]

	switch funcName {
	case "JE": // 本期发生额
		var total float64
		db.Model(&models.AccountBalance{}).
			Joins("JOIN accounts ON accounts.id = account_balances.account_id").
			Where("account_balances.book_id = ? AND account_balances.period = ? AND accounts.code LIKE ?", bookID, period, code+"%").
			Select("COALESCE(SUM(account_balances.period_debit), 0) - COALESCE(SUM(account_balances.period_credit), 0)").
			Row().Scan(&total)
		if direction == "credit" {
			total = -total
		}
		return total

	case "QM": // 期末余额
		var total float64
		db.Model(&models.AccountBalance{}).
			Joins("JOIN accounts ON accounts.id = account_balances.account_id").
			Where("account_balances.book_id = ? AND account_balances.period = ? AND accounts.code LIKE ?", bookID, period, code+"%").
			Select("COALESCE(SUM(account_balances.closing_debit), 0) - COALESCE(SUM(account_balances.closing_credit), 0)").
			Row().Scan(&total)
		if direction == "credit" {
			total = -total
		}
		return total

	case "QC": // 期初余额
		var total float64
		db.Model(&models.AccountBalance{}).
			Joins("JOIN accounts ON accounts.id = account_balances.account_id").
			Where("account_balances.book_id = ? AND account_balances.period = ? AND accounts.code LIKE ?", bookID, period, code+"%").
			Select("COALESCE(SUM(account_balances.opening_debit), 0) - COALESCE(SUM(account_balances.opening_credit), 0)").
			Row().Scan(&total)
		if direction == "credit" {
			total = -total
		}
		return total

	case "JL": // 本年累计发生额
		// Get all periods up to current
		yearPrefix := period[:4]
		var total float64
		db.Model(&models.AccountBalance{}).
			Joins("JOIN accounts ON accounts.id = account_balances.account_id").
			Where("account_balances.book_id = ? AND account_balances.period LIKE ? AND accounts.code LIKE ?", bookID, yearPrefix+"%", code+"%").
			Select("COALESCE(SUM(account_balances.period_debit), 0) - COALESCE(SUM(account_balances.period_credit), 0)").
			Row().Scan(&total)
		if direction == "credit" {
			total = -total
		}
		return total
	}

	return 0
}
