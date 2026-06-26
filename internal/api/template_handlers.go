package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sevenclockseven/zhangyi/internal/models"
	"github.com/sevenclockseven/zhangyi/internal/services"
	"gorm.io/gorm"
)

func listVoucherTemplates(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var templates []models.VoucherTemplate
		db.Where("book_id = ? OR book_id IS NULL", bookID).Order("category ASC, name ASC").Find(&templates)
		c.JSON(http.StatusOK, gin.H{"data": templates})
	}
}

func createVoucherTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var req struct {
			Name     string `json:"name" binding:"required"`
			Category string `json:"category"`
			Items    string `json:"items" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tpl := models.VoucherTemplate{
			BookID:   &[]uint{uint(bookID)}[0],
			Name:     req.Name,
			Category: req.Category,
			Items:    req.Items,
		}
		if err := db.Create(&tpl).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": tpl})
	}
}

func updateVoucherTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid := c.Param("tid")
		var tpl models.VoucherTemplate
		if err := db.First(&tpl, tid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
			return
		}
		var req struct {
			Name     string `json:"name"`
			Category string `json:"category"`
			Items    string `json:"items"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updates := map[string]interface{}{}
		if req.Name != "" {
			updates["name"] = req.Name
		}
		if req.Category != "" {
			updates["category"] = req.Category
		}
		if req.Items != "" {
			updates["items"] = req.Items
		}
		db.Model(&tpl).Updates(updates)
		c.JSON(http.StatusOK, gin.H{"data": tpl})
	}
}

func deleteVoucherTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid := c.Param("tid")
		if err := db.Delete(&models.VoucherTemplate{}, tid).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "已删除"})
	}
}

// getTemplateManifest returns the v2 template manifest
func getTemplateManifest(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		manifest, err := services.GetManifest(templateDir())
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "模板清单不存在，请先生成v2模板"})
			return
		}
		c.JSON(http.StatusOK, manifest)
	}
}

// templateVersions returns version info for all templates
func templateVersions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dir := templateDir()
		files, err := os.ReadDir(dir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取模板目录失败"})
			return
		}

		type TemplateInfo struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Version string `json:"version"`
		}

		var templates []TemplateInfo
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				continue
			}
			var tpl struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Version string `json:"version"`
			}
			json.Unmarshal(data, &tpl)
			if tpl.ID != "" {
				templates = append(templates, TemplateInfo{
					ID:      tpl.ID,
					Name:    tpl.Name,
					Version: tpl.Version,
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{"data": templates})
	}
}
