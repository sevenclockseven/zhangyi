package api

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/models"
	"github.com/sevenclockseven/zhangyi/internal/services"
)

// Template directory - can be overridden by env var
func templateDir() string {
	if d := os.Getenv("TEMPLATE_DIR"); d != "" {
		return d
	}
	return "templates"
}

// ===== Account Books =====

func listBooks(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var books []models.AccountBook
		if err := db.Order("created_at DESC").Find(&books).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": books})
	}
}

func createBook(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name         string   `json:"name" binding:"required"`
			Code         string   `json:"code"`
			Industry     []string `json:"industry" binding:"required"`
			TaxpayerType string   `json:"taxpayer_type"`
			StartDate    string   `json:"start_date" binding:"required"`
			Contact      string   `json:"contact"`
			Phone        string   `json:"phone"`
			Address      string   `json:"address"`
			Memo         string   `json:"memo"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Auto generate code if not provided
		code := req.Code
		if code == "" {
			code = fmt.Sprintf("BK%06d", generateID(db))
		}

		book := models.AccountBook{
			Code:         code,
			Name:         req.Name,
			Industry:     strings.Join(req.Industry, ","),
			TaxpayerType: req.TaxpayerType,
			StartDate:    req.StartDate,
			Currency:     "CNY",
			Status:       "active",
			Contact:      req.Contact,
			Phone:        req.Phone,
			Address:      req.Address,
			Memo:         req.Memo,
		}

		// Begin transaction
		tx := db.Begin()

		if err := tx.Create(&book).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Load and apply template
		if err := services.ApplyTemplateToBook(tx, book.ID, templateDir(), req.Industry); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "加载科目模板失败: " + err.Error()})
			return
		}

		tx.Commit()

		c.JSON(http.StatusCreated, gin.H{"data": book})
	}
}

func getBook(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var book models.AccountBook
		if err := db.First(&book, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "账套不存在"})
			return
		}

		// Count accounts and vouchers
		var accountCount, voucherCount int64
		db.Model(&models.Account{}).Where("book_id = ?", book.ID).Count(&accountCount)
		db.Model(&models.Voucher{}).Where("book_id = ?", book.ID).Count(&voucherCount)

		c.JSON(http.StatusOK, gin.H{
			"data": book,
			"meta": gin.H{
				"account_count": accountCount,
				"voucher_count": voucherCount,
			},
		})
	}
}

func updateBook(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var book models.AccountBook
		if err := db.First(&book, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "账套不存在"})
			return
		}

		var req struct {
			Name         string `json:"name"`
			TaxpayerType string `json:"taxpayer_type"`
			Contact      string `json:"contact"`
			Phone        string `json:"phone"`
			Address      string `json:"address"`
			Memo         string `json:"memo"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updates := map[string]interface{}{}
		if req.Name != "" {
			updates["name"] = req.Name
		}
		if req.TaxpayerType != "" {
			updates["taxpayer_type"] = req.TaxpayerType
		}
		if req.Contact != "" {
			updates["contact"] = req.Contact
		}
		if req.Phone != "" {
			updates["phone"] = req.Phone
		}
		if req.Address != "" {
			updates["address"] = req.Address
		}
		if req.Memo != "" {
			updates["memo"] = req.Memo
		}

		db.Model(&book).Updates(updates)
		c.JSON(http.StatusOK, gin.H{"data": book})
	}
}

func deleteBook(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		tx := db.Begin()
		// Delete related data
		tx.Where("book_id = ?", id).Delete(&models.AccountBalance{})
		tx.Where("book_id = ?", id).Delete(&models.OpeningBalance{})
		tx.Where("book_id = ?", id).Delete(&models.AuxItem{})
		tx.Where("book_id = ?", id).Delete(&models.ReportTemplate{})
		tx.Where("book_id = ?", id).Delete(&models.VoucherTemplate{})
		tx.Where("book_id = ?", id).Delete(&models.OperationLog{})

		// Delete voucher items first
		var voucherIDs []uint
		tx.Model(&models.Voucher{}).Where("book_id = ?", id).Pluck("id", &voucherIDs)
		if len(voucherIDs) > 0 {
			tx.Where("voucher_id IN ?", voucherIDs).Delete(&models.VoucherItem{})
		}
		tx.Where("book_id = ?", id).Delete(&models.Voucher{})
		tx.Where("book_id = ?", id).Delete(&models.Account{})
		tx.Where("id = ?", id).Delete(&models.AccountBook{})

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
	}
}

