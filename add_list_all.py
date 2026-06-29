#!/usr/bin/env python3
path = '/opt/zhangyi/internal/api/asset_transaction_handlers.go'
with open(path, 'r') as f:
    content = f.read()

marker = '''func listAssetTransactions(db *gorm.DB) gin.HandlerFunc {
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
}'''

addition = '''

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
}'''

if marker in content:
    content = content.replace(marker, marker + addition, 1)
    with open(path, 'w') as f:
        f.write(content)
    print("OK: listAllAssetTransactions added")
else:
    print("ERROR: marker not found")

