package api

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sevenclockseven/zhangyi/internal/models"
	"gorm.io/gorm"
)

// ========== 资产分类 ==========

func listAssetCategories(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var categories []models.AssetCategory
		if err := db.Order("code").Find(&categories).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": categories})
	}
}

func createAssetCategory(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name                  string  `json:"name" binding:"required"`
			Code                  string  `json:"code"`
			ParentID              *uint   `json:"parent_id"`
			Method                string  `json:"method"`
			UsefulLifeMonths      int     `json:"useful_life_months"`
			ResidualValueRate     float64 `json:"residual_value_rate"`
			BookAccountID         uint    `json:"book_account_id"`
			DepreciationAccountID uint    `json:"depreciation_account_id"`
			ExpenseAccountID      uint    `json:"expense_account_id"`
			Memo                  string  `json:"memo"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cat := models.AssetCategory{
			Name:                  req.Name,
			Code:                  req.Code,
			ParentID:              req.ParentID,
			Method:                req.Method,
			UsefulLifeMonths:      req.UsefulLifeMonths,
			ResidualValueRate:     req.ResidualValueRate,
			BookAccountID:         req.BookAccountID,
			DepreciationAccountID: req.DepreciationAccountID,
			ExpenseAccountID:      req.ExpenseAccountID,
			Memo:                  req.Memo,
		}
		if cat.Method == "" {
			cat.Method = "straight_line"
		}
		if cat.UsefulLifeMonths == 0 {
			cat.UsefulLifeMonths = 60
		}
		if cat.ResidualValueRate == 0 {
			cat.ResidualValueRate = 0.05
		}
		if err := db.Create(&cat).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": cat})
	}
}

func updateAssetCategory(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var cat models.AssetCategory
		if err := db.First(&cat, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "分类不存在"})
			return
		}
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := db.Model(&cat).Updates(req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": cat})
	}
}

func deleteAssetCategory(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var count int64
		db.Model(&models.AssetCard{}).Where("category_id = ?", id).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "该分类下有资产卡片，无法删除"})
			return
		}
		if err := db.Delete(&models.AssetCategory{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "已删除"})
	}
}

// ========== 资产卡片 ==========

func listAssetCards(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var cards []models.AssetCard
		db := db.Where("book_id = ?", bookID)

		// 筛选
		if status := c.Query("status"); status != "" {
			db = db.Where("status = ?", status)
		}
		if categoryID := c.Query("category_id"); categoryID != "" {
			db = db.Where("category_id = ?", categoryID)
		}
		if keyword := c.Query("keyword"); keyword != "" {
			db = db.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}

		if err := db.Order("code").Find(&cards).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": cards})
	}
}

func getAssetCard(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var card models.AssetCard
		if err := db.First(&card, c.Param("cardId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "资产卡片不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": card})
	}
}

func createAssetCard(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		bid, _ := strconv.ParseUint(bookID, 10, 64)

		var req struct {
			Name                   string  `json:"name" binding:"required"`
			Code                   string  `json:"code" binding:"required"`
			SpecModel              string  `json:"spec_model"`
			SerialNumber           string  `json:"serial_number"`
			CategoryID             uint    `json:"category_id" binding:"required"`
			OriginalValue          float64 `json:"original_value" binding:"required"`
			Status                 string  `json:"status"`
			DepartmentID           *uint   `json:"department_id"`
			EmployeeID             *uint   `json:"employee_id"`
			Department             string  `json:"department"`
			EmployeeName           string  `json:"employee_name"`
			Location               string  `json:"location"`
			AcquisitionDate        string  `json:"acquisition_date"`
			DepreciationStartMonth string  `json:"depreciation_start_month"`
			UsefulLifeMonths       int     `json:"useful_life_months"`
			ResidualValueRate      float64 `json:"residual_value_rate"`
			Source                 string  `json:"source"`
			Vendor                 string  `json:"vendor"`
			InvoiceNo              string  `json:"invoice_no"`
			Remark                 string  `json:"remark"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 从分类继承默认值
		var cat models.AssetCategory
		if err := db.First(&cat, req.CategoryID).Error; err == nil {
			if req.UsefulLifeMonths == 0 {
				req.UsefulLifeMonths = cat.UsefulLifeMonths
			}
			if req.ResidualValueRate == 0 {
				req.ResidualValueRate = cat.ResidualValueRate
			}
		}

		// 计算净残值和月折旧额
		residualValue := req.OriginalValue * req.ResidualValueRate
		monthlyDepreciation := 0.0
		if req.UsefulLifeMonths > 0 {
			monthlyDepreciation = (req.OriginalValue - residualValue) / float64(req.UsefulLifeMonths)
		}

		// Resolve department/employee names from aux_items if IDs provided
		if req.DepartmentID != nil {
			var dept models.AuxItem
			if err := db.Where("id = ? AND type = ?", *req.DepartmentID, "department").First(&dept).Error; err == nil {
				req.Department = dept.Name
			}
		}
		if req.EmployeeID != nil {
			var emp models.AuxItem
			if err := db.Where("id = ? AND type = ?", *req.EmployeeID, "employee").First(&emp).Error; err == nil {
				req.EmployeeName = emp.Name
			}
		}

		card := models.AssetCard{
			Code:                   req.Code,
			Name:                   req.Name,
			SpecModel:              req.SpecModel,
			SerialNumber:           req.SerialNumber,
			CategoryID:             req.CategoryID,
			OriginalValue:          req.OriginalValue,
			AccumulatedDepreciation: 0,
			NetValue:               req.OriginalValue,
			ResidualValue:          residualValue,
			Status:                 req.Status,
			BookID:                 uint(bid),
			DepartmentID:           req.DepartmentID,
			EmployeeID:             req.EmployeeID,
			Department:             req.Department,
			EmployeeName:           req.EmployeeName,
			Location:               req.Location,
			AcquisitionDate:        req.AcquisitionDate,
			DepreciationStartMonth: req.DepreciationStartMonth,
			UsefulLifeMonths:       req.UsefulLifeMonths,
			ResidualValueRate:      req.ResidualValueRate,
			MonthlyDepreciation:    monthlyDepreciation,
			Source:                 req.Source,
			Vendor:                 req.Vendor,
			InvoiceNo:              req.InvoiceNo,
			Remark:                 req.Remark,
		}
		if card.Status == "" {
			card.Status = "in_use"
		}

		if err := db.Create(&card).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 记录变动流水
		db.Create(&models.AssetTransaction{
			CardID:       card.ID,
			Type:         "acquire",
			Date:         card.AcquisitionDate,
			AmountBefore: 0,
			AmountAfter:  card.OriginalValue,
			Note:         "资产购入",
		})

		c.JSON(http.StatusCreated, gin.H{"data": card})
	}
}