// ===== Accounts =====

func listAccounts(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var accounts []models.Account
		query := db.Where("book_id = ?", bookID)

		// Optional filters
		if level := c.Query("level"); level != "" {
			query = query.Where("level = ?", level)
		}
		if active := c.Query("active"); active != "" {
			query = query.Where("is_active = ?", active == "true")
		}

		query.Order("code").Find(&accounts)
		c.JSON(http.StatusOK, gin.H{"data": accounts})
	}
}

func getAccountTree(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var accounts []models.Account
		db.Where("book_id = ?", bookID).Order("code").Find(&accounts)

		// Build tree
		tree := buildAccountTree(accounts, "")
		c.JSON(http.StatusOK, gin.H{"data": tree})
	}
}

func createAccount(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var account models.Account
		if err := c.ShouldBindJSON(&account); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		bid, _ := strconv.ParseUint(bookID, 10, 64)
		account.BookID = uint(bid)
		account.IsSystem = false

		// Auto detect level and parent
		if account.Level == 0 {
			account.Level = len(strings.Split(account.Code, "."))
		}
		if account.ParentCode == "" && account.Level > 1 {
			parts := strings.Split(account.Code, ".")
			account.ParentCode = strings.Join(parts[:len(parts)-1], ".")
		}

		if err := db.Create(&account).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Update parent's is_leaf to false
		if account.ParentCode != "" {
			db.Model(&models.Account{}).
				Where("book_id = ? AND code = ?", account.BookID, account.ParentCode).
				Update("is_leaf", false)
		}

		c.JSON(http.StatusCreated, gin.H{"data": account})
	}
}

