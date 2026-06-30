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
			// 只读接口
			bookScoped.GET("/accounts", listAccounts(db))
			bookScoped.GET("/accounts/tree", getAccountTree(db))
			bookScoped.GET("/vouchers", listVouchers(db))
			bookScoped.GET("/vouchers/:vid", getVoucher(db))
			bookScoped.GET("/vouchers/export", exportVouchers(db))
			bookScoped.GET("/voucher-templates", listVoucherTemplates(db))
			bookScoped.GET("/ledger/journal", journal(db))
			bookScoped.GET("/ledger/multi-column", multiColumnLedger(db))
			bookScoped.GET("/reports/balance-sheet", balanceSheet(db))
			bookScoped.GET("/reports/income-statement", incomeStatement(db))
			bookScoped.GET("/reports/cash-flow", cashFlowStatement(db))
			bookScoped.GET("/reports/account-balance", accountBalanceReport(db))
			bookScoped.GET("/reports/export", exportReport(db))
			bookScoped.GET("/reports/templates", listReportTemplates(db))
			bookScoped.GET("/reports/custom/:rid", customReport(db))
			bookScoped.GET("/reports/income-statement-v2", incomeStatementEnhanced(db))
			bookScoped.GET("/reports/expense", expenseReport(db))
			bookScoped.GET("/reports/general-ledger", generalLedgerReport(db))
			bookScoped.GET("/reports/ar-ap", arApReport(db))
			bookScoped.GET("/assets/categories", listAssetCategories(db))
			bookScoped.GET("/assets", listAssetCards(db))
			bookScoped.GET("/assets/:cardId", getAssetCard(db))
			bookScoped.GET("/assets/depreciation/calc", calcDepreciation(db))
			bookScoped.GET("/assets/summary", assetSummary(db))
			bookScoped.GET("/assets/transactions/:cardId", listAssetTransactions(db))
			bookScoped.GET("/assets/transactions", listAllAssetTransactions(db))
			bookScoped.GET("/assets/export", exportAssets(db))
			bookScoped.GET("/aux/:type", listAuxItems(db))
			bookScoped.GET("/aux/:type/export", exportAuxItems(db))
			bookScoped.GET("/closing/status", closingStatus(db))
			bookScoped.GET("/users", listBookUsers(db))
		}

		// 需要写入权限的接口
		bookWritable := auth.Group("/books/:id")
		bookWritable.Use(middleware.BookAccess(db))
		bookWritable.Use(middleware.BookWritable())
		{
			// 科目
			bookWritable.POST("/accounts", createAccount(db))
			bookWritable.PUT("/accounts/:acid", updateAccount(db))
			bookWritable.DELETE("/accounts/:acid", deleteAccount(db))
			bookWritable.POST("/accounts/dedup", func(c *gin.Context) {
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
			bookWritable.POST("/vouchers", createVoucher(db))
			bookWritable.PUT("/vouchers/:vid", updateVoucher(db))
			bookWritable.DELETE("/vouchers/:vid", deleteVoucher(db))
			bookWritable.POST("/vouchers/:vid/review", reviewVoucher(db))
			bookWritable.POST("/vouchers/:vid/unreview", unreviewVoucher(db))
			bookWritable.POST("/vouchers/:vid/post", postVoucher(db))
			bookWritable.POST("/vouchers/:vid/unpost", unpostVoucher(db))
			bookWritable.POST("/vouchers/:vid/void", voidVoucher(db))
			bookWritable.POST("/vouchers/:vid/restore", restoreVoucher(db))
			bookWritable.POST("/vouchers/batch-review", batchReview(db))
			bookWritable.POST("/vouchers/batch-post", batchPost(db))

			// 凭证模板
			bookWritable.POST("/voucher-templates", createVoucherTemplate(db))
			bookWritable.PUT("/voucher-templates/:tid", updateVoucherTemplate(db))
			bookWritable.DELETE("/voucher-templates/:tid", deleteVoucherTemplate(db))

			// 报表模板
			bookWritable.POST("/reports/templates", createReportTemplate(db))
			bookWritable.PUT("/reports/templates/:tid", updateReportTemplate(db))
			bookWritable.DELETE("/reports/templates/:tid", deleteReportTemplate(db))

			// 设备管理
			bookWritable.POST("/assets/categories", createAssetCategory(db))
			bookWritable.PUT("/assets/categories/:aid", updateAssetCategory(db))
			bookWritable.DELETE("/assets/categories/:aid", deleteAssetCategory(db))
			bookWritable.POST("/assets", createAssetCard(db))
			bookWritable.PUT("/assets/:cardId", updateAssetCard(db))
			bookWritable.DELETE("/assets/:cardId", deleteAssetCard(db))
			bookWritable.POST("/assets/depreciation/run", runDepreciation(db))
			bookWritable.PUT("/assets/:cardId/status", changeAssetStatus(db))
			bookWritable.POST("/assets/import", importAssets(db))

			// 辅助核算
			bookWritable.POST("/aux/:type", createAuxItem(db))
			bookWritable.PUT("/aux/:type/:aid", updateAuxItem(db))
			bookWritable.DELETE("/aux/:type/:aid", deleteAuxItem(db))
			bookWritable.POST("/aux/:type/import", importAuxItems(db))
			bookWritable.POST("/aux/:type/batch-delete", batchDeleteAuxItems(db))

			// 期末处理
			bookWritable.POST("/closing/auto-transfer", autoTransfer(db))
			bookWritable.POST("/closing/close", closePeriod(db))
			bookWritable.POST("/closing/unclose", unclosePeriod(db))

			// 账套权限管理
			bookWritable.POST("/users", addBookUser(db))
			bookWritable.PUT("/users/:buid", updateBookUser(db))
			bookWritable.DELETE("/users/:buid", deleteBookUser(db))
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