func updateAssetCard(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var card models.AssetCard
		if err := db.First(&card, c.Param("cardId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "资产卡片不存在"})
			return
		}
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := db.Model(&card).Updates(req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": card})
	}
}

func deleteAssetCard(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("cardId")
		if err := db.Delete(&models.AssetCard{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "已删除"})
	}
}

// ========== 折旧计提 ==========

func calcDepreciation(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", time.Now().Format("2006-01"))

		var cards []models.AssetCard
		db.Where("book_id = ? AND status = ?", bookID, "in_use").Find(&cards)

		type Result struct {
			CardID   uint    `json:"card_id"`
			CardName string  `json:"card_name"`
			Amount   float64 `json:"amount"`
			Skipped  bool    `json:"skipped"`
			Reason   string  `json:"reason"`
		}
		var results []Result

		for _, card := range cards {
			// 检查是否已提足折旧
			if card.NetValue <= card.ResidualValue {
				results = append(results, Result{CardID: card.ID, CardName: card.Name, Skipped: true, Reason: "已提足折旧"})
				continue
			}
			// 检查是否到折旧起始月
			if card.DepreciationStartMonth > period {
				results = append(results, Result{CardID: card.ID, CardName: card.Name, Skipped: true, Reason: "未到折旧起始月"})
				continue
			}
			// 检查本月是否已计提
			var existCount int64
			db.Model(&models.AssetDepreciation{}).Where("card_id = ? AND period = ?", card.ID, period).Count(&existCount)
			if existCount > 0 {
				results = append(results, Result{CardID: card.ID, CardName: card.Name, Skipped: true, Reason: "本月已计提"})
				continue
			}

			amount := card.MonthlyDepreciation
			if amount <= 0 {
				results = append(results, Result{CardID: card.ID, CardName: card.Name, Skipped: true, Reason: "月折旧额为0"})
				continue
			}
			// 最后一个月提完剩余
			newNet := card.NetValue - amount
			if newNet < card.ResidualValue {
				amount = card.NetValue - card.ResidualValue
				newNet = card.ResidualValue
			}

			results = append(results, Result{CardID: card.ID, CardName: card.Name, Amount: amount})
		}

		c.JSON(http.StatusOK, gin.H{"data": results, "period": period})
	}
}