func updateAccount(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		acid := c.Param("acid")
		var account models.Account
		if err := db.First(&account, acid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "科目不存在"})
			return
		}

		var req struct {
			Name     string `json:"name"`
			IsActive *bool  `json:"is_active"`
			AuxTypes string `json:"aux_types"`
			Memo     string `json:"memo"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updates := map[string]interface{}{}
		if req.Name != "" {
			updates["name"] = req.Name
		}
		if req.IsActive != nil {
			updates["is_active"] = *req.IsActive
		}
		if req.AuxTypes != "" {
			updates["aux_types"] = req.AuxTypes
		}
		if req.Memo != "" {
			updates["memo"] = req.Memo
		}

		db.Model(&account).Updates(updates)
		c.JSON(http.StatusOK, gin.H{"data": account})
	}
}

func deleteAccount(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		acid := c.Param("acid")
		var account models.Account
		if err := db.First(&account, acid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "科目不存在"})
			return
		}

		// Check if has voucher references
		var count int64
		db.Model(&models.VoucherItem{}).Where("account_id = ?", account.ID).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "该科目已有凭证引用，不能删除，可以停用"})
			return
		}

		// Check if has children
		db.Model(&models.Account{}).Where("parent_code = ? AND book_id = ?", account.Code, account.BookID).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "该科目有下级科目，不能删除"})
			return
		}

		db.Delete(&account)

		// Update parent's is_leaf if no more children
		if account.ParentCode != "" {
			db.Model(&models.Account{}).
				Where("book_id = ? AND code = ?", account.BookID, account.ParentCode).
				Update("is_leaf", true)
		}

		c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
	}
}

func syncTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var book models.AccountBook
		if err := db.First(&book, bookID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "账套不存在"})
			return
		}

		industries := strings.Split(book.Industry, ",")
		if err := services.SyncTemplateUpdates(db, book.ID, templateDir(), industries); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "同步模板失败: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "模板同步成功"})
	}
}

// ===== Vouchers =====

func listVouchers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var vouchers []models.Voucher
		query := db.Where("book_id = ?", bookID)

		if status := c.Query("status"); status != "" {
			query = query.Where("status = ?", status)
		}
		if dateFrom := c.Query("date_from"); dateFrom != "" {
			query = query.Where("date >= ?", dateFrom)
		}
		if dateTo := c.Query("date_to"); dateTo != "" {
			query = query.Where("date <= ?", dateTo)
		}

		query.Order("date DESC, number DESC").Find(&vouchers)
		c.JSON(http.StatusOK, gin.H{"data": vouchers})
	}
}

func createVoucher(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var req struct {
			Date        string `json:"date" binding:"required"`
			VoucherType string `json:"voucher_type"`
			Attachments int    `json:"attachments"`
			Memo        string `json:"memo"`
			Items       []struct {
				AccountID    uint    `json:"account_id"`
				AccountCode  string  `json:"account_code"`
				AccountName  string  `json:"account_name"`
				Debit        float64 `json:"debit"`
				Credit       float64 `json:"credit"`
				Memo         string  `json:"memo"`
				AuxCustomer  *uint   `json:"aux_customer_id"`
				AuxSupplier  *uint   `json:"aux_supplier_id"`
				AuxDept      *uint   `json:"aux_department_id"`
				AuxProject   *uint   `json:"aux_project_id"`
				AuxEmployee  *uint   `json:"aux_employee_id"`
				AuxWarehouse *uint   `json:"aux_warehouse_id"`
				AuxBank      *uint   `json:"aux_bank_account_id"`
				CashFlow     *uint   `json:"cash_flow_id"`
			} `json:"items" binding:"required,min=2"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		bid, _ := strconv.ParseUint(bookID, 10, 64)

		// Calculate totals
		var totalDebit, totalCredit float64
		for _, item := range req.Items {
			totalDebit += item.Debit
			totalCredit += item.Credit
		}

		// Validate balance
		if fmt.Sprintf("%.2f", totalDebit) != fmt.Sprintf("%.2f", totalCredit) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("借贷不平衡：借方 %.2f ≠ 贷方 %.2f", totalDebit, totalCredit)})
			return
		}

		// Generate voucher number
		number := generateVoucherNumber(db, uint(bid), req.Date)

		voucher := models.Voucher{
			BookID:      uint(bid),
			Date:        req.Date,
			Number:      number,
			VoucherType: req.VoucherType,
			Status:      "draft",
			TotalDebit:  totalDebit,
			TotalCredit: totalCredit,
			Attachments: req.Attachments,
			Memo:        req.Memo,
		}

		tx := db.Begin()

		if err := tx.Create(&voucher).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Create voucher items
		for i, item := range req.Items {
			vi := models.VoucherItem{
				VoucherID:        voucher.ID,
				LineNo:           i + 1,
				AccountID:        item.AccountID,
				AccountCode:      item.AccountCode,
				AccountName:      item.AccountName,
				Debit:            item.Debit,
				Credit:           item.Credit,
				Memo:             item.Memo,
				AuxCustomerID:    item.AuxCustomer,
				AuxSupplierID:    item.AuxSupplier,
				AuxDepartmentID:  item.AuxDept,
				AuxProjectID:     item.AuxProject,
				AuxEmployeeID:    item.AuxEmployee,
				AuxWarehouseID:   item.AuxWarehouse,
				AuxBankAccountID: item.AuxBank,
				CashFlowID:       item.CashFlow,
			}
			if err := tx.Create(&vi).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		tx.Commit()

		// Reload with items
		db.Preload("Items").First(&voucher, voucher.ID)
		c.JSON(http.StatusCreated, gin.H{"data": voucher})
	}
}

func getVoucher(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		vid := c.Param("vid")
		var voucher models.Voucher
		if err := db.Preload("Items").First(&voucher, vid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "凭证不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": voucher})
	}
}

