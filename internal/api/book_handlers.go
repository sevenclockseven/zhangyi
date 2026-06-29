package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sevenclockseven/zhangyi/internal/models"
	"github.com/sevenclockseven/zhangyi/internal/services"
	"gorm.io/gorm"
)

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
			Currency:           "CNY",
			Status:             "active",
			Contact:            req.Contact,
			Phone:              req.Phone,
			Address:            req.Address,
			Memo:               req.Memo,
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

		// Initialize default cash flow items
		initCashFlowItems(tx, book.ID)

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
			Status             string `json:"status"`
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
		if req.Status != "" {
			updates["status"] = req.Status
		}

		if len(updates) > 0 {
			if err := db.Model(&book).Updates(updates).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// reload updated book
			if err := db.First(&book, id).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

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


// initCashFlowItems creates default cash flow items for a new book
func initCashFlowItems(tx *gorm.DB, bookID uint) {
	type CashFlowDef struct {
		Code string
		Name string
		Cat  string // operating/investing/financing
	}

	defs := []CashFlowDef{
		// 经营活动
		{"CF-0101", "销售商品、提供劳务收到的现金", "operating"},
		{"CF-0102", "收到的税费返还", "operating"},
		{"CF-0103", "收到其他与经营活动有关的现金", "operating"},
		{"CF-0104", "购买商品、接受劳务支付的现金", "operating"},
		{"CF-0105", "支付给职工以及为职工支付的现金", "operating"},
		{"CF-0106", "支付的各项税费", "operating"},
		{"CF-0107", "支付其他与经营活动有关的现金", "operating"},
		// 投资活动
		{"CF-0201", "收回投资收到的现金", "investing"},
		{"CF-0202", "取得投资收益收到的现金", "investing"},
		{"CF-0203", "处置固定资产、无形资产和其他长期资产收回的现金净额", "investing"},
		{"CF-0204", "处置子公司及其他营业单位收到的现金净额", "investing"},
		{"CF-0205", "收到其他与投资活动有关的现金", "investing"},
		{"CF-0206", "购建固定资产、无形资产和其他长期资产支付的现金", "investing"},
		{"CF-0207", "投资支付的现金", "investing"},
		{"CF-0208", "取得子公司及其他营业单位支付的现金净额", "investing"},
		{"CF-0209", "支付其他与投资活动有关的现金", "investing"},
		// 筹资活动
		{"CF-0301", "吸收投资收到的现金", "financing"},
		{"CF-0302", "取得借款收到的现金", "financing"},
		{"CF-0303", "收到其他与筹资活动有关的现金", "financing"},
		{"CF-0304", "偿还债务支付的现金", "financing"},
		{"CF-0305", "分配股利、利润或偿付利息支付的现金", "financing"},
		{"CF-0306", "支付其他与筹资活动有关的现金", "financing"},
	}

	for _, d := range defs {
		extra := fmt.Sprintf(`{"category":"%s"}`, d.Cat)
		item := models.AuxItem{
			BookID:   bookID,
			Type:     "cash_flow",
			Code:     d.Code,
			Name:     d.Name,
			Extra:    extra,
			IsActive: true,
		}
		tx.Create(&item)
	}
}
