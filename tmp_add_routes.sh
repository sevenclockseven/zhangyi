#!/bin/bash
cd /opt/zhangyi

# Add new routes to assets group
# Note: we place /transactions routes BEFORE /:cardId to avoid route conflicts
sed -i '/assets.GET("\/summary", assetSummary(db))/a\\n			// 资产变动\n			assets.PUT("/:cardId/status", changeAssetStatus(db))\n			assets.GET("/transactions/:cardId", listAssetTransactions(db))\n			assets.GET("/transactions", listAllAssetTransactions(db))\n			// 资产导入导出\n			assets.POST("/import", importAssets(db))\n			assets.GET("/export", exportAssets(db))' internal/api/routes.go

echo "Routes added successfully"

