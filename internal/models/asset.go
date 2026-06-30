package models

import "time"

// 资产分类
type AssetCategory struct {
	ID                   uint    `json:"id" gorm:"primaryKey"`
	ParentID             *uint   `json:"parent_id" gorm:"index"`
	Name                 string  `json:"name" gorm:"size:100;not null"`
	Code                 string  `json:"code" gorm:"size:20"`
	Method               string  `json:"method" gorm:"size:20;default:'straight_line'"` // straight_line: 直线法
	UsefulLifeMonths     int     `json:"useful_life_months" gorm:"default:60"`           // 预计使用年限（月）
	ResidualValueRate    float64 `json:"residual_value_rate" gorm:"type:decimal(5,4);default:0.05"` // 预计净残值率 5%
	BookAccountID        uint    `json:"book_account_id"`                                // 固定资产科目
	DepreciationAccountID uint   `json:"depreciation_account_id"`                         // 累计折旧科目
	ExpenseAccountID     uint    `json:"expense_account_id"`                              // 折旧费用科目
	Memo                 string  `json:"memo" gorm:"type:text"`
	Created            time.Time `json:"created_at"`
	Updated            time.Time `json:"updated_at"`
}

// 资产卡片
type AssetCard struct {
	ID                    uint      `json:"id" gorm:"primaryKey"`
	Code                  string    `json:"code" gorm:"size:30;not null;index"` // 资产编号
	Name                  string    `json:"name" gorm:"size:100;not null"`
	SpecModel             string    `json:"spec_model" gorm:"size:100"`          // 规格型号
	SerialNumber          string    `json:"serial_number" gorm:"size:100"`        // 出厂序列号
	CategoryID            uint      `json:"category_id" gorm:"index;not null"`
	OriginalValue         float64   `json:"original_value" gorm:"type:decimal(14,2);not null"`
	AccumulatedDepreciation float64 `json:"accumulated_depreciation" gorm:"type:decimal(14,2);default:0"`
	NetValue              float64   `json:"net_value" gorm:"type:decimal(14,2)"`   // 净值=原值-累计折旧
	ResidualValue         float64   `json:"residual_value" gorm:"type:decimal(14,2)"` // 预计净残值
	Status                string    `json:"status" gorm:"size:20;default:'in_use';index"` // in_use/idle/scrapped/maintenance
	BookID                uint      `json:"book_id" gorm:"index;not null"`
	DepartmentID         *uint     `json:"department_id" gorm:"index"`           // 使用部门(关联aux_items)
	EmployeeID           *uint     `json:"employee_id" gorm:"index"`              // 责任人(关联aux_items)
	Department           string    `json:"department" gorm:"size:50"`             // 部门名称(冗余显示)
	EmployeeName         string    `json:"employee_name" gorm:"size:30"`          // 人员姓名(冗余显示)
	Location              string    `json:"location" gorm:"size:200"`               // 存放地点
	AcquisitionDate       string    `json:"acquisition_date" gorm:"size:10"`        // 取得日期 YYYY-MM-DD
	DepreciationStartMonth string   `json:"depreciation_start_month" gorm:"size:7"` // 折旧起始月 YYYY-MM
	UsefulLifeMonths      int       `json:"useful_life_months"`                     // 使用年限（月），可从分类继承
	ResidualValueRate     float64   `json:"residual_value_rate" gorm:"type:decimal(5,4);default:0.05"`
	MonthlyDepreciation   float64   `json:"monthly_depreciation" gorm:"type:decimal(14,2);default:0"` // 月折旧额
	Source                string    `json:"source" gorm:"size:30"`                  // purchase/donate/transfer/self_made
	Vendor                string    `json:"vendor" gorm:"size:100"`
	InvoiceNo             string    `json:"invoice_no" gorm:"size:50"`
	Remark                string    `json:"remark" gorm:"type:text"`
	Created               time.Time `json:"created_at"`
	Updated               time.Time `json:"updated_at"`
}

// 资产变动流水
type AssetTransaction struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	CardID       uint      `json:"card_id" gorm:"index;not null"`
	Type         string    `json:"type" gorm:"size:20;not null"`  // acquire/depreciate/maintenance/scrap/transfer/idle
	Date         string    `json:"date" gorm:"size:10"`            // YYYY-MM-DD
	AmountBefore float64   `json:"amount_before" gorm:"type:decimal(14,2)"`
	AmountAfter  float64   `json:"amount_after" gorm:"type:decimal(14,2)"`
	Note         string    `json:"note" gorm:"type:text"`
	Operator     string    `json:"operator" gorm:"size:30"`
	RefVoucherID uint      `json:"ref_voucher_id"` // 关联凭证
	Created      time.Time `json:"created_at"`
}

// 折旧明细（每月一行）
type AssetDepreciation struct {
	ID         uint    `json:"id" gorm:"primaryKey"`
	CardID     uint    `json:"card_id" gorm:"index;not null"`
	Period     string  `json:"period" gorm:"size:7;not null;index"` // YYYY-MM
	StartNet   float64 `json:"start_net" gorm:"type:decimal(14,2)"`  // 期初净值
	Amount     float64 `json:"amount" gorm:"type:decimal(14,2)"`      // 本月折旧额
	EndNet     float64 `json:"end_net" gorm:"type:decimal(14,2)"`     // 期末净值
	VoucherID  uint    `json:"voucher_id"`                           // 关联折旧凭证
}
