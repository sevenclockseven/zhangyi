package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sevenclockseven/zhangyi/internal/models"
	"gorm.io/gorm"
)

// ========== 资产变动（状态流转） ==========

func changeAssetStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cardID := c.Param("cardId")

		var req struct {
			Status      string `json:"status" binding:"required"`
			Location    string `json:"location"`
			Department  string `json:"department"`
			EmployeeName string `json:"employee_name"`
			Note        string `json:"note"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var card models.AssetCard
		if err := db.First(&card, cardID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "资产卡片不存在"})
			return
		}

		oldStatus := card.Status

		updates := map[string]interface{}{"status": req.Status}
		if req.Location != "" {
			updates["location"] = req.Location
		}
		if req.Department != "" {
			updates["department"] = req.Department
		}
		if req.EmployeeName != "" {
			updates["employee_name"] = req.EmployeeName
		}

		if err := db.Model(&card).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		db.Create(&models.AssetTransaction{
			CardID:       card.ID,
			Type:         "transfer",
			AmountBefore: card.NetValue,
			AmountAfter:  card.NetValue,
			Note:         req.Note,
		})

		db.First(&card, cardID)

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"card":       card,
				"old_status": oldStatus,
				"new_status": req.Status,
			},
		})
	}
}

// ========== 资产变动记录 ==========

func listAssetTransactions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		cardID := c.Param("cardId")

		var transactions []models.AssetTransaction
		if cardID == "0" {
			db.Raw(`
				SELECT t.* FROM asset_transactions t
				JOIN asset_cards a ON t.card_id = a.id
				WHERE a.book_id = ?
				ORDER BY t.created_at DESC
				LIMIT 200
			`, bookID).Scan(&transactions)
		} else {
			db.Where("card_id = ?", cardID).Order("created_at DESC").Find(&transactions)
		}

		c.JSON(http.StatusOK, gin.H{"data": transactions})
	}
}

func listAllAssetTransactions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var transactions []models.AssetTransaction
		db.Raw(`
			SELECT t.* FROM asset_transactions t
			JOIN asset_cards a ON t.card_id = a.id
			WHERE a.book_id = ?
			ORDER BY t.created_at DESC
			LIMIT 200
		`, bookID).Scan(&transactions)
		c.JSON(http.StatusOK, gin.H{"data": transactions})
	}
}

// ========== 资产导入 ==========

func importAssets(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		bid, _ := strconv.ParseUint(bookID, 10, 64)

		var req struct {
			Items []struct {
				Code                   string  `json:"code"`
				Name                   string  `json:"name"`
				SpecModel              string  `json:"spec_model"`
				CategoryID             uint    `json:"category_id"`
				OriginalValue          float64 `json:"original_value"`
				Status                 string  `json:"status"`
				Department             string  `json:"department"`
				EmployeeName           string  `json:"employee_name"`
				Location               string  `json:"location"`
				AcquisitionDate        string  `json:"acquisition_date"`
				DepreciationStartMonth string  `json:"depreciation_start_month"`
				UsefulLifeMonths       int     `json:"useful_life_months"`
				ResidualValueRate      float64 `json:"residual_value_rate"`
				Source                 string  `json:"source"`
				Remark                 string  `json:"remark"`
			} `json:"items" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		imported := 0
		var importErrors []string

		for _, item := range req.Items {
			var cat models.AssetCategory
			if item.CategoryID > 0 {
				db.First(&cat, item.CategoryID)
			}

			usefulLife := item.UsefulLifeMonths
			if usefulLife == 0 {
				usefulLife = cat.UsefulLifeMonths
			}
			if usefulLife == 0 {
				usefulLife = 60
			}

			residualRate := item.ResidualValueRate
			if residualRate == 0 {
				residualRate = cat.ResidualValueRate
			}
			if residualRate == 0 {
				residualRate = 0.05
			}

			residualValue := item.OriginalValue * residualRate
			monthlyDep := 0.0
			if usefulLife > 0 {
				monthlyDep = (item.OriginalValue - residualValue) / float64(usefulLife)
			}

			status := item.Status
			if status == "" {
				status = "in_use"
			}

			card := models.AssetCard{
				Code:                   item.Code,
				Name:                   item.Name,
				SpecModel:              item.SpecModel,
				CategoryID:             item.CategoryID,
				OriginalValue:          item.OriginalValue,
				AccumulatedDepreciation: 0,
				NetValue:               item.OriginalValue,
				ResidualValue:          residualValue,
				Status:                 status,
				BookID:                 uint(bid),
				Department:             item.Department,
				EmployeeName:           item.EmployeeName,
				Location:               item.Location,
				AcquisitionDate:        item.AcquisitionDate,
				DepreciationStartMonth: item.DepreciationStartMonth,
				UsefulLifeMonths:       usefulLife,
				ResidualValueRate:      residualRate,
				MonthlyDepreciation:    monthlyDep,
				Source:                 item.Source,
				Remark:                 item.Remark,
			}

			if err := db.Create(&card).Error; err != nil {
				importErrors = append(importErrors, item.Code+": "+err.Error())
				continue
			}

			db.Create(&models.AssetTransaction{
				CardID:       card.ID,
				Type:         "acquire",
				AmountBefore: 0,
				AmountAfter:  card.OriginalValue,
				Note:         "批量导入",
			})

			imported++
		}

		c.JSON(http.StatusOK, gin.H{
			"imported": imported,
			"errors":   importErrors,
			"total":    len(req.Items),
		})
	}
}

// ========== 资产导出 ==========

func exportAssets(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")

		var cards []models.AssetCard
		db.Where("book_id = ?", bookID).Order("code").Find(&cards)

		type ExportItem struct {
			Code                   string  `json:"code"`
			Name                   string  `json:"name"`
			SpecModel              string  `json:"spec_model"`
			CategoryName           string  `json:"category_name"`
			OriginalValue          float64 `json:"original_value"`
			AccumulatedDepreciation float64 `json:"accumulated_depreciation"`
			NetValue              float64 `json:"net_value"`
			MonthlyDepreciation   float64 `json:"monthly_depreciation"`
			Status                 string  `json:"status"`
			Department             string  `json:"department"`
			EmployeeName           string  `json:"employee_name"`
			Location               string  `json:"location"`
			AcquisitionDate        string  `json:"acquisition_date"`
			DepreciationStartMonth string  `json:"depreciation_start_month"`
			UsefulLifeMonths       int     `json:"useful_life_months"`
			ResidualValueRate      float64 `json:"residual_value_rate"`
			Source                 string  `json:"source"`
			Remark                 string  `json:"remark"`
		}

		var items []ExportItem
		for _, card := range cards {
			catName := ""
			var cat models.AssetCategory
			if err := db.First(&cat, card.CategoryID).Error; err == nil {
				catName = cat.Name
			}
			items = append(items, ExportItem{
				Code:                   card.Code,
				Name:                   card.Name,
				SpecModel:              card.SpecModel,
				CategoryName:           catName,
				OriginalValue:          card.OriginalValue,
				AccumulatedDepreciation: card.AccumulatedDepreciation,
				NetValue:              card.NetValue,
				MonthlyDepreciation:   card.MonthlyDepreciation,
				Status:                 card.Status,
				Department:             card.Department,
				EmployeeName:           card.EmployeeName,
				Location:               card.Location,
				AcquisitionDate:        card.AcquisitionDate,
				DepreciationStartMonth: card.DepreciationStartMonth,
				UsefulLifeMonths:       card.UsefulLifeMonths,
				ResidualValueRate:      card.ResidualValueRate,
				Source:                 card.Source,
				Remark:                 card.Remark,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}


