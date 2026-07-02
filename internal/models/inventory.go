package models

import "time"

// Goods 商品档案
type Goods struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	BookID             uint      `json:"book_id" gorm:"index;not null"`
	Code               string    `json:"code" gorm:"size:20;not null"`
	Name               string    `json:"name" gorm:"size:100;not null"`
	Category           string    `json:"category" gorm:"size:50"`
	Unit               string    `json:"unit" gorm:"size:20;not null"`
	Barcode            string    `json:"barcode" gorm:"size:50"`
	CostMethod         string    `json:"cost_method" gorm:"size:20;default:weighted_avg"`
	TaxRate            float64   `json:"tax_rate" gorm:"type:decimal(5,2);default:0"`
	PurchaseAccountID  uint      `json:"purchase_account_id"`
	SalesAccountID     uint      `json:"sales_account_id"`
	InventoryAccountID uint      `json:"inventory_account_id"`
	RefPrice           float64   `json:"ref_price" gorm:"type:decimal(14,2);default:0"`
	MinStock           float64   `json:"min_stock" gorm:"type:decimal(14,4);default:0"`
	IsActive           bool      `json:"is_active" gorm:"default:true"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// StockFlow 库存流水
type StockFlow struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	BookID       uint      `json:"book_id" gorm:"index;not null"`
	GoodsID      uint      `json:"goods_id" gorm:"index;not null"`
	WarehouseID  uint      `json:"warehouse_id" gorm:"index;not null"`
	FlowType     string    `json:"flow_type" gorm:"size:20;not null"` // purchase_in/sales_out/transfer_in/transfer_out/adjust_in/adjust_out/initial
	Quantity     float64   `json:"quantity" gorm:"type:decimal(14,4);not null"`
	UnitPrice    float64   `json:"unit_price" gorm:"type:decimal(14,4);not null"`
	Amount       float64   `json:"amount" gorm:"type:decimal(14,2);not null"`
	RefType      string    `json:"ref_type" gorm:"size:20"`
	RefID        uint      `json:"ref_id"`
	RefVoucherID uint      `json:"ref_voucher_id"`
	Memo         string    `json:"memo" gorm:"size:200"`
	Date         string    `json:"date" gorm:"size:10;not null"`
	CreatedAt    time.Time `json:"created_at"`
}

// PurchaseOrder 采购入库单
type PurchaseOrder struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	BookID       uint      `json:"book_id" gorm:"index;not null"`
	OrderNo      string    `json:"order_no" gorm:"size:30;not null;uniqueIndex:idx_purchase_order_no,priority:2"`
	SupplierID   uint      `json:"supplier_id" gorm:"index;not null"`
	WarehouseID  uint      `json:"warehouse_id" gorm:"index;not null"`
	Date         string    `json:"date" gorm:"size:10;not null"`
	Status       string    `json:"status" gorm:"size:20;default:draft"`
	TotalAmount  float64   `json:"total_amount" gorm:"type:decimal(14,2);default:0"`
	PaymentTerm  string    `json:"payment_term" gorm:"size:20;default:credit"`
	RefVoucherID uint      `json:"ref_voucher_id"`
	Memo         string    `json:"memo" gorm:"size:200"`
	PreparedBy   string    `json:"prepared_by" gorm:"size:50"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Items []PurchaseOrderItem `json:"items" gorm:"foreignKey:PurchaseOrderID"`
}

// PurchaseOrderItem 采购入库单明细
type PurchaseOrderItem struct {
	ID              uint    `json:"id" gorm:"primaryKey"`
	PurchaseOrderID uint    `json:"purchase_order_id" gorm:"index;not null"`
	GoodsID         uint    `json:"goods_id" gorm:"not null"`
	Quantity        float64 `json:"quantity" gorm:"type:decimal(14,4);not null"`
	UnitPrice       float64 `json:"unit_price" gorm:"type:decimal(14,4);not null"`
	Amount          float64 `json:"amount" gorm:"type:decimal(14,2);not null"`
	TaxAmount       float64 `json:"tax_amount" gorm:"type:decimal(14,2);default:0"`
	Memo            string  `json:"memo" gorm:"size:200"`
}

