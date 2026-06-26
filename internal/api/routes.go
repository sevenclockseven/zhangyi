package api

import (
	"net/http"

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

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
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
			c.JSON(http.StatusOK, gin.H{"status": "ok", "name": "账易", "version": AppVersion})
		})
		api.POST("/auth/login", loginHandler(db))
		api.POST("/auth/register", registerHandler(db))

		// 需要登录的接口
		auth := api.Group("")
		auth.Use(middleware.AuthRequired())
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

			// 科目
			accounts := auth.Group("/books/:id/accounts")
			{
				accounts.GET("", listAccounts(db))
				accounts.GET("/tree", getAccountTree(db))
				accounts.POST("", createAccount(db))
				accounts.PUT("/:acid", updateAccount(db))
				accounts.DELETE("/:acid", deleteAccount(db))
				accounts.POST("/dedup", func(c *gin.Context) {
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
			}

			// 凭证
			vouchers := auth.Group("/books/:id/vouchers")
			{
				vouchers.GET("", listVouchers(db))
				vouchers.POST("", createVoucher(db))
				vouchers.GET("/:vid", getVoucher(db))
				vouchers.PUT("/:vid", updateVoucher(db))
				vouchers.DELETE("/:vid", deleteVoucher(db))
				vouchers.POST("/:vid/review", reviewVoucher(db))
				vouchers.POST("/:vid/unreview", unreviewVoucher(db))
				vouchers.POST("/:vid/post", postVoucher(db))
				vouchers.POST("/:vid/unpost", unpostVoucher(db))
				vouchers.POST("/:vid/void", voidVoucher(db))
				vouchers.POST("/:vid/restore", restoreVoucher(db))
				vouchers.POST("/batch-review", batchReview(db))
				vouchers.POST("/batch-post", batchPost(db))
				vouchers.GET("/export", exportVouchers(db))
			}

			// 凭证模板
			tpls := auth.Group("/books/:id/voucher-templates")
			{
			tpls.GET("", listVoucherTemplates(db))
			tpls.POST("", createVoucherTemplate(db))
			tpls.PUT("/:tid", updateVoucherTemplate(db))
			tpls.DELETE("/:tid", deleteVoucherTemplate(db))
			}

			// 账簿查询
			books.GET("/:id/ledger/journal", journal(db))
			books.GET("/:id/ledger/multi-column", multiColumnLedger(db))

			// 报表
			reports := auth.Group("/books/:id/reports")
			{
				reports.GET("/balance-sheet", balanceSheet(db))
				reports.GET("/income-statement", incomeStatement(db))
				reports.GET("/cash-flow", cashFlowStatement(db))
				reports.GET("/account-balance", accountBalanceReport(db))
				reports.GET("/export", exportReport(db))
			reports.GET("/templates", listReportTemplates(db))
			reports.POST("/templates", createReportTemplate(db))
			reports.PUT("/templates/:tid", updateReportTemplate(db))
			reports.DELETE("/templates/:tid", deleteReportTemplate(db))
			reports.GET("/custom/:rid", customReport(db))
			reports.GET("/income-statement-v2", incomeStatementEnhanced(db))
			reports.GET("/expense", expenseReport(db))
			reports.GET("/general-ledger", generalLedgerReport(db))
			reports.GET("/ar-ap", arApReport(db))
			}

			// 辅助核算
			aux := auth.Group("/books/:id/aux/:type")
			{
				aux.GET("", listAuxItems(db))
				aux.POST("", createAuxItem(db))
				aux.PUT("/:aid", updateAuxItem(db))
				aux.DELETE("/:aid", deleteAuxItem(db))
			aux.GET("/export", exportAuxItems(db))
			aux.POST("/import", importAuxItems(db))
			aux.POST("/batch-delete", batchDeleteAuxItems(db))
			}

			// 期末处理
			closing := auth.Group("/books/:id/closing")
			{
				closing.POST("/auto-transfer", autoTransfer(db))
				closing.POST("/close", closePeriod(db))
				closing.POST("/unclose", unclosePeriod(db))
				closing.GET("/status", closingStatus(db))
			}
		}
	}
}
