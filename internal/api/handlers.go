package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
		if keyword := c.Query("keyword"); keyword != "" {
			query = query.Where("number LIKE ? OR memo LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
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
// ===== Additional Phase 1 Handlers =====

// voidVoucher marks a voucher as voided
func voidVoucher(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		vid, _ := strconv.ParseUint(c.Param("vid"), 10, 64)

		var voucher models.Voucher
		if err := db.Where("id = ? AND book_id = ?", vid, bookID).First(&voucher).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "凭证不存在"})
			return
		}

		if voucher.Status == "posted" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "已记账凭证不能作废，请先反记账"})
			return
		}

		if voucher.Status == "voided" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "凭证已作废"})
			return
		}

		if err := db.Model(&voucher).Update("status", "voided").Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "凭证已作废", "data": voucher})
	}
}

// restoreVoucher restores a voided voucher to draft
func restoreVoucher(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		vid, _ := strconv.ParseUint(c.Param("vid"), 10, 64)

		var voucher models.Voucher
		if err := db.Where("id = ? AND book_id = ?", vid, bookID).First(&voucher).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "凭证不存在"})
			return
		}

		if voucher.Status != "voided" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只有作废凭证才能恢复"})
			return
		}

		if err := db.Model(&voucher).Update("status", "draft").Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "凭证已恢复为草稿", "data": voucher})
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
func exportAuxItems(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		auxType := c.Param("type")

		var items []models.AuxItem
		if err := db.Where("book_id = ? AND type = ?", bookID, auxType).Order("code ASC").Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Build CSV with BOM for Excel
		var buf strings.Builder
		buf.WriteString("\xEF\xBB\xBF") // UTF-8 BOM

		// Header based on type
		switch auxType {
		case "customer", "supplier":
			buf.WriteString("编码,名称,联系人,电话,地址,备注,状态\n")
			for _, item := range items {
				extra := parseExtra(item.Extra)
				buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s\n",
					quoteCSV(item.Code), quoteCSV(item.Name),
					quoteCSV(extra["contact"]), quoteCSV(extra["phone"]),
					quoteCSV(extra["address"]), quoteCSV(extra["memo"]),
					boolStatus(item.IsActive)))
			}
		case "department":
			buf.WriteString("编码,名称,上级部门,状态\n")
			for _, item := range items {
				extra := parseExtra(item.Extra)
				buf.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
					quoteCSV(item.Code), quoteCSV(item.Name),
					quoteCSV(extra["parent"]), boolStatus(item.IsActive)))
			}
		case "project":
			buf.WriteString("编码,名称,状态,开始日期,结束日期,备注\n")
			for _, item := range items {
				extra := parseExtra(item.Extra)
				buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
					quoteCSV(item.Code), quoteCSV(item.Name),
					quoteCSV(extra["status"]), quoteCSV(extra["start_date"]),
					quoteCSV(extra["end_date"]), quoteCSV(extra["memo"])))
			}
		case "employee":
			buf.WriteString("编码,姓名,部门,电话,备注,状态\n")
			for _, item := range items {
				extra := parseExtra(item.Extra)
				buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
					quoteCSV(item.Code), quoteCSV(item.Name),
					quoteCSV(extra["department"]), quoteCSV(extra["phone"]),
					quoteCSV(extra["memo"]), boolStatus(item.IsActive)))
			}
		case "warehouse":
			buf.WriteString("编码,名称,地址,备注,状态\n")
			for _, item := range items {
				extra := parseExtra(item.Extra)
				buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
					quoteCSV(item.Code), quoteCSV(item.Name),
					quoteCSV(extra["address"]), quoteCSV(extra["memo"]),
					boolStatus(item.IsActive)))
			}
		case "bank_account":
			buf.WriteString("编码,名称,银行账号,开户行,户名,地址,备注,状态\n")
			for _, item := range items {
				extra := parseExtra(item.Extra)
				buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s\n",
					quoteCSV(item.Code), quoteCSV(item.Name),
					quoteCSV(extra["account_number"]), quoteCSV(extra["bank_name"]),
					quoteCSV(extra["account_holder"]), quoteCSV(extra["address"]),
					quoteCSV(extra["memo"]), boolStatus(item.IsActive)))
			}
		default:
			buf.WriteString("编码,名称,状态\n")
			for _, item := range items {
				buf.WriteString(fmt.Sprintf("%s,%s,%s\n",
					quoteCSV(item.Code), quoteCSV(item.Name), boolStatus(item.IsActive)))
			}
		}

		filename := fmt.Sprintf("%s_%s.csv", auxType, time.Now().Format("20060102"))
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.String(http.StatusOK, buf.String())
	}
}

