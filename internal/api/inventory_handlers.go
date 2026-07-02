package api

import (
	"encoding/csv"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sevenclockseven/zhangyi/internal/models"
	"gorm.io/gorm"
)

// ==================== 商品档案 ====================

func listGoods(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var goods []models.Goods
		query := db.Where("book_id = ? AND is_active = ?", bookID, true)
		if keyword := c.Query("keyword"); keyword != "" {
			query = query.Where("code LIKE ? OR name LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
		if category := c.Query("category"); category != "" {
			query = query.Where("category = ?", category)
		}
		query.Order("code").Find(&goods)
		c.JSON(http.StatusOK, gin.H{"data": goods})
	}
}

func createGoods(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var req models.Goods
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		req.BookID = uint(bookID)
		if err := db.Create(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": req})
	}
}

func updateGoods(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		gid := c.Param("gid")
		var goods models.Goods
		if err := db.First(&goods, gid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "商品不存在"})
			return
		}
		var req models.Goods
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		db.Model(&goods).Updates(map[string]interface{}{
			"code": req.Code, "name": req.Name, "category": req.Category,
			"unit": req.Unit, "barcode": req.Barcode, "cost_method": req.CostMethod,
			"tax_rate": req.TaxRate, "ref_price": req.RefPrice, "min_stock": req.MinStock,
			"purchase_account_id": req.PurchaseAccountID, "sales_account_id": req.SalesAccountID,
			"inventory_account_id": req.InventoryAccountID,
		})
		c.JSON(http.StatusOK, gin.H{"data": goods})
	}
}

func deleteGoods(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		gid := c.Param("gid")
		db.Model(&models.Goods{}).Where("id = ?", gid).Update("is_active", false)
		c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
	}
}

func exportGoods(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var goods []models.Goods
		db.Where("book_id = ? AND is_active = ?", bookID, true).Order("code").Find(&goods)

		var buf strings.Builder
		buf.WriteString("\uFEFF编码,名称,分类,单位,条码,成本方法,税率,参考进价,最低库存\n")
		for _, g := range goods {
			buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%.2f,%.2f,%.4f\n",
				g.Code, g.Name, g.Category, g.Unit, g.Barcode, g.CostMethod,
				g.TaxRate, g.RefPrice, g.MinStock))
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=goods.csv")
		c.String(http.StatusOK, buf.String())
	}
}

func importGoods(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "CSV解析失败"})
			return
		}

		var created, skipped int
		for i, row := range records {
			if i == 0 { // skip header
				continue
			}
			if len(row) < 4 {
				skipped++
				continue
			}
			goods := models.Goods{
				BookID:     uint(bookID),
				Code:       row[0],
				Name:       row[1],
				Category:   row[2],
				Unit:       row[3],
				IsActive:   true,
				CostMethod: "weighted_avg",
			}
			if len(row) > 4 {
				goods.Barcode = row[4]
			}
			if len(row) > 6 {
				fmt.Sscanf(row[6], "%f", &goods.TaxRate)
			}
			if len(row) > 7 {
				fmt.Sscanf(row[7], "%f", &goods.RefPrice)
			}
			if len(row) > 8 {
				fmt.Sscanf(row[8], "%f", &goods.MinStock)
			}
			if err := db.Create(&goods).Error; err != nil {
				skipped++
				continue
			}
			created++
		}
		c.JSON(http.StatusOK, gin.H{"created": created, "skipped": skipped})
	}
}

// ==================== 采购入库单 ====================

func listPurchases(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var orders []models.PurchaseOrder
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
		query.Order("date DESC, id DESC").Find(&orders)
		c.JSON(http.StatusOK, gin.H{"data": orders})
	}
}