// SalesOrder 销售出库单
type SalesOrder struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	BookID        uint      `json:"book_id" gorm:"index;not null"`
	OrderNo       string    `json:"order_no" gorm:"size:30;not null;uniqueIndex:idx_sales_order_no,priority:2"`
	CustomerID    uint      `json:"customer_id" gorm:"index;not null"`
	WarehouseID   uint      `json:"warehouse_id" gorm:"index;not null"`
	Date          string    `json:"date" gorm:"size:10;not null"`
	Status        string    `json:"status" gorm:"size:20;default:draft"`
	TotalAmount   float64   `json:"total_amount" gorm:"type:decimal(14,2);default:0"`
	CostAmount    float64   `json:"cost_amount" gorm:"type:decimal(14,2);default:0"`
	PaymentTerm   string    `json:"payment_term" gorm:"size:20;default:credit"`
	RefVoucherIDs string    `json:"ref_voucher_ids" gorm:"type:text"`
	Memo          string    `json:"memo" gorm:"size:200"`
	PreparedBy    string    `json:"prepared_by" gorm:"size:50"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	Items []SalesOrderItem `json:"items" gorm:"foreignKey:SalesOrderID"`
}

// SalesOrderItem 销售出库单明细
type SalesOrderItem struct {
	ID           uint    `json:"id" gorm:"primaryKey"`
	SalesOrderID uint    `json:"sales_order_id" gorm:"index;not null"`
	GoodsID      uint    `json:"goods_id" gorm:"not null"`
	Quantity     float64 `json:"quantity" gorm:"type:decimal(14,4);not null"`
	UnitPrice    float64 `json:"unit_price" gorm:"type:decimal(14,4);not null"`
	Amount       float64 `json:"amount" gorm:"type:decimal(14,2);not null"`
	TaxAmount    float64 `json:"tax_amount" gorm:"type:decimal(14,2);default:0"`
	CostPrice    float64 `json:"cost_price" gorm:"type:decimal(14,4);default:0"`
	CostAmount   float64 `json:"cost_amount" gorm:"type:decimal(14,2);default:0"`
	Memo         string  `json:"memo" gorm:"size:200"`
}

// PaymentRecord 收付款单
type PaymentRecord struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	BookID           uint      `json:"book_id" gorm:"index;not null"`
	RecordNo         string    `json:"record_no" gorm:"size:30;not null;uniqueIndex:idx_payment_record_no,priority:2"`
	Type             string    `json:"type" gorm:"size:20;not null"` // receipt/payment
	CounterpartyType string    `json:"counterparty_type" gorm:"size:20;not null"` // customer/supplier
	CounterpartyID   uint      `json:"counterparty_id" gorm:"index;not null"`
	BankAccountID    uint      `json:"bank_account_id"`
	Date             string    `json:"date" gorm:"size:10;not null"`
	Amount           float64   `json:"amount" gorm:"type:decimal(14,2);not null"`
	Method           string    `json:"method" gorm:"size:20;default:bank"`
	Status           string    `json:"status" gorm:"size:20;default:draft"`
	RefVoucherID     uint      `json:"ref_voucher_id"`
	Memo             string    `json:"memo" gorm:"size:200"`
	PreparedBy       string    `json:"prepared_by" gorm:"size:50"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	Details []PaymentRecordDetail `json:"details" gorm:"foreignKey:PaymentRecordID"`
}

// PaymentRecordDetail 收付款核销明细
type PaymentRecordDetail struct {
	ID              uint    `json:"id" gorm:"primaryKey"`
	PaymentRecordID uint    `json:"payment_record_id" gorm:"index;not null"`
	SourceType      string  `json:"source_type" gorm:"size:20;not null"` // purchase/sales
	SourceID        uint    `json:"source_id" gorm:"not null"`
	Amount          float64 `json:"amount" gorm:"type:decimal(14,2);not null"`
}

// StockSnapshot 库存快照
type StockSnapshot struct {
	ID          uint    `json:"id" gorm:"primaryKey"`
	BookID      uint    `json:"book_id" gorm:"index;not null"`
	GoodsID     uint    `json:"goods_id" gorm:"index;not null"`
	WarehouseID uint    `json:"warehouse_id" gorm:"index;not null"`
	Period      string  `json:"period" gorm:"size:7;not null"`
	OpeningQty  float64 `json:"opening_qty" gorm:"type:decimal(14,4);default:0"`
	InQty       float64 `json:"in_qty" gorm:"type:decimal(14,4);default:0"`
	OutQty      float64 `json:"out_qty" gorm:"type:decimal(14,4);default:0"`
	ClosingQty  float64 `json:"closing_qty" gorm:"type:decimal(14,4);default:0"`
	ClosingAmt  float64 `json:"closing_amt" gorm:"type:decimal(14,2);default:0"`
	UnitCost    float64 `json:"unit_cost" gorm:"type:decimal(14,4);default:0"`
}