func runDepreciation(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", time.Now().Format("2006-01"))

		var cards []models.AssetCard
		db.Where("book_id = ? AND status = ?", bookID, "in_use").Find(&cards)

		type Result struct {
			CardID   uint    `json:"card_id"`
			CardName string  `json:"card_name"`
			Amount   float64 `json:"amount"`
			VoucherID uint   `json:"voucher_id"`
		}
		var results []Result
		var totalAmount float64

		for _, card := range cards {
			if card.NetValue <= card.ResidualValue {
				continue
			}
			if card.DepreciationStartMonth > period {
				continue
			}
			var existCount int64
			db.Model(&models.AssetDepreciation{}).Where("card_id = ? AND period = ?", card.ID, period).Count(&existCount)
			if existCount > 0 {
				continue
			}

			amount := card.MonthlyDepreciation
			if amount <= 0 {
				continue
			}
			newNet := card.NetValue - amount
			if newNet < card.ResidualValue {
				amount = card.NetValue - card.ResidualValue
				newNet = card.ResidualValue
			}

			// 更新卡片累计折旧和净值
			card.AccumulatedDepreciation += amount
			card.NetValue = newNet
			db.Save(&card)

			// 记录折旧明细
			dep := models.AssetDepreciation{
				CardID:   card.ID,
				Period:   period,
				StartNet: card.NetValue + amount,
				Amount:   amount,
				EndNet:   newNet,
			}
			db.Create(&dep)

			// 记录变动流水
			db.Create(&models.AssetTransaction{
				CardID:       card.ID,
				Type:         "depreciate",
				Date:         period + "-01",
				AmountBefore: card.NetValue + amount,
				AmountAfter:  newNet,
				Note:         period + " 计提折旧",
			})

			results = append(results, Result{CardID: card.ID, CardName: card.Name, Amount: amount})
			totalAmount += amount
		}

		// 生成折旧凭证
		if totalAmount > 0 {
			catDepMap := map[uint]float64{}
			for _, card := range cards {
				if card.NetValue <= card.ResidualValue || card.DepreciationStartMonth > period {
					continue
				}
				amt := card.MonthlyDepreciation
				if amt <= 0 {
					continue
				}
				newNV := card.NetValue - amt
				if newNV < card.ResidualValue {
					amt = card.NetValue - card.ResidualValue
				}
				catDepMap[card.CategoryID] += amt
			}

			type vItem struct {
				AcctID uint
				Code   string
				Name   string
				Debit  float64
				Credit float64
				Memo   string
			}
			var vItems []vItem
			for catID, amt := range catDepMap {
				var cat models.AssetCategory
				if err := db.First(&cat, catID).Error; err != nil {
					continue
				}
				if cat.ExpenseAccountID == 0 || cat.DepreciationAccountID == 0 {
					continue
				}
				var expAcct models.Account
				db.First(&expAcct, cat.ExpenseAccountID)
				vItems = append(vItems, vItem{
					AcctID: cat.ExpenseAccountID,
					Code:   expAcct.Code,
					Name:   expAcct.Name,
					Debit:  amt,
					Memo:   "折旧费用",
				})
				var depAcct models.Account
				db.First(&depAcct, cat.DepreciationAccountID)
				vItems = append(vItems, vItem{
					AcctID: cat.DepreciationAccountID,
					Code:   depAcct.Code,
					Name:   depAcct.Name,
					Credit: amt,
					Memo:   "累计折旧",
				})
			}

			if len(vItems) > 0 {
				bid := parseBookID(bookID)
				periodDate := period + "-01"
				voucher := models.Voucher{
					BookID:      bid,
					Date:        periodDate,
					Number:      generateVoucherNumber(db, bid, periodDate),
					VoucherType: "depreciation",
					Status:      "posted",
					TotalDebit:  totalAmount,
					TotalCredit: totalAmount,
					Memo:        period + " 固定资产折旧计提",
					PreparedBy:  "system",
					ReviewedBy:  "system",
					PostedBy:    "system",
				}
				if err := db.Create(&voucher).Error; err == nil {
					for i, vi := range vItems {
						db.Create(&models.VoucherItem{
							VoucherID:   voucher.ID,
							LineNo:      i + 1,
							AccountID:   vi.AcctID,
							AccountCode: vi.Code,
							AccountName: vi.Name,
							Debit:       vi.Debit,
							Credit:      vi.Credit,
							Memo:        vi.Memo,
						})
					}
					updateAccountBalances(db, &voucher)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"data":   results,
			"period": period,
			"total":  totalAmount,
			"count":  len(results),
		})
	}
}

