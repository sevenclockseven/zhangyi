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
			Name               string   `json:"name" binding:"required"`
			Code               string   `json:"code"`
			Industry           []string `json:"industry" binding:"required"`
			TaxpayerType       string   `json:"taxpayer_type"`
			AccountingStandard string   `json:"accounting_standard"`
			StartDate          string   `json:"start_date" binding:"required"`
			Contact            string   `json:"contact"`
			Phone              string   `json:"phone"`
			Address            string   `json:"address"`
			Memo               string   `json:"memo"`
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
			Code:               code,
			Name:               req.Name,
			Industry:           strings.Join(req.Industry, ","),
			TaxpayerType:       req.TaxpayerType,
			AccountingStandard: req.AccountingStandard,
			StartDate:          req.StartDate,
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
		if err := services.ApplyTemplateToBook(tx, book.ID, templateDir(), req.Industry, req.TaxpayerType, req.AccountingStandard); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "加载科目模板失败: " + err.Error()})
			return
		}

		// Load and apply report templates
		industry := ""
		if len(req.Industry) > 0 {
			industry = req.Industry[0]
		}
		services.ApplyReportTemplates(tx, book.ID, templateDir(), req.TaxpayerType, industry)

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
			Name               string `json:"name"`
			TaxpayerType       string `json:"taxpayer_type"`
			AccountingStandard string `json:"accounting_standard"`
			Contact            string `json:"contact"`
			Phone              string `json:"phone"`
			Address            string `json:"address"`
			Memo               string `json:"memo"`
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
		if req.AccountingStandard != "" {
			updates["accounting_standard"] = req.AccountingStandard
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
