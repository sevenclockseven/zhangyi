package api

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/models"
	"github.com/sevenclockseven/zhangyi/internal/middleware"
	"github.com/sevenclockseven/zhangyi/internal/services"
)

// AppVersion is set by main.go at startup
var AppVersion = "0.5.3"

// RegisterRoutes registers all API routes
func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	// 初始化管理员
	services.InitAdmin(db)

	// CORS middleware - 白名单
	allowedOrigins := map[string]bool{
		"http://localhost:8080":  true,
		"http://localhost:3000": true,
	}
	// 从环境变量加载额外的白名单
	if envOrigins := os.Getenv("CORS_ORIGINS"); envOrigins != "" {
		for _, o := range strings.Split(envOrigins, ",") {
			allowedOrigins[strings.TrimSpace(o)] = true
		}
	}
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		// 公开接口（不需要登录）
		api.GET("/templates/versions", templateVersions(db))
		api.GET("/templates/manifest", getTemplateManifest(db))
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "name": "易记", "version": AppVersion})
		})
		api.POST("/auth/login", middleware.LoginRateLimit(), loginHandler(db))
		// 注册功能已禁用，用户只能由管理员创建

		// 需要登录的接口
		auth := api.Group("")
		auth.Use(middleware.AuthRequired())
		auth.Use(middleware.AuditLog(db))
		{
			// 用户信息
			auth.GET("/auth/me", getMeHandler(db))
			auth.PUT("/auth/password", changePasswordHandler(db))

			// 用户管理（管理员）
			users := auth.Group("/users")
			users.Use(middleware.AdminRequired())
			{
				users.GET("", listUsers(db))
				users.POST("", createUser(db))
				users.PUT("/:uid", updateUser(db))
				users.DELETE("/:uid", deleteUser(db))
				users.PUT("/:uid/reset-password", resetPassword(db))
			}

		// 账套
		books := auth.Group("/books")
		{
			books.GET("", listBooks(db))
			books.POST("", createBook(db))
			books.GET("/:id", getBook(db))
			books.PUT("/:id", updateBook(db))
			books.DELETE("/:id", deleteBook(db))
			books.POST("/:id/sync-template", syncTemplate(db))
			books.POST("/:id/sync-all-templates", syncAllTemplates(db))
			books.GET("/:id/trial-balance", trialBalance(db))
		books.GET("/:id/opening-balances", getOpeningBalances(db))
		books.POST("/:id/opening-balances", saveOpeningBalances(db))
		books.GET("/:id/opening-balances/export", exportOpeningBalances(db))
		books.POST("/:id/opening-balances/import", importOpeningBalances(db))
		
		}

		// 需要账套权限的接口
		bookScoped := auth.Group("/books/:id")
		bookScoped.Use(middleware.BookAccess(db))
		{
			// 科目
			bookScoped.GET("/accounts", listAccounts(db))
			bookScoped.GET("/accounts/tree", getAccountTree(db))
			bookScoped.POST("/accounts", createAccount(db))
			bookScoped.PUT("/accounts/:acid", updateAccount(db))
			bookScoped.DELETE("/accounts/:acid", deleteAccount(db))
			bookScoped.POST("/accounts/dedup", func(c *gin.Context) {
				bookID := c.Param("id")
				var accounts []models.Account
				db.Where("book_id = ?", bookID).Order("id").Find(&accounts)
				seen := make(map[string]bool)
				var toDelete []uint
				for _, a := range accounts {
					if seen[a.Code] {
						toDelete = append(toDelete, a.ID)
					} else {
						seen[a.Code] = true
					}
				}
				if len(toDelete) > 0 {
					db.Where("id IN ?", toDelete).Delete(&models.Account{})
				}
				c.JSON(http.StatusOK, gin.H{"deleted": len(toDelete), "remaining": len(accounts) - len(toDelete)})
			})

			// 凭证
			bookScoped.GET("/vouchers", listVouchers(db))
			bookScoped.POST("/vouchers", createVoucher(db))
			bookScoped.GET("/vouchers/:vid", getVoucher(db))
			bookScoped.PUT("/vouchers/:vid", updateVoucher(db))
			bookScoped.DELETE("/vouchers/:vid", deleteVoucher(db))
			bookScoped.POST("/vouchers/:vid/review", reviewVoucher(db))
			bookScoped.POST("/vouchers/:vid/unreview", unreviewVoucher(db))
			bookScoped.POST("/vouchers/:vid/post", postVoucher(db))
			bookScoped.POST("/vouchers/:vid/unpost", unpostVoucher(db))
			bookScoped.POST("/vouchers/:vid/void", voidVoucher(db))
			bookScoped.POST("/vouchers/:vid/restore", restoreVoucher(db))
			bookScoped.POST("/vouchers/batch-review", batchReview(db))
			bookScoped.POST("/vouchers/batch-post", batchPost(db))
			bookScoped.GET("/vouchers/export", exportVouchers(db))

			// 凭证模板
			bookScoped.GET("/voucher-templates", listVoucherTemplates(db))
			bookScoped.POST("/voucher-templates", createVoucherTemplate(db))
			bookScoped.PUT("/voucher-templates/:tid", updateVoucherTemplate(db))
			bookScoped.DELETE("/voucher-templates/:tid", deleteVoucherTemplate(db))

			// 账簿查询
			bookScoped.GET("/ledger/journal", journal(db))
			bookScoped.GET("/ledger/multi-column", multiColumnLedger(db))

			// 报表
			bookScoped.GET("/reports/balance-sheet", balanceSheet(db))
			bookScoped.GET("/reports/income-statement", incomeStatement(db))
			bookScoped.GET("/reports/cash-flow", cashFlowStatement(db))
			bookScoped.GET("/reports/account-balance", accountBalanceReport(db))
			bookScoped.GET("/reports/export", exportReport(db))
			bookScoped.GET("/reports/templates", listReportTemplates(db))
			bookScoped.POST("/reports/templates", createReportTemplate(db))
			bookScoped.PUT("/reports/templates/:tid", updateReportTemplate(db))
			bookScoped.DELETE("/reports/templates/:tid", deleteReportTemplate(db))
			bookScoped.GET("/reports/custom/:rid", customReport(db))
			bookScoped.GET("/reports/income-statement-v2", incomeStatementEnhanced(db))
			bookScoped.GET("/reports/expense", expenseReport(db))
			bookScoped.GET("/reports/general-ledger", generalLedgerReport(db))
			bookScoped.GET("/reports/ar-ap", arApReport(db))

			// 设备管理
			bookScoped.GET("/assets/categories", listAssetCategories(db))
			bookScoped.POST("/assets/categories", createAssetCategory(db))
			bookScoped.PUT("/assets/categories/:aid", updateAssetCategory(db))
			bookScoped.DELETE("/assets/categories/:aid", deleteAssetCategory(db))
			bookScoped.GET("/assets", listAssetCards(db))
			bookScoped.POST("/assets", createAssetCard(db))
			bookScoped.GET("/assets/:cardId", getAssetCard(db))
			bookScoped.PUT("/assets/:cardId", updateAssetCard(db))
			bookScoped.DELETE("/assets/:cardId", deleteAssetCard(db))
			bookScoped.GET("/assets/depreciation/calc", calcDepreciation(db))
			bookScoped.POST("/assets/depreciation/run", runDepreciation(db))
			bookScoped.GET("/assets/summary", assetSummary(db))
			bookScoped.PUT("/assets/:cardId/status", changeAssetStatus(db))
			bookScoped.GET("/assets/transactions/:cardId", listAssetTransactions(db))
			bookScoped.GET("/assets/transactions", listAllAssetTransactions(db))
			bookScoped.POST("/assets/import", importAssets(db))
			bookScoped.GET("/assets/export", exportAssets(db))

			// 辅助核算
			bookScoped.GET("/aux/:type", listAuxItems(db))
			bookScoped.POST("/aux/:type", createAuxItem(db))
			bookScoped.PUT("/aux/:type/:aid", updateAuxItem(db))
			bookScoped.DELETE("/aux/:type/:aid", deleteAuxItem(db))
			bookScoped.GET("/aux/:type/export", exportAuxItems(db))
			bookScoped.POST("/aux/:type/import", importAuxItems(db))
			bookScoped.POST("/aux/:type/batch-delete", batchDeleteAuxItems(db))

			// 期末处理
			bookScoped.POST("/closing/auto-transfer", autoTransfer(db))
			bookScoped.POST("/closing/close", closePeriod(db))
			bookScoped.POST("/closing/unclose", unclosePeriod(db))
			bookScoped.GET("/closing/status", closingStatus(db))

			// 账套权限管理
			bookScoped.GET("/users", listBookUsers(db))
			bookScoped.POST("/users", addBookUser(db))
			bookScoped.PUT("/users/:buid", updateBookUser(db))
			bookScoped.DELETE("/users/:buid", deleteBookUser(db))
		}

			// 系统级接口（备份、日志）
			auth.GET("/system/backups", listBackups(db))
			auth.POST("/system/backups", createBackup(db))
			auth.GET("/system/backups/:name", downloadBackup(db))
			auth.DELETE("/system/backups/:name", deleteBackup(db))
			auth.POST("/system/backups/:name/restore", restoreBackup(db))
			auth.GET("/system/logs", listOperationLogs(db))
		}
	}
}