func parseBookID(s string) uint {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return uint(n)
}

// ========== 资产台账报表 ==========

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func assetSummary(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")

		type Stat struct {
			CategoryName       string  `json:"category_name"`
			Count              int64   `json:"count"`
			TotalOriginalValue          float64 `json:"total_original_value"`
			TotalAccumulatedDepreciation float64 `json:"total_accumulated_depreciation"`
			TotalNetValue               float64 `json:"total_net_value"`
		}
		var stats []Stat

		db.Raw(`
			SELECT c.name AS category_name, 
			       COUNT(*) AS count, 
			       COALESCE(SUM(a.original_value), 0) AS total_original_value,
		       COALESCE(SUM(a.accumulated_depreciation), 0) AS total_accumulated_depreciation,
			       COALESCE(SUM(a.net_value), 0) AS total_net_value
			FROM asset_cards a
			LEFT JOIN asset_categories c ON a.category_id = c.id
			WHERE a.book_id = ?
			GROUP BY c.name
			ORDER BY total_original_value DESC
		`, bookID).Scan(&stats)
	for i := range stats {
		stats[i].TotalOriginalValue = round2(stats[i].TotalOriginalValue)
		stats[i].TotalAccumulatedDepreciation = round2(stats[i].TotalAccumulatedDepreciation)
		stats[i].TotalNetValue = round2(stats[i].TotalNetValue)
	}

	// 总计
		var totalCount int64
		var totalOriginal, totalDep, totalNet float64
		db.Model(&models.AssetCard{}).Where("book_id = ?", bookID).Count(&totalCount)
		db.Model(&models.AssetCard{}).Where("book_id = ?", bookID).Select("COALESCE(SUM(original_value), 0)").Scan(&totalOriginal)
		db.Model(&models.AssetCard{}).Where("book_id = ?", bookID).Select("COALESCE(SUM(accumulated_depreciation), 0)").Scan(&totalDep)
		db.Model(&models.AssetCard{}).Where("book_id = ?", bookID).Select("COALESCE(SUM(net_value), 0)").Scan(&totalNet)

		c.JSON(http.StatusOK, gin.H{
			"summary":     stats,
			"total_count": totalCount,
			"total_original_value":          round2(totalOriginal),
			"total_accumulated_depreciation": round2(totalDep),
			"total_net_value":               round2(totalNet),
		})
	}
}
