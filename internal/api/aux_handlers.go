package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sevenclockseven/zhangyi/internal/models"
	"gorm.io/gorm"
)

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
		case "cash_flow":
			buf.WriteString("编码,名称,分类,状态\n")
			for _, item := range items {
				extra := parseExtra(item.Extra)
				buf.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
					quoteCSV(item.Code), quoteCSV(item.Name),
					quoteCSV(extra["category"]), boolStatus(item.IsActive)))
			}
		case "fixed_asset":
			buf.WriteString("编码,名称,原值,累计折旧,净值,折旧方法,使用年限,残值率,使用部门,存放地点,状态\n")
			for _, item := range items {
				extra := parseExtra(item.Extra)
				buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
					quoteCSV(item.Code), quoteCSV(item.Name),
					quoteCSV(extra["original_value"]), quoteCSV(extra["accumulated_depreciation"]),
					quoteCSV(extra["net_value"]), quoteCSV(extra["depreciation_method"]),
					quoteCSV(extra["useful_life"]), quoteCSV(extra["residual_rate"]),
					quoteCSV(extra["department"]), quoteCSV(extra["location"]),
					boolStatus(item.IsActive)))
			}
		case "vat_detail":
			buf.WriteString("编码,名称,进项税额,销项税额,已交税金,进项税额转出,状态\n")
			for _, item := range items {
				extra := parseExtra(item.Extra)
				buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s\n",
					quoteCSV(item.Code), quoteCSV(item.Name),
					quoteCSV(extra["input_vat"]), quoteCSV(extra["output_vat"]),
					quoteCSV(extra["paid_vat"]), quoteCSV(extra["input_vat_transfer"]),
					boolStatus(item.IsActive)))
			}
		case "cost_object":
			buf.WriteString("编码,名称,成本中心,产品/订单,状态\n")
			for _, item := range items {
				extra := parseExtra(item.Extra)
				buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
					quoteCSV(item.Code), quoteCSV(item.Name),
					quoteCSV(extra["cost_center"]), quoteCSV(extra["product_order"]),
					boolStatus(item.IsActive)))
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
				if len(fields) > 2 {
					extra["contact"] = fields[2]
				}
				if len(fields) > 3 {
					extra["phone"] = fields[3]
				}
				if len(fields) > 4 {
					extra["address"] = fields[4]
				}
				if len(fields) > 5 {
					extra["memo"] = fields[5]
				}
			case "department":
				if len(fields) > 2 {
					extra["parent"] = fields[2]
				}
			case "project":
				if len(fields) > 2 {
					extra["status"] = fields[2]
				}
				if len(fields) > 3 {
					extra["start_date"] = fields[3]
				}
				if len(fields) > 4 {
					extra["end_date"] = fields[4]
				}
				if len(fields) > 5 {
					extra["memo"] = fields[5]
				}
			case "employee":
				if len(fields) > 2 {
					extra["department"] = fields[2]
				}
				if len(fields) > 3 {
					extra["phone"] = fields[3]
				}
				if len(fields) > 4 {
					extra["memo"] = fields[4]
				}
			case "warehouse":
				if len(fields) > 2 {
					extra["address"] = fields[2]
				}
				if len(fields) > 3 {
					extra["memo"] = fields[3]
				}
			case "bank_account":
				if len(fields) > 2 {
					extra["account_number"] = fields[2]
				}
				if len(fields) > 3 {
					extra["bank_name"] = fields[3]
				}
				if len(fields) > 4 {
					extra["account_holder"] = fields[4]
				}
				if len(fields) > 5 {
					extra["address"] = fields[5]
				}
				if len(fields) > 6 {
					extra["memo"] = fields[6]
				}
			case "cash_flow":
				if len(fields) > 2 {
					extra["category"] = fields[2]
				}
			case "fixed_asset":
			if len(fields) > 2 { extra["original_value"] = fields[2] }
			if len(fields) > 3 { extra["accumulated_depreciation"] = fields[3] }
			if len(fields) > 4 { extra["net_value"] = fields[4] }
			if len(fields) > 5 { extra["depreciation_method"] = fields[5] }
			if len(fields) > 6 { extra["useful_life"] = fields[6] }
			if len(fields) > 7 { extra["residual_rate"] = fields[7] }
			if len(fields) > 8 { extra["department"] = fields[8] }
			if len(fields) > 9 { extra["location"] = fields[9] }
		case "vat_detail":
			if len(fields) > 2 { extra["input_vat"] = fields[2] }
			if len(fields) > 3 { extra["output_vat"] = fields[3] }
			if len(fields) > 4 { extra["paid_vat"] = fields[4] }
			if len(fields) > 5 { extra["input_vat_transfer"] = fields[5] }
		case "cost_object":
			if len(fields) > 2 { extra["cost_center"] = fields[2] }
			if len(fields) > 3 { extra["product_order"] = fields[3] }

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
