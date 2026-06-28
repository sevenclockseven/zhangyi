package models

import (
	"time"
)

// AccountBook 账套
type AccountBook struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Code         string    `json:"code" gorm:"uniqueIndex;size:20;not null"`
	Name         string    `json:"name" gorm:"size:100;not null"`
	Industry     string    `json:"industry" gorm:"size:200"` // comma separated
	TaxpayerType     string    `json:"taxpayer_type" gorm:"size:20"`
	AccountingStandard string    `json:"accounting_standard" gorm:"size:30"` // small_business / enterprise
	StartDate        string    `json:"start_date" gorm:"size:7;not null"` // YYYY-MM
	Currency     string    `json:"currency" gorm:"size:10;default:CNY"`
	Status       string    `json:"status" gorm:"size:20;default:active"`
	Contact      string    `json:"contact" gorm:"size:50"`
	Phone        string    `json:"phone" gorm:"size:20"`
	Address      string    `json:"address" gorm:"size:200"`
	Memo         string    `json:"memo"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Account 科目
type Account struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	BookID       uint      `json:"book_id" gorm:"index;not null"`
	Code         string    `json:"code" gorm:"size:20;not null"`
	Name         string    `json:"name" gorm:"size:100;not null"`
	ParentCode   string    `json:"parent_code" gorm:"size:20"`
	Direction    string    `json:"direction" gorm:"size:4;not null"` // debit / credit
	Level        int       `json:"level" gorm:"default:1"`
	IsLeaf       bool      `json:"is_leaf" gorm:"default:true"`
	IsSystem     bool      `json:"is_system" gorm:"default:false"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	AuxTypes     string    `json:"aux_types" gorm:"size:200"` // comma separated
	Memo         string    `json:"memo"`
	SortOrder    int       `json:"sort_order" gorm:"default:0"`
	CreatedAt    time.Time `json:"created_at"`

	Book AccountBook `json:"book,omitempty" gorm:"foreignKey:BookID"`
}