// importAuxItems imports aux items from CSV
func importAuxItems(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		auxType := c.Param("type")

		file, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
			return
		}
		defer file.Close()

		// Read content
		content, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
			return
		}

		// Remove BOM if present
		text := string(content)
		if len(text) > 3 && text[:3] == "\xEF\xBB\xBF" {
			text = text[3:]
		}

		lines := strings.Split(strings.TrimSpace(text), "\n")
		if len(lines) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件内容为空"})
			return
		}

		// Skip header
		var created, updated, skipped int
		for i, line := range lines[1:] {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fields := parseCSVLine(line)
			if len(fields) < 2 {
				skipped++
				continue
			}

			code := fields[0]
			name := fields[1]
			if code == "" || name == "" {
				skipped++
				continue
			}

			// Build extra based on type
			extra := make(map[string]string)
			switch auxType {
			case "customer", "supplier":
				if len(fields) > 2 { extra["contact"] = fields[2] }
				if len(fields) > 3 { extra["phone"] = fields[3] }
				if len(fields) > 4 { extra["address"] = fields[4] }
				if len(fields) > 5 { extra["memo"] = fields[5] }
			case "department":
				if len(fields) > 2 { extra["parent"] = fields[2] }
			case "project":
				if len(fields) > 2 { extra["status"] = fields[2] }
				if len(fields) > 3 { extra["start_date"] = fields[3] }
				if len(fields) > 4 { extra["end_date"] = fields[4] }
				if len(fields) > 5 { extra["memo"] = fields[5] }
			case "employee":
				if len(fields) > 2 { extra["department"] = fields[2] }
				if len(fields) > 3 { extra["phone"] = fields[3] }
				if len(fields) > 4 { extra["memo"] = fields[4] }
			case "warehouse":
				if len(fields) > 2 { extra["address"] = fields[2] }
				if len(fields) > 3 { extra["memo"] = fields[3] }
			case "bank_account":
				if len(fields) > 2 { extra["account_number"] = fields[2] }
				if len(fields) > 3 { extra["bank_name"] = fields[3] }
				if len(fields) > 4 { extra["account_holder"] = fields[4] }
				if len(fields) > 5 { extra["address"] = fields[5] }
				if len(fields) > 6 { extra["memo"] = fields[6] }
			}

			extraJSON, _ := json.Marshal(extra)

			// Check existing
			var existing models.AuxItem
			result := db.Where("book_id = ? AND type = ? AND code = ?", bookID, auxType, code).First(&existing)
			if result.Error == gorm.ErrRecordNotFound {
				item := models.AuxItem{
					BookID:   uint(bookID),
					Type:     auxType,
					Code:     code,
					Name:     name,
					Extra:    string(extraJSON),
					IsActive: true,
				}
				if err := db.Create(&item).Error; err != nil {
					skipped++
					continue
				}
				created++
			} else {
				// Update existing
				db.Model(&existing).Updates(map[string]interface{}{
					"name":  name,
					"extra": string(extraJSON),
				})
				updated++
			}
			_ = i
		}

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("导入完成：新增%d，更新%d，跳过%d", created, updated, skipped),
			"created": created,
			"updated": updated,
			"skipped": skipped,
		})
	}
}

// batchDeleteAuxItems batch deletes aux items
func batchDeleteAuxItems(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		auxType := c.Param("type")

		var req struct {
			IDs []uint `json:"ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要删除的项目"})
			return
		}

		result := db.Where("id IN ? AND book_id = ? AND type = ?", req.IDs, bookID, auxType).Delete(&models.AuxItem{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("已删除%d条", result.RowsAffected)})
	}
}

// Helper functions
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
// ===== Opening Balance Handlers =====

// getOpeningBalances returns opening balances for all accounts in a book
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
