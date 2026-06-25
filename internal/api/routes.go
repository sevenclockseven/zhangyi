package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/middleware"
	"github.com/sevenclockseven/zhangyi/internal/services"
)

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
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "name": "账易", "version": "0.2.0"})
		})
		api.POST("/auth/login", loginHandler(db))
		api.POST("/auth/register", registerHandler(db)) // 首次注册用，之后关闭

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
				books.GET("/:id/trial-balance", trialBalance(db))
			}

			// 科目
			accounts := auth.Group("/books/:id/accounts")
			{
				accounts.GET("", listAccounts(db))
				accounts.GET("/tree", getAccountTree(db))
				accounts.POST("", createAccount(db))
				accounts.PUT("/:acid", updateAccount(db))
				accounts.DELETE("/:acid", deleteAccount(db))
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
				vouchers.POST("/batch-review", batchReview(db))
				vouchers.POST("/batch-post", batchPost(db))
			}

			// 报表
			reports := auth.Group("/books/:id/reports")
			{
				reports.GET("/balance-sheet", balanceSheet(db))
				reports.GET("/income-statement", incomeStatement(db))
				reports.GET("/account-balance", accountBalanceReport(db))
			}

			// 辅助核算
			aux := auth.Group("/books/:id/aux/:type")
			{
				aux.GET("", listAuxItems(db))
				aux.POST("", createAuxItem(db))
				aux.PUT("/:aid", updateAuxItem(db))
				aux.DELETE("/:aid", deleteAuxItem(db))
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