// Voucher 凭证
type Voucher struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	BookID      uint      `json:"book_id" gorm:"index;not null"`
	Date        string    `json:"date" gorm:"size:10;not null"` // YYYY-MM-DD
	Number      string    `json:"number" gorm:"size:20;not null"`
	VoucherType string    `json:"voucher_type" gorm:"size:20;default:general"`
	Status      string    `json:"status" gorm:"size:20;default:draft"`
	TotalDebit  float64   `json:"total_debit" gorm:"type:decimal(14,2);default:0"`
	TotalCredit float64   `json:"total_credit" gorm:"type:decimal(14,2);default:0"`
	Attachments int       `json:"attachments" gorm:"default:0"`
	PreparedBy  string    `json:"prepared_by" gorm:"size:50"`
	ReviewedBy  string    `json:"reviewed_by" gorm:"size:50"`
	PostedBy    string    `json:"posted_by" gorm:"size:50"`
	Memo        string    `json:"memo"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Book  AccountBook   `json:"book,omitempty" gorm:"foreignKey:BookID"`
	Items []VoucherItem `json:"items,omitempty" gorm:"foreignKey:VoucherID"`
}

// VoucherItem 凭证明细
type VoucherItem struct {
	ID                uint    `json:"id" gorm:"primaryKey"`
	VoucherID         uint    `json:"voucher_id" gorm:"index;not null"`
	LineNo            int     `json:"line_no" gorm:"not null"`
	AccountID         uint    `json:"account_id" gorm:"not null"`
	AccountCode       string  `json:"account_code" gorm:"size:20;not null"`
	AccountName       string  `json:"account_name" gorm:"size:100;not null"`
	Debit             float64 `json:"debit" gorm:"type:decimal(14,2);default:0"`
	Credit            float64 `json:"credit" gorm:"type:decimal(14,2);default:0"`
	Memo              string  `json:"memo" gorm:"size:200"`
	AuxCustomerID     *uint   `json:"aux_customer_id"`
	AuxSupplierID     *uint   `json:"aux_supplier_id"`
	AuxDepartmentID   *uint   `json:"aux_department_id"`
	AuxProjectID      *uint   `json:"aux_project_id"`
	AuxEmployeeID     *uint   `json:"aux_employee_id"`
	AuxWarehouseID    *uint   `json:"aux_warehouse_id"`
	AuxBankAccountID  *uint   `json:"aux_bank_account_id"`
	AuxFixedAssetID   *uint   `json:"aux_fixed_asset_id"`
	AuxVatDetailID    *uint   `json:"aux_vat_detail_id"`
	AuxCostObjectID   *uint   `json:"aux_cost_object_id"`
	CashFlowID        *uint   `json:"cash_flow_id"`

	Voucher Voucher `json:"voucher,omitempty" gorm:"foreignKey:VoucherID"`
	Account Account `json:"account,omitempty" gorm:"foreignKey:AccountID"`
}

// OpeningBalance 期初余额
type OpeningBalance struct {
	ID        uint    `json:"id" gorm:"primaryKey"`
	BookID    uint    `json:"book_id" gorm:"index;not null"`
	AccountID uint    `json:"account_id" gorm:"not null"`
	Period    string  `json:"period" gorm:"size:7;not null"` // YYYY-MM
	Debit     float64 `json:"debit" gorm:"type:decimal(14,2);default:0"`
	Credit    float64 `json:"credit" gorm:"type:decimal(14,2);default:0"`
	AuxKey    string  `json:"aux_key" gorm:"size:200"`
}

// AccountBalance 科目余额（记账时自动更新）
type AccountBalance struct {
	ID            uint    `json:"id" gorm:"primaryKey"`
	BookID        uint    `json:"book_id" gorm:"index;not null"`
	AccountID     uint    `json:"account_id" gorm:"not null"`
	Period        string  `json:"period" gorm:"size:7;not null"`
	OpeningDebit  float64 `json:"opening_debit" gorm:"type:decimal(14,2);default:0"`
	OpeningCredit float64 `json:"opening_credit" gorm:"type:decimal(14,2);default:0"`
	PeriodDebit   float64 `json:"period_debit" gorm:"type:decimal(14,2);default:0"`
	PeriodCredit  float64 `json:"period_credit" gorm:"type:decimal(14,2);default:0"`
	YTDDebit      float64 `json:"ytd_debit" gorm:"type:decimal(14,2);default:0"`
	YTDCredit     float64 `json:"ytd_credit" gorm:"type:decimal(14,2);default:0"`
	ClosingDebit  float64 `json:"closing_debit" gorm:"type:decimal(14,2);default:0"`
	ClosingCredit float64 `json:"closing_credit" gorm:"type:decimal(14,2);default:0"`
	AuxKey        string  `json:"aux_key" gorm:"size:200"`
}

// AuxItem 辅助核算项
type AuxItem struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BookID    uint      `json:"book_id" gorm:"index;not null"`
	Type      string    `json:"type" gorm:"size:20;not null"` // customer/supplier/department/project/employee/warehouse/bank_account/cash_flow/fixed_asset/vax_detail/cost_object
	Code      string    `json:"code" gorm:"size:20;not null"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	ParentID  *uint     `json:"parent_id"`
	Extra     string    `json:"extra" gorm:"type:text"` // JSON
	IsActive  bool      `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
}

// VoucherTemplate 凭证模板
type VoucherTemplate struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BookID    *uint     `json:"book_id"` // nil = global template
	Name      string    `json:"name" gorm:"size:100;not null"`
	Category  string    `json:"category" gorm:"size:50"`
	Items     string    `json:"items" gorm:"type:text;not null"` // JSON
	CreatedAt time.Time `json:"created_at"`
}

// ReportTemplate 报表模板
type ReportTemplate struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BookID    *uint     `json:"book_id"` // nil = global template
	Name      string    `json:"name" gorm:"size:100;not null"`
	Type      string    `json:"type" gorm:"size:20;not null"` // standard / custom
	Config    string    `json:"config" gorm:"type:text;not null"` // JSON
	CreatedAt time.Time `json:"created_at"`
}

// BookUser 账套权限

type BookUser struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BookID    uint      `json:"book_id" gorm:"index;not null"`
	UserID    uint      `json:"user_id" gorm:"index;not null"`
	Role      string    `json:"role" gorm:"size:20;default:readonly"` // admin / writable / readonly
	CreatedAt time.Time `json:"created_at"`

	Book AccountBook `json:"book,omitempty" gorm:"foreignKey:BookID"`
	User User         `json:"user,omitempty" gorm:"foreignKey:UserID"` // User defined in user.go
}

// OperationLog 操作日志
type OperationLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BookID    uint      `json:"book_id" gorm:"index;not null"`
	Module    string    `json:"module" gorm:"size:20;not null"`
	Action    string    `json:"action" gorm:"size:20;not null"`
	TargetID  *uint     `json:"target_id"`
	Detail    string    `json:"detail"`
	Operator  string    `json:"operator" gorm:"size:50"`
	CreatedAt time.Time `json:"created_at"`
}
