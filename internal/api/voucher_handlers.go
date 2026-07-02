package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sevenclockseven/zhangyi/internal/models"
	"gorm.io/gorm"
)

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
			subQuery := db.Model(&models.VoucherItem{}).Select("DISTINCT voucher_id").Where("memo LIKE ?", "%"+keyword+"%")
			query = query.Where("number LIKE ? OR memo LIKE ? OR vouchers.id IN (?)", "%"+keyword+"%", "%"+keyword+"%", subQuery)
		}

		query.Preload("Items").Order("date DESC, number DESC").Find(&vouchers)
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
				AuxFixed     *uint   `json:"aux_fixed_asset_id"`
				AuxVat       *uint   `json:"aux_vat_detail_id"`
				AuxCost      *uint   `json:"aux_cost_object_id"`
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
				AuxFixedAssetID:  item.AuxFixed,
				AuxVatDetailID:   item.AuxVat,
				AuxCostObjectID:  item.AuxCost,
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
				AuxFixed     *uint   `json:"aux_fixed_asset_id"`
				AuxVat       *uint   `json:"aux_vat_detail_id"`
				AuxCost      *uint   `json:"aux_cost_object_id"`
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
			if err := tx.Where("voucher_id = ?", voucher.ID).Delete(&models.VoucherItem{}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

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
					AuxFixedAssetID:  item.AuxFixed,
					AuxVatDetailID:   item.AuxVat,
					AuxCostObjectID:  item.AuxCost,
					CashFlowID:       item.CashFlow,
				}
				if err := tx.Create(&vi).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}

			if err := tx.Model(&voucher).Updates(map[string]interface{}{
				"total_debit":  totalDebit,
				"total_credit": totalCredit,
			}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
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
		if err := tx.Where("voucher_id = ?", vid).Delete(&models.VoucherItem{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := tx.Delete(&models.Voucher{}, vid).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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
		username, _ := c.Get("username")
		if err := db.Model(&voucher).Updates(map[string]interface{}{
			"status":      "reviewed",
			"reviewed_by": username,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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
		if err := db.Model(&voucher).Updates(map[string]interface{}{
			"status":      "draft",
			"reviewed_by": "",
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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

		if err := db.Model(&voucher).Updates(map[string]interface{}{
			"status":    "posted",
			"posted_by": c.GetString("username"),
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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

		if err := db.Model(&voucher).Updates(map[string]interface{}{
			"status":    "reviewed",
			"posted_by": "",
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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

// exportVouchers exports vouchers as CSV (Excel compatible)
func exportVouchers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		period := c.Query("period") // YYYY-MM

		var vouchers []models.Voucher
		query := db.Where("book_id = ?", bookID)
		if period != "" {
			query = query.Where("date LIKE ?", period+"%")
		}
		query.Preload("Items").Order("date ASC, number ASC").Find(&vouchers)

		var buf strings.Builder
		buf.WriteString("\xEF\xBB\xBF")
		buf.WriteString("凭证字号,日期,科目编码,科目名称,摘要,借方金额,贷方金额,状态\n")

		for _, v := range vouchers {
			statusLabel := map[string]string{"draft": "草稿", "reviewed": "已审核", "posted": "已记账", "voided": "已作废"}[v.Status]
			for _, item := range v.Items {
				buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%.2f,%.2f,%s\n",
					quoteCSV(v.Number), v.Date, item.AccountCode, item.AccountName,
					quoteCSV(item.Memo), item.Debit, item.Credit, statusLabel))
			}
		}

		filename := fmt.Sprintf("vouchers_%s.csv", time.Now().Format("20060102"))
		if period != "" {
			filename = fmt.Sprintf("vouchers_%s_%s.csv", period, time.Now().Format("20060102"))
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.String(http.StatusOK, buf.String())
	}
}

// detectVoucherGaps detects gaps in voucher numbering for a given period
func detectVoucherGaps(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", "")

		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定期间"})
			return
		}

		// Query all voucher numbers for this period (both formats)
		var numbers []string
		db.Model(&models.Voucher{}).
			Where("book_id = ? AND (number LIKE ? OR number LIKE ?)",
				bookID, "记-"+period+"-%", "记-"+strings.ReplaceAll(period, "-", "")+"-%").
			Pluck("number", &numbers)

		// Extract sequence numbers
		seen := make(map[int]bool)
		maxSeq := 0
		for _, num := range numbers {
			idx := strings.LastIndex(num, "-")
			if idx < 0 {
				continue
			}
			seq := 0
			fmt.Sscanf(num[idx+1:], "%d", &seq)
			if seq > 0 {
				seen[seq] = true
				if seq > maxSeq {
					maxSeq = seq
				}
			}
		}

		// Find gaps
		var gaps []int
		for i := 1; i <= maxSeq; i++ {
			if !seen[i] {
				gaps = append(gaps, i)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"period":    period,
			"total":     maxSeq,
			"count":     len(numbers),
			"gaps":      gaps,
			"has_gaps":  len(gaps) > 0,
		})
	}
}
