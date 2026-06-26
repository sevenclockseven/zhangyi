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

		// Load all aux items for this book, grouped by type
		var auxItems []models.AuxItem
		db.Where("book_id = ? AND is_active = ?", bookID, true).Find(&auxItems)
		auxByType := make(map[string][]models.AuxItem)
		for _, item := range auxItems {
			auxByType[item.Type] = append(auxByType[item.Type], item)
		}

		// Build response with aux_options
		type AccountResp struct {
			models.Account
			AuxOptions map[string][]models.AuxItem `json:"aux_options"`
		}
		var resp []AccountResp
		for _, acct := range accounts {
			opts := map[string][]models.AuxItem{}
			if acct.AuxTypes != "" {
				for _, t := range strings.Split(acct.AuxTypes, ",") {
					t = strings.TrimSpace(t)
					if t != "" {
						opts[t] = auxByType[t]
					}
				}
			}
			resp = append(resp, AccountResp{Account: acct, AuxOptions: opts})
		}

		c.JSON(http.StatusOK, gin.H{"data": resp})
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

func syncAllTemplates(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

		var book models.AccountBook
		if err := db.First(&book, bookID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "账套不存在"})
			return
		}

		industries := strings.Split(book.Industry, ",")
		if err := services.ApplyTemplateToBook(db, uint(bookID), templateDir(), industries, book.TaxpayerType, book.AccountingStandard); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "同步失败: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "模板同步成功"})
	}
}
