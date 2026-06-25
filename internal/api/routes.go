package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
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
		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "name": "账易", "version": "0.1.0"})
		})

		// Account books (账套)
		books := api.Group("/books")
		{
			books.GET("", listBooks(db))
			books.POST("", createBook(db))
			books.GET("/:id", getBook(db))
			books.PUT("/:id", updateBook(db))
			books.DELETE("/:id", deleteBook(db))
			books.POST("/:id/sync-template", syncTemplate(db))
		}

		// Accounts (科目)
		accounts := api.Group("/books/:id/accounts")
		{
			accounts.GET("", listAccounts(db))
			accounts.GET("/tree", getAccountTree(db))
			accounts.POST("", createAccount(db))
			accounts.PUT("/:acid", updateAccount(db))
			accounts.DELETE("/:acid", deleteAccount(db))
		}

		// Vouchers (凭证)
		vouchers := api.Group("/books/:id/vouchers")
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

		// Reports (报表)
		reports := api.Group("/books/:id/reports")
		{
			reports.GET("/balance-sheet", balanceSheet(db))
			reports.GET("/income-statement", incomeStatement(db))
			reports.GET("/account-balance", accountBalanceReport(db))
		}

		// Auxiliary accounting (辅助核算)
		aux := api.Group("/books/:id/aux/:type")
		{
			aux.GET("", listAuxItems(db))
			aux.POST("", createAuxItem(db))
			aux.PUT("/:aid", updateAuxItem(db))
			aux.DELETE("/:aid", deleteAuxItem(db))
		}
	}
}