func updateVoucher(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		vid := c.Param("vid")
		var voucher models.Voucher
		if err := db.First(&voucher, vid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "凭证不存在"})
			return
		}
		if voucher.Status != "draft" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只能修改草稿状态的凭证"})
			return
		}

		var req struct {
			Date        string `json:"date"`
			VoucherType string `json:"voucher_type"`
			Attachments int    `json:"attachments"`
			Memo        string `json:"memo"`
			Items       []struct {
				AccountID    uint    `json:"account_id"`
				AccountCode  string  `json:"account_code"`
				AccountName  string  `json:"account_name"`
				Debit        float64 `json:"debit"`
				Credit       float64 `json:"credit"`
				Memo         string  `json:"memo"`
				AuxCustomer  *uint   `json:"aux_customer_id"`
				AuxSupplier  *uint   `json:"aux_supplier_id"`
				AuxDept      *uint   `json:"aux_department_id"`
				AuxProject   *uint   `json:"aux_project_id"`
				AuxEmployee  *uint   `json:"aux_employee_id"`
				AuxWarehouse *uint   `json:"aux_warehouse_id"`
				AuxBank      *uint   `json:"aux_bank_account_id"`
				CashFlow     *uint   `json:"cash_flow_id"`
			} `json:"items"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tx := db.Begin()

		// Update voucher header
		if req.Date != "" {
			tx.Model(&voucher).Update("date", req.Date)
		}
		if req.VoucherType != "" {
			tx.Model(&voucher).Update("voucher_type", req.VoucherType)
		}
		tx.Model(&voucher).Update("attachments", req.Attachments)
		if req.Memo != "" {
			tx.Model(&voucher).Update("memo", req.Memo)
		}

		// Update items if provided
		if len(req.Items) > 0 {
			// Delete old items
			tx.Where("voucher_id = ?", voucher.ID).Delete(&models.VoucherItem{})

			var totalDebit, totalCredit float64
			for i, item := range req.Items {
				totalDebit += item.Debit
				totalCredit += item.Credit
				vi := models.VoucherItem{
					VoucherID:        voucher.ID,
					LineNo:           i + 1,
					AccountID:        item.AccountID,
					AccountCode:      item.AccountCode,
					AccountName:      item.AccountName,
					Debit:            item.Debit,
					Credit:           item.Credit,
					Memo:             item.Memo,
					AuxCustomerID:    item.AuxCustomer,
					AuxSupplierID:    item.AuxSupplier,
					AuxDepartmentID:  item.AuxDept,
					AuxProjectID:     item.AuxProject,
					AuxEmployeeID:    item.AuxEmployee,
					AuxWarehouseID:   item.AuxWarehouse,
					AuxBankAccountID: item.AuxBank,
					CashFlowID:       item.CashFlow,
				}
				tx.Create(&vi)
			}

			tx.Model(&voucher).Updates(map[string]interface{}{
				"total_debit":  totalDebit,
				"total_credit": totalCredit,
			})
		}

		tx.Commit()

		db.Preload("Items").First(&voucher, voucher.ID)
		c.JSON(http.StatusOK, gin.H{"data": voucher})
	}
}

func deleteVoucher(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		vid := c.Param("vid")
		var voucher models.Voucher
		if err := db.First(&voucher, vid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "凭证不存在"})
			return
		}
		if voucher.Status != "draft" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只能删除草稿状态的凭证"})
			return
		}

		tx := db.Begin()
		tx.Where("voucher_id = ?", vid).Delete(&models.VoucherItem{})
		tx.Delete(&models.Voucher{}, vid)
		tx.Commit()

		c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
	}
}

func reviewVoucher(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		vid := c.Param("vid")
		var voucher models.Voucher
		if err := db.First(&voucher, vid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "凭证不存在"})
			return
		}
		if voucher.Status != "draft" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只能审核草稿状态的凭证"})
			return
		}
		db.Model(&voucher).Updates(map[string]interface{}{
			"status":      "reviewed",
			"reviewed_by": "admin",
		})
		c.JSON(http.StatusOK, gin.H{"message": "审核成功"})
	}
}

func unreviewVoucher(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		vid := c.Param("vid")
		var voucher models.Voucher
		if err := db.First(&voucher, vid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "凭证不存在"})
			return
		}
		if voucher.Status != "reviewed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只能反审核已审核状态的凭证"})
			return
		}
		db.Model(&voucher).Updates(map[string]interface{}{
			"status":       "draft",
			"reviewed_by":  "",
		})
		c.JSON(http.StatusOK, gin.H{"message": "反审核成功"})
	}
}

func postVoucher(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		vid := c.Param("vid")
		var voucher models.Voucher
		if err := db.First(&voucher, vid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "凭证不存在"})
			return
		}
		if voucher.Status != "reviewed" && voucher.Status != "draft" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只能记账已审核或草稿状态的凭证"})
			return
		}

		// Update account balances
		if err := updateAccountBalances(db, &voucher); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新科目余额失败: " + err.Error()})
			return
		}

		db.Model(&voucher).Updates(map[string]interface{}{
			"status":    "posted",
			"posted_by": "admin",
		})
		c.JSON(http.StatusOK, gin.H{"message": "记账成功"})
	}
}

func unpostVoucher(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		vid := c.Param("vid")
		var voucher models.Voucher
		if err := db.First(&voucher, vid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "凭证不存在"})
			return
		}
		if voucher.Status != "posted" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只能反记账已记账状态的凭证"})
			return
		}

		// Reverse account balances
		if err := reverseAccountBalances(db, &voucher); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "冲销科目余额失败: " + err.Error()})
			return
		}

		db.Model(&voucher).Updates(map[string]interface{}{
			"status":    "reviewed",
			"posted_by": "",
		})
		c.JSON(http.StatusOK, gin.H{"message": "反记账成功"})
	}
}

func batchReview(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var req struct {
			IDs []uint `json:"ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result := db.Model(&models.Voucher{}).
			Where("book_id = ? AND id IN ? AND status = ?", bookID, req.IDs, "draft").
			Updates(map[string]interface{}{"status": "reviewed", "reviewed_by": "admin"})

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("批量审核成功，共 %d 条", result.RowsAffected),
		})
	}
}