func createPurchase(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var req struct {
			SupplierID  uint    `json:"supplier_id" binding:"required"`
			WarehouseID uint    `json:"warehouse_id" binding:"required"`
			Date        string  `json:"date" binding:"required"`
			PaymentTerm string  `json:"payment_term"`
			Memo        string  `json:"memo"`
			Items       []struct {
				GoodsID   uint    `json:"goods_id" binding:"required"`
				Quantity  float64 `json:"quantity" binding:"required,gt=0"`
				UnitPrice float64 `json:"unit_price" binding:"required,gt=0"`
				Memo      string  `json:"memo"`
			} `json:"items" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Generate order number
		orderNo := generateOrderNumber(db, uint(bookID), "PUR", req.Date)

		tx := db.Begin()

		var totalAmount float64
		var orderItems []models.PurchaseOrderItem
		for _, item := range req.Items {
			amount := item.Quantity * item.UnitPrice
			totalAmount += amount
			orderItems = append(orderItems, models.PurchaseOrderItem{
				GoodsID:    item.GoodsID,
				Quantity:   item.Quantity,
				UnitPrice:  item.UnitPrice,
				Amount:     amount,
				Memo:       item.Memo,
			})
		}

		order := models.PurchaseOrder{
			BookID:      uint(bookID),
			OrderNo:     orderNo,
			SupplierID:  req.SupplierID,
			WarehouseID: req.WarehouseID,
			Date:        req.Date,
			Status:      "draft",
			TotalAmount: totalAmount,
			PaymentTerm: req.PaymentTerm,
			Memo:        req.Memo,
			PreparedBy:  getCurrentUser(c),
		}
		if err := tx.Create(&order).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for i := range orderItems {
			orderItems[i].PurchaseOrderID = order.ID
		}
		if err := tx.Create(&orderItems).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"data": order})
	}
}

func getPurchase(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		pid := c.Param("pid")
		var order models.PurchaseOrder
		if err := db.Preload("Items").First(&order, pid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "采购单不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": order})
	}
}

func postPurchase(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		pid := c.Param("pid")
		var order models.PurchaseOrder
		if err := db.Preload("Items").First(&order, pid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "采购单不存在"})
			return
		}
		if order.Status != "draft" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只能审核草稿状态的采购单"})
			return
		}

		tx := db.Begin()

		// 1. Record stock flows and calculate total
		var totalAmount float64
		for _, item := range order.Items {
			amount := item.Quantity * item.UnitPrice
			totalAmount += amount

			flow := models.StockFlow{
				BookID:      order.BookID,
				GoodsID:     item.GoodsID,
				WarehouseID: order.WarehouseID,
				FlowType:    "purchase_in",
				Quantity:    item.Quantity,
				UnitPrice:   item.UnitPrice,
				Amount:      amount,
				RefType:     "purchase",
				RefID:       order.ID,
				Date:        order.Date,
				Memo:        fmt.Sprintf("采购入库 %s", order.OrderNo),
			}
			if err := tx.Create(&flow).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		// 2. Generate voucher
		// Get inventory account code (default 1405)
		var invAccount models.Account
		db.Where("book_id = ? AND code = ?", order.BookID, "1405").First(&invAccount)
		if invAccount.ID == 0 {
			// Fallback: use first inventory account
			db.Where("book_id = ? AND code LIKE ?", order.BookID, "14%").First(&invAccount)
		}

		// Get AP account code (2202)
		var apAccount models.Account
		db.Where("book_id = ? AND code = ?", order.BookID, "2202").First(&apAccount)

		voucherDate := order.Date
		if len(voucherDate) > 7 {
			voucherDate = voucherDate[:7] + "-01"
		}
		voucher := models.Voucher{
			BookID:      order.BookID,
			Date:        order.Date,
			Number:      generateVoucherNumber(tx, order.BookID, order.Date),
			VoucherType: "purchase",
			Status:      "posted",
			TotalDebit:  totalAmount,
			TotalCredit: totalAmount,
			Memo:        fmt.Sprintf("采购入库 %s", order.OrderNo),
			PreparedBy:  "system",
			ReviewedBy:  "system",
			PostedBy:    "system",
		}
		if err := tx.Create(&voucher).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Voucher items: Debit inventory, Credit AP
		vi1 := models.VoucherItem{
			VoucherID:     voucher.ID,
			LineNo:        1,
			AccountID:     invAccount.ID,
			AccountCode:   invAccount.Code,
			AccountName:   invAccount.Name,
			Debit:         totalAmount,
			Memo:          "采购入库",
			AuxWarehouseID: &order.WarehouseID,
		}
		vi2 := models.VoucherItem{
			VoucherID:     voucher.ID,
			LineNo:        2,
			AccountID:     apAccount.ID,
			AccountCode:   apAccount.Code,
			AccountName:   apAccount.Name,
			Credit:        totalAmount,
			Memo:          "应付账款",
			AuxSupplierID: &order.SupplierID,
		}
		tx.Create(&vi1)
		tx.Create(&vi2)

		// Update account balances
		updateAccountBalances(tx, &voucher)

		// 3. Update order status
		tx.Model(&order).Updates(map[string]interface{}{
			"status":         "posted",
			"total_amount":   totalAmount,
			"ref_voucher_id": voucher.ID,
		})

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"message": "审核成功", "voucher_id": voucher.ID})
	}
}

func voidPurchase(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		pid := c.Param("pid")
		var order models.PurchaseOrder
		if err := db.First(&order, pid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "采购单不存在"})
			return
		}
		if order.Status == "voided" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "采购单已作废"})
			return
		}

		tx := db.Begin()

		if order.Status == "posted" {
			// Reverse stock flows
			tx.Where("ref_type = ? AND ref_id = ?", "purchase", order.ID).Delete(&models.StockFlow{})

			// Reverse voucher
			if order.RefVoucherID > 0 {
				var voucher models.Voucher
				if err := tx.First(&voucher, order.RefVoucherID).Error; err == nil {
					// Delete voucher items and update account balances
					tx.Where("voucher_id = ?", voucher.ID).Delete(&models.VoucherItem{})
					tx.Delete(&voucher)
				}
			}
		}

		tx.Model(&order).Update("status", "voided")
		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"message": "作废成功"})
	}
}

// ==================== 销售出库单 ====================

func listSales(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var orders []models.SalesOrder
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
		query.Order("date DESC, id DESC").Find(&orders)
		c.JSON(http.StatusOK, gin.H{"data": orders})
	}
}

func createSales(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var req struct {
			CustomerID  uint    `json:"customer_id" binding:"required"`
			WarehouseID uint    `json:"warehouse_id" binding:"required"`
			Date        string  `json:"date" binding:"required"`
			PaymentTerm string  `json:"payment_term"`
			Memo        string  `json:"memo"`
			Items       []struct {
				GoodsID   uint    `json:"goods_id" binding:"required"`
				Quantity  float64 `json:"quantity" binding:"required,gt=0"`
				UnitPrice float64 `json:"unit_price" binding:"required,gt=0"`
				Memo      string  `json:"memo"`
			} `json:"items" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		orderNo := generateOrderNumber(db, uint(bookID), "SAL", req.Date)

		tx := db.Begin()

		var totalAmount float64
		var orderItems []models.SalesOrderItem
		for _, item := range req.Items {
			amount := item.Quantity * item.UnitPrice
			totalAmount += amount
			orderItems = append(orderItems, models.SalesOrderItem{
				GoodsID:    item.GoodsID,
				Quantity:   item.Quantity,
				UnitPrice:  item.UnitPrice,
				Amount:     amount,
				Memo:       item.Memo,
			})
		}

		order := models.SalesOrder{
			BookID:       uint(bookID),
			OrderNo:      orderNo,
			CustomerID:   req.CustomerID,
			WarehouseID:  req.WarehouseID,
			Date:         req.Date,
			Status:       "draft",
			TotalAmount:  totalAmount,
			PaymentTerm:  req.PaymentTerm,
			Memo:         req.Memo,
			PreparedBy:   getCurrentUser(c),
		}
		if err := tx.Create(&order).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for i := range orderItems {
			orderItems[i].SalesOrderID = order.ID
		}
		if err := tx.Create(&orderItems).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"data": order})
	}
}

