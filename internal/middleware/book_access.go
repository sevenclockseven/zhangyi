package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/models"
)

func BookAccess(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role == "admin" {
			c.Next()
			return
		}

		bookId := c.Param("id")
		if bookId == "" {
			bookId = c.Param("bookId")
		}
		userId, _ := c.Get("user_id")

		var bu models.BookUser
		if err := db.Where("book_id = ? AND user_id = ?", bookId, userId).First(&bu).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此账套"})
			c.Abort()
			return
		}

		c.Set("book_role", bu.Role)
		c.Next()
	}
}

func BookWritable() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role == "admin" {
			c.Next()
			return
		}

		bookRole, exists := c.Get("book_role")
		if !exists || bookRole == "readonly" {
			c.JSON(http.StatusForbidden, gin.H{"error": "只读权限，无法操作"})
			c.Abort()
			return
		}
		c.Next()
	}
}