func batchPost(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var req struct {
			IDs []uint `json:"ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var vouchers []models.Voucher
		db.Where("book_id = ? AND id IN ? AND status = ?", bookID, req.IDs, "reviewed").Find(&vouchers)

		count := 0
		for _, v := range vouchers {
			if err := updateAccountBalances(db, &v); err != nil {
				continue
			}
			db.Model(&v).Updates(map[string]interface{}{"status": "posted", "posted_by": "admin"})
			count++
		}

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("批量记账成功，共 %d 条", count),
		})
	}
}

// ===== Reports =====

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

func listAuxItems(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		auxType := c.Param("type")

		var items []models.AuxItem
		db.Where("book_id = ? AND type = ?", bookID, auxType).Order("code").Find(&items)
		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}

func createAuxItem(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		auxType := c.Param("type")

		bid, _ := strconv.ParseUint(bookID, 10, 64)
		var item models.AuxItem
		if err := c.ShouldBindJSON(&item); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		item.BookID = uint(bid)
		item.Type = auxType

		if err := db.Create(&item).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": item})
	}
}

func updateAuxItem(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		aid := c.Param("aid")
		var item models.AuxItem
		if err := db.First(&item, aid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "辅助核算项不存在"})
			return
		}

		var req struct {
			Name     string `json:"name"`
			Code     string `json:"code"`
			IsActive *bool  `json:"is_active"`
			Extra    string `json:"extra"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updates := map[string]interface{}{}
		if req.Name != "" {
			updates["name"] = req.Name
		}
		if req.Code != "" {
			updates["code"] = req.Code
		}
		if req.IsActive != nil {
			updates["is_active"] = *req.IsActive
		}
		if req.Extra != "" {
			updates["extra"] = req.Extra
		}

		db.Model(&item).Updates(updates)
		c.JSON(http.StatusOK, gin.H{"data": item})
	}
}

func deleteAuxItem(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		aid := c.Param("aid")
		if err := db.Delete(&models.AuxItem{}, aid).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
	}
}

// ===== Helpers =====

func generateID(db *gorm.DB) uint {
	var count int64
	db.Model(&models.AccountBook{}).Count(&count)
	return uint(count + 1)
}

func generateVoucherNumber(db *gorm.DB, bookID uint, date string) string {
	// Format: 记-YYYYMM-001
	parts := strings.Split(date, "-")
	period := parts[0] + parts[1]

	var count int64
	db.Model(&models.Voucher{}).
		Where("book_id = ? AND date LIKE ?", bookID, period+"%").
		Count(&count)

	return fmt.Sprintf("记-%s-%03d", period, count+1)
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
			"period_debit":  gorm.Expr("period_debit + ?", item.Debit),
			"period_credit": gorm.Expr("period_credit + ?", item.Credit),
			"ytd_debit":     gorm.Expr("ytd_debit + ?", item.Debit),
			"ytd_credit":    gorm.Expr("ytd_credit + ?", item.Credit),
			"closing_debit": gorm.Expr("closing_debit + ?", item.Debit),
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