func getSales(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := c.Param("sid")
		var order models.SalesOrder
		if err := db.Preload("Items").First(&order, sid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "销售单不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": order})
	}
}

func postSales(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := c.Param("sid")
		var order models.SalesOrder
		if err := db.Preload("Items").First(&order, sid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "销售单不存在"})
			return
		}
		if order.Status != "draft" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只能审核草稿状态的销售单"})
			return
		}

		tx := db.Begin()

		// 1. Calculate cost and record stock flows
		var totalAmount, totalCost float64
		for _, item := range order.Items {
			amount := item.Quantity * item.UnitPrice
			totalAmount += amount

			// Calculate weighted average cost
			costPrice := calcWeightedAvgCost(db, order.BookID, item.GoodsID, order.WarehouseID)
			costAmount := item.Quantity * costPrice
			totalCost += costAmount

			flow := models.StockFlow{
				BookID:      order.BookID,
				GoodsID:     item.GoodsID,
				WarehouseID: order.WarehouseID,
				FlowType:    "sales_out",
				Quantity:    item.Quantity,
				UnitPrice:   costPrice,
				Amount:      costAmount,
				RefType:     "sales",
				RefID:       order.ID,
				Date:        order.Date,
				Memo:        fmt.Sprintf("销售出库 %s", order.OrderNo),
			}
			if err := tx.Create(&flow).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// Update item cost
			tx.Model(&models.SalesOrderItem{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
				"cost_price":  costPrice,
				"cost_amount": costAmount,
			})
		}

		// 2. Generate voucher 1: Revenue (Debit AR, Credit Revenue)
		var arAccount models.Account
		db.Where("book_id = ? AND code = ?", order.BookID, "1122").First(&arAccount)

		// Flexible revenue account: try 6001 → 5001 → name match → code LIKE 5%/6%
		var revenueAccount models.Account
		db.Where("book_id = ? AND code = ?", order.BookID, "6001").First(&revenueAccount)
		if revenueAccount.ID == 0 {
			db.Where("book_id = ? AND code = ?", order.BookID, "5001").First(&revenueAccount)
		}
		if revenueAccount.ID == 0 {
			db.Where("book_id = ? AND name LIKE ?", order.BookID, "%主营%收入%").First(&revenueAccount)
		}
		if revenueAccount.ID == 0 {
			db.Where("book_id = ? AND code LIKE ?", order.BookID, "5%").Order("code").First(&revenueAccount)
		}

		voucher1 := models.Voucher{
			BookID:      order.BookID,
			Date:        order.Date,
			Number:      generateVoucherNumber(tx, order.BookID, order.Date),
			VoucherType: "sales",
			Status:      "posted",
			TotalDebit:  totalAmount,
			TotalCredit: totalAmount,
			Memo:        fmt.Sprintf("销售出库 %s", order.OrderNo),
			PreparedBy:  "system",
			ReviewedBy:  "system",
			PostedBy:    "system",
		}
		if err := tx.Create(&voucher1).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建收入凭证失败: %v", err)})
			return
		}

		vi1 := models.VoucherItem{
			VoucherID:      voucher1.ID,
			LineNo:         1,
			AccountID:      arAccount.ID,
			AccountCode:    arAccount.Code,
			AccountName:    arAccount.Name,
			Debit:          totalAmount,
			Memo:           "应收账款",
			AuxCustomerID:  &order.CustomerID,
		}
		vi2 := models.VoucherItem{
			VoucherID:   voucher1.ID,
			LineNo:      2,
			AccountID:   revenueAccount.ID,
			AccountCode: revenueAccount.Code,
			AccountName: revenueAccount.Name,
			Credit:      totalAmount,
			Memo:        "主营业务收入",
		}
		tx.Create(&vi1)
		tx.Create(&vi2)
		updateAccountBalances(tx, &voucher1)

		// 3. Generate voucher 2: Cost (Debit Cost, Credit Inventory)
		// Flexible cost account: try 6401 → 5401 → name match → code LIKE 5%/6%
		var costAccount models.Account
		db.Where("book_id = ? AND code = ?", order.BookID, "6401").First(&costAccount)
		if costAccount.ID == 0 {
			db.Where("book_id = ? AND code = ?", order.BookID, "5401").First(&costAccount)
		}
		if costAccount.ID == 0 {
			db.Where("book_id = ? AND name LIKE ?", order.BookID, "%主营%成本%").First(&costAccount)
		}
		if costAccount.ID == 0 {
			db.Where("book_id = ? AND code LIKE ?", order.BookID, "5%").Order("code DESC").First(&costAccount)
		}

		var invAccount models.Account
		db.Where("book_id = ? AND code = ?", order.BookID, "1405").First(&invAccount)

		voucher2 := models.Voucher{
			BookID:      order.BookID,
			Date:        order.Date,
			Number:      generateVoucherNumber(tx, order.BookID, order.Date),
			VoucherType: "sales_cost",
			Status:      "posted",
			TotalDebit:  totalCost,
			TotalCredit: totalCost,
			Memo:        fmt.Sprintf("结转成本 %s", order.OrderNo),
			PreparedBy:  "system",
			ReviewedBy:  "system",
			PostedBy:    "system",
		}
		if err := tx.Create(&voucher2).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建成本凭证失败: %v", err)})
			return
		}

		vi3 := models.VoucherItem{
			VoucherID:   voucher2.ID,
			LineNo:      1,
			AccountID:   costAccount.ID,
			AccountCode: costAccount.Code,
			AccountName: costAccount.Name,
			Debit:       totalCost,
			Memo:        "主营业务成本",
		}
		vi4 := models.VoucherItem{
			VoucherID:      voucher2.ID,
			LineNo:         2,
			AccountID:      invAccount.ID,
			AccountCode:    invAccount.Code,
			AccountName:    invAccount.Name,
			Credit:         totalCost,
			Memo:           "库存商品",
			AuxWarehouseID: &order.WarehouseID,
		}
		tx.Create(&vi3)
		tx.Create(&vi4)
		updateAccountBalances(tx, &voucher2)

		// 4. Update order status
		tx.Model(&order).Updates(map[string]interface{}{
			"status":          "posted",
			"total_amount":    totalAmount,
			"cost_amount":     totalCost,
			"ref_voucher_ids": fmt.Sprintf("[%d,%d]", voucher1.ID, voucher2.ID),
		})

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"message": "审核成功", "voucher_ids": []uint{voucher1.ID, voucher2.ID}})
	}
}

