#!/usr/bin/env python3
import sys

with open('/opt/zhangyi/internal/api/asset_handlers.go', 'r') as f:
    content = f.read()

# The old code ends with the c.JSON return inside runDepreciation
old_marker = '''		c.JSON(http.StatusOK, gin.H{
			"data":   results,
			"period": period,
			"total":  totalAmount,
			"count":  len(results),
		})
	}
}

// ========== 资产台账报表 =========='''

if old_marker not in content:
    print("ERROR: could not find insertion point in asset_handlers.go")
    sys.exit(1)

replacement = '''		// 生成折旧凭证
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

// ========== 资产台账报表 =========='''

content = content.replace(old_marker, replacement, 1)

with open('/opt/zhangyi/internal/api/asset_handlers.go', 'w') as f:
    f.write(content)

print("OK: depreciation voucher generation added")