func voidSales(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := c.Param("sid")
		var order models.SalesOrder
		if err := db.First(&order, sid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "销售单不存在"})
			return
		}
		if order.Status == "voided" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "销售单已作废"})
			return
		}

		tx := db.Begin()

		if order.Status == "posted" {
			// Reverse stock flows
			tx.Where("ref_type = ? AND ref_id = ?", "sales", order.ID).Delete(&models.StockFlow{})

			// Reverse vouchers (revenue + cost)
			if order.RefVoucherIDs != "" {
				var voucherIDs []uint
				fmt.Sscanf(order.RefVoucherIDs, "[%d,%d]", &voucherIDs)
				for _, vid := range voucherIDs {
					if vid > 0 {
						tx.Where("voucher_id = ?", vid).Delete(&models.VoucherItem{})
						tx.Delete(&models.Voucher{}, vid)
					}
				}
			}
		}

		tx.Model(&order).Update("status", "voided")
		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"message": "作废成功"})
	}
}

// ==================== 收付款单 ====================

func listPayments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var records []models.PaymentRecord
		query := db.Where("book_id = ?", bookID)
		if typ := c.Query("type"); typ != "" {
			query = query.Where("type = ?", typ)
		}
		if status := c.Query("status"); status != "" {
			query = query.Where("status = ?", status)
		}
		query.Order("date DESC, id DESC").Find(&records)
		c.JSON(http.StatusOK, gin.H{"data": records})
	}
}

func createPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var req struct {
			Type             string  `json:"type" binding:"required"`
			CounterpartyType string  `json:"counterparty_type" binding:"required"`
			CounterpartyID   uint    `json:"counterparty_id" binding:"required"`
			BankAccountID    uint    `json:"bank_account_id"`
			Date             string  `json:"date" binding:"required"`
			Amount           float64 `json:"amount" binding:"required,gt=0"`
			Method           string  `json:"method"`
			Memo             string  `json:"memo"`
			Details          []struct {
				SourceType string  `json:"source_type"`
				SourceID   uint    `json:"source_id"`
				Amount     float64 `json:"amount"`
			} `json:"details"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		recordNo := generateOrderNumber(db, uint(bookID), "PAY", req.Date)
		if req.Type == "receipt" {
			recordNo = generateOrderNumber(db, uint(bookID), "RCV", req.Date)
		}

		tx := db.Begin()

		record := models.PaymentRecord{
			BookID:           uint(bookID),
			RecordNo:         recordNo,
			Type:             req.Type,
			CounterpartyType: req.CounterpartyType,
			CounterpartyID:   req.CounterpartyID,
			BankAccountID:    req.BankAccountID,
			Date:             req.Date,
			Amount:           req.Amount,
			Method:           req.Method,
			Status:           "draft",
			Memo:             req.Memo,
			PreparedBy:       getCurrentUser(c),
		}
		if err := tx.Create(&record).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, d := range req.Details {
			if d.Amount > 0 {
				detail := models.PaymentRecordDetail{
					PaymentRecordID: record.ID,
					SourceType:      d.SourceType,
					SourceID:        d.SourceID,
					Amount:          d.Amount,
				}
				tx.Create(&detail)
			}
		}

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"data": record})
	}
}

func getPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		pid := c.Param("pid")
		var record models.PaymentRecord
		if err := db.Preload("Details").First(&record, pid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "收付款单不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": record})
	}
}

func postPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		pid := c.Param("pid")
		var record models.PaymentRecord
		if err := db.Preload("Details").First(&record, pid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "收付款单不存在"})
			return
		}
		if record.Status != "draft" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只能审核草稿状态的收付款单"})
			return
		}

		tx := db.Begin()

		// Get accounts
		var bankAccount models.Account
		db.Where("book_id = ? AND code = ?", record.BookID, "1002").First(&bankAccount)

		var arAccount, apAccount models.Account
		db.Where("book_id = ? AND code = ?", record.BookID, "1122").First(&arAccount)
		db.Where("book_id = ? AND code = ?", record.BookID, "2202").First(&apAccount)

		voucher := models.Voucher{
			BookID:      record.BookID,
			Date:        record.Date,
			Number:      generateVoucherNumber(tx, record.BookID, record.Date),
			VoucherType: record.Type,
			Status:      "posted",
			TotalDebit:  record.Amount,
			TotalCredit: record.Amount,
			Memo:        fmt.Sprintf("收付款 %s", record.RecordNo),
			PreparedBy:  "system",
			ReviewedBy:  "system",
			PostedBy:    "system",
		}
		tx.Create(&voucher)

		if record.Type == "receipt" {
			// Receipt: Debit Bank, Credit AR
			vi1 := models.VoucherItem{
				VoucherID:   voucher.ID,
				LineNo:      1,
				AccountID:   bankAccount.ID,
				AccountCode: bankAccount.Code,
				AccountName: bankAccount.Name,
				Debit:       record.Amount,
				Memo:        "银行存款",
			}
			vi2 := models.VoucherItem{
				VoucherID:      voucher.ID,
				LineNo:         2,
				AccountID:      arAccount.ID,
				AccountCode:    arAccount.Code,
				AccountName:    arAccount.Name,
				Credit:         record.Amount,
				Memo:           "应收账款",
				AuxCustomerID:  &record.CounterpartyID,
			}
			tx.Create(&vi1)
			tx.Create(&vi2)
		} else {
			// Payment: Debit AP, Credit Bank
			vi1 := models.VoucherItem{
				VoucherID:      voucher.ID,
				LineNo:         1,
				AccountID:      apAccount.ID,
				AccountCode:    apAccount.Code,
				AccountName:    apAccount.Name,
				Debit:          record.Amount,
				Memo:           "应付账款",
				AuxSupplierID:  &record.CounterpartyID,
			}
			vi2 := models.VoucherItem{
				VoucherID:   voucher.ID,
				LineNo:      2,
				AccountID:   bankAccount.ID,
				AccountCode: bankAccount.Code,
				AccountName: bankAccount.Name,
				Credit:      record.Amount,
				Memo:        "银行存款",
			}
			tx.Create(&vi1)
			tx.Create(&vi2)
		}

		updateAccountBalances(tx, &voucher)

		tx.Model(&record).Updates(map[string]interface{}{
			"status":         "posted",
			"ref_voucher_id": voucher.ID,
		})

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"message": "审核成功", "voucher_id": voucher.ID})
	}
}

func voidPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		pid := c.Param("pid")
		var record models.PaymentRecord
		if err := db.First(&record, pid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "收付款单不存在"})
			return
		}
		if record.Status == "voided" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "收付款单已作废"})
			return
		}

		tx := db.Begin()

		if record.Status == "posted" {
			// Reverse voucher
			if record.RefVoucherID > 0 {
				tx.Where("voucher_id = ?", record.RefVoucherID).Delete(&models.VoucherItem{})
				tx.Delete(&models.Voucher{}, record.RefVoucherID)
			}
		}

		tx.Model(&record).Update("status", "voided")
		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"message": "作废成功"})
	}
}

// ==================== 库存报表 ====================

func stockSummary(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		keyword := c.Query("keyword")
		var results []struct {
			GoodsID      uint    `json:"goods_id"`
			Code         string  `json:"code"`
			Name         string  `json:"name"`
			Unit         string  `json:"unit"`
			WarehouseID  uint    `json:"warehouse_id"`
			WarehouseName string `json:"warehouse_name"`
			InQty        float64 `json:"in_qty"`
			InAmount     float64 `json:"in_amount"`
			OutQty       float64 `json:"out_qty"`
			OutAmount    float64 `json:"out_amount"`
			ClosingQty   float64 `json:"quantity"`
			UnitCost     float64 `json:"unit_cost"`
			TotalCost    float64 `json:"total_cost"`
		}

		query := db.Model(&models.StockFlow{}).
			Select(`stock_flows.goods_id, g.code, g.name, g.unit,
				stock_flows.warehouse_id,
				COALESCE(ai.name, '') as warehouse_name,
				SUM(CASE WHEN stock_flows.flow_type IN ('purchase_in','transfer_in','adjust_in','initial') THEN stock_flows.quantity ELSE 0 END) as in_qty,
				SUM(CASE WHEN stock_flows.flow_type IN ('purchase_in','transfer_in','adjust_in','initial') THEN stock_flows.amount ELSE 0 END) as in_amount,
				SUM(CASE WHEN stock_flows.flow_type IN ('sales_out','transfer_out','adjust_out') THEN stock_flows.quantity ELSE 0 END) as out_qty,
				SUM(CASE WHEN stock_flows.flow_type IN ('sales_out','transfer_out','adjust_out') THEN stock_flows.amount ELSE 0 END) as out_amount,
				SUM(CASE WHEN stock_flows.flow_type IN ('purchase_in','transfer_in','adjust_in','initial') THEN stock_flows.quantity ELSE 0 END) -
				SUM(CASE WHEN stock_flows.flow_type IN ('sales_out','transfer_out','adjust_out') THEN stock_flows.quantity ELSE 0 END) as quantity,
				SUM(CASE WHEN stock_flows.flow_type IN ('purchase_in','transfer_in','adjust_in','initial') THEN stock_flows.amount ELSE 0 END) -
				SUM(CASE WHEN stock_flows.flow_type IN ('sales_out','transfer_out','adjust_out') THEN stock_flows.amount ELSE 0 END) as total_cost`).
			Joins("JOIN goods g ON g.id = stock_flows.goods_id").
			Joins("LEFT JOIN aux_items ai ON ai.id = stock_flows.warehouse_id AND ai.type = 'warehouse'").
			Where("stock_flows.book_id = ?", bookID)

		if keyword != "" {
			query = query.Where("g.code LIKE ? OR g.name LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}

		query.Group("stock_flows.goods_id, stock_flows.warehouse_id").
			Order("g.code").
			Scan(&results)

		// Calculate unit cost
		for i := range results {
			if results[i].ClosingQty > 0 {
				results[i].UnitCost = results[i].TotalCost / results[i].ClosingQty
			}
		}

		c.JSON(http.StatusOK, gin.H{"data": results})
	}
}

func stockFlowList(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		var flows []models.StockFlow
		query := db.Where("book_id = ?", bookID)
		if goodsID := c.Query("goods_id"); goodsID != "" {
			query = query.Where("goods_id = ?", goodsID)
		}
		if warehouseID := c.Query("warehouse_id"); warehouseID != "" {
			query = query.Where("warehouse_id = ?", warehouseID)
		}
		if dateFrom := c.Query("date_from"); dateFrom != "" {
			query = query.Where("date >= ?", dateFrom)
		}
		if dateTo := c.Query("date_to"); dateTo != "" {
			query = query.Where("date <= ?", dateTo)
		}
		query.Order("date DESC, id DESC").Find(&flows)
		c.JSON(http.StatusOK, gin.H{"data": flows})
	}
}

func lowStockAlert(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")

		type StockResult struct {
			GoodsID    uint    `json:"goods_id"`
			GoodsCode  string  `json:"goods_code"`
			GoodsName  string  `json:"goods_name"`
			Unit       string  `json:"unit"`
			MinStock   float64 `json:"min_stock"`
			ClosingQty float64 `json:"closing_qty"`
		}

		var results []StockResult
		db.Model(&models.StockFlow{}).
			Select(`stock_flows.goods_id, g.code as goods_code, g.name as goods_name, g.unit, g.min_stock,
				SUM(CASE WHEN stock_flows.flow_type IN ('purchase_in','transfer_in','adjust_in','initial') THEN stock_flows.quantity ELSE 0 END) -
				SUM(CASE WHEN stock_flows.flow_type IN ('sales_out','transfer_out','adjust_out') THEN stock_flows.quantity ELSE 0 END) as closing_qty`).
			Joins("JOIN goods g ON g.id = stock_flows.goods_id").
			Where("stock_flows.book_id = ? AND g.min_stock > 0", bookID).
			Group("stock_flows.goods_id").
			Having("closing_qty < g.min_stock").
			Order("closing_qty ASC").
			Scan(&results)

		c.JSON(http.StatusOK, gin.H{"data": results})
	}
}

// ==================== 采购/销售报表 ====================

func purchaseReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", time.Now().Format("2006-01"))

		type PurchaseResult struct {
			SupplierID   uint    `json:"supplier_id"`
			SupplierName string  `json:"supplier_name"`
			TotalAmount  float64 `json:"total_amount"`
			OrderCount   int64   `json:"order_count"`
		}

		var results []PurchaseResult
		db.Model(&models.PurchaseOrder{}).
			Select(`purchase_orders.supplier_id, a.name as supplier_name,
				SUM(purchase_orders.total_amount) as total_amount,
				COUNT(*) as order_count`).
			Joins("LEFT JOIN aux_items a ON a.id = purchase_orders.supplier_id").
			Where("purchase_orders.book_id = ? AND purchase_orders.status = ? AND purchase_orders.date LIKE ?",
				bookID, "posted", period+"%").
			Group("purchase_orders.supplier_id").
			Order("total_amount DESC").
			Scan(&results)

		c.JSON(http.StatusOK, gin.H{"data": results, "period": period})
	}
}

func salesReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", time.Now().Format("2006-01"))

		type SalesResult struct {
			CustomerID   uint    `json:"customer_id"`
			CustomerName string  `json:"customer_name"`
			TotalAmount  float64 `json:"total_amount"`
			CostAmount   float64 `json:"cost_amount"`
			GrossProfit  float64 `json:"gross_profit"`
			Margin       float64 `json:"margin"`
			OrderCount   int64   `json:"order_count"`
		}

		var results []SalesResult
		db.Model(&models.SalesOrder{}).
			Select(`sales_orders.customer_id, a.name as customer_name,
				SUM(sales_orders.total_amount) as total_amount,
				SUM(sales_orders.cost_amount) as cost_amount,
				SUM(sales_orders.total_amount) - SUM(sales_orders.cost_amount) as gross_profit,
				CASE WHEN SUM(sales_orders.total_amount) > 0
					THEN (SUM(sales_orders.total_amount) - SUM(sales_orders.cost_amount)) / SUM(sales_orders.total_amount) * 100
					ELSE 0 END as margin,
				COUNT(*) as order_count`).
			Joins("LEFT JOIN aux_items a ON a.id = sales_orders.customer_id").
			Where("sales_orders.book_id = ? AND sales_orders.status = ? AND sales_orders.date LIKE ?",
				bookID, "posted", period+"%").
			Group("sales_orders.customer_id").
			Order("total_amount DESC").
			Scan(&results)

		c.JSON(http.StatusOK, gin.H{"data": results, "period": period})
	}
}

func marginReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID := c.Param("id")
		period := c.DefaultQuery("period", time.Now().Format("2006-01"))

		type MarginResult struct {
			GoodsID      uint    `json:"goods_id"`
			GoodsCode    string  `json:"goods_code"`
			GoodsName    string  `json:"goods_name"`
			TotalQty     float64 `json:"total_qty"`
			TotalAmount  float64 `json:"total_amount"`
			CostAmount   float64 `json:"cost_amount"`
			GrossProfit  float64 `json:"gross_profit"`
			Margin       float64 `json:"margin"`
		}

		var results []MarginResult
		db.Model(&models.SalesOrderItem{}).
			Select(`soi.goods_id, g.code as goods_code, g.name as goods_name,
				SUM(soi.quantity) as total_qty,
				SUM(soi.amount) as total_amount,
				SUM(soi.cost_amount) as cost_amount,
				SUM(soi.amount) - SUM(soi.cost_amount) as gross_profit,
				CASE WHEN SUM(soi.amount) > 0
					THEN (SUM(soi.amount) - SUM(soi.cost_amount)) / SUM(soi.amount) * 100
					ELSE 0 END as margin`).
			Joins("JOIN sales_orders so ON so.id = soi.sales_order_id").
			Joins("JOIN goods g ON g.id = soi.goods_id").
			Where("so.book_id = ? AND so.status = ? AND so.date LIKE ?",
				bookID, "posted", period+"%").
			Group("soi.goods_id").
			Order("total_amount DESC").
			Scan(&results)

		c.JSON(http.StatusOK, gin.H{"data": results, "period": period})
	}
}

// ==================== 辅助函数 ====================

func generateOrderNumber(db *gorm.DB, bookID uint, prefix string, date string) string {
	period := ""
	if len(date) >= 7 {
		period = date[:7]
	}
	var maxNo string
	var scan string

	db.Model(&models.PurchaseOrder{}).
		Where("book_id = ? AND order_no LIKE ?", bookID, prefix+"-"+period+"-%").
		Select("COALESCE(MAX(order_no), '')").
		Row().Scan(&scan)
	if scan > maxNo {
		maxNo = scan
	}
	db.Model(&models.SalesOrder{}).
		Where("book_id = ? AND order_no LIKE ?", bookID, prefix+"-"+period+"-%").
		Select("COALESCE(MAX(order_no), '')").
		Row().Scan(&scan)
	if scan > maxNo {
		maxNo = scan
	}
	db.Model(&models.PaymentRecord{}).
		Where("book_id = ? AND record_no LIKE ?", bookID, prefix+"-"+period+"-%").
		Select("COALESCE(MAX(record_no), '')").
		Row().Scan(&scan)
	if scan > maxNo {
		maxNo = scan
	}

	seq := 0
	if maxNo != "" {
		idx := strings.LastIndex(maxNo, "-")
		if idx >= 0 {
			fmt.Sscanf(maxNo[idx+1:], "%d", &seq)
		}
	}
	return fmt.Sprintf("%s-%s-%03d", prefix, period, seq+1)
}

func calcWeightedAvgCost(db *gorm.DB, bookID uint, goodsID uint, warehouseID uint) float64 {
	type CostResult struct {
		TotalIn     float64
		TotalInAmt  float64
		TotalOut    float64
		TotalOutAmt float64
	}
	var result CostResult
	db.Model(&models.StockFlow{}).
		Select(`SUM(CASE WHEN flow_type IN ('purchase_in','transfer_in','adjust_in','initial') THEN quantity ELSE 0 END) as total_in,
			SUM(CASE WHEN flow_type IN ('purchase_in','transfer_in','adjust_in','initial') THEN amount ELSE 0 END) as total_in_amt,
			SUM(CASE WHEN flow_type IN ('sales_out','transfer_out','adjust_out') THEN quantity ELSE 0 END) as total_out,
			SUM(CASE WHEN flow_type IN ('sales_out','transfer_out','adjust_out') THEN amount ELSE 0 END) as total_out_amt`).
		Where("book_id = ? AND goods_id = ? AND warehouse_id = ?", bookID, goodsID, warehouseID).
		Scan(&result)

	closingQty := result.TotalIn - result.TotalOut
	if closingQty <= 0 {
		return 0
	}
	closingAmt := result.TotalInAmt - result.TotalOutAmt
	return math.Round(closingAmt/closingQty*10000) / 10000
}

func getCurrentUser(c *gin.Context) string {
	if user, exists := c.Get("username"); exists {
		if username, ok := user.(string); ok {
			return username
		}
	}
	return "system"
}
