package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/models"
	"github.com/sevenclockseven/zhangyi/internal/services"
)

// ===== Auth =====

func loginHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请输入用户名和密码"})
			return
		}

		user, token, err := services.Login(db, req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Get book permissions for non-admin users
		var permissions []gin.H
		if user.Role != "admin" {
			var bookUsers []models.BookUser
			db.Where("user_id = ?", user.ID).Find(&bookUsers)
			for _, bu := range bookUsers {
				permissions = append(permissions, gin.H{"book_id": bu.BookID, "role": bu.Role})
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"id":        user.ID,
				"username":  user.Username,
				"real_name": user.RealName,
				"role":      user.Role,
			},
			"book_permissions": permissions,
		})
	}
}

func registerHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
			RealName string `json:"real_name"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hash, err := services.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
			return
		}

		user := models.User{
			Username: req.Username,
			Password: hash,
			RealName: req.RealName,
			Role:     "user",
			Status:   "active",
		}

		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已存在"})
			return
		}

		token, _ := services.GenerateToken(&user)
		c.JSON(http.StatusCreated, gin.H{"token": token, "user": gin.H{"id": user.ID, "username": user.Username, "real_name": user.RealName, "role": user.Role}})
	}
}

func getMeHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": user.ID, "username": user.Username, "real_name": user.RealName, "role": user.Role, "last_login": user.LastLogin})
	}
}

func changePasswordHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		var req struct {
			OldPassword string `json:"old_password" binding:"required"`
			NewPassword string `json:"new_password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user models.User
		db.First(&user, userID)
		if !services.CheckPassword(req.OldPassword, user.Password) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "原密码错误"})
			return
		}

		hash, _ := services.HashPassword(req.NewPassword)
		db.Model(&user).Update("password", hash)
		c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
	}
}

// ===== User Management (Admin) =====

func listUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.User
		db.Order("created_at DESC").Find(&users)
		result := []gin.H{}
		for _, u := range users {
			result = append(result, gin.H{"id": u.ID, "username": u.Username, "real_name": u.RealName, "role": u.Role, "status": u.Status, "last_login": u.LastLogin, "created_at": u.CreatedAt})
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

func createUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
			RealName string `json:"real_name"`
			Role     string `json:"role"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Role == "" {
			req.Role = "user"
		}

		hash, _ := services.HashPassword(req.Password)
		user := models.User{Username: req.Username, Password: hash, RealName: req.RealName, Role: req.Role, Status: "active"}
		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已存在"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": gin.H{"id": user.ID, "username": user.Username}})
	}
}

func updateUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")
		var user models.User
		if err := db.First(&user, uid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		var req struct {
			RealName string `json:"real_name"`
			Role     string `json:"role"`
			Status   string `json:"status"`
		}
		c.ShouldBindJSON(&req)
		updates := map[string]interface{}{}
		if req.RealName != "" {
			updates["real_name"] = req.RealName
		}
		if req.Role != "" {
			updates["role"] = req.Role
		}
		if req.Status != "" {
			updates["status"] = req.Status
		}
		db.Model(&user).Updates(updates)
		c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
	}
}

func deleteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")
		currentUserID, _ := c.Get("user_id")
		if fmt.Sprintf("%v", currentUserID) == uid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己"})
			return
		}
		db.Delete(&models.User{}, uid)
		c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
	}
}

func resetPassword(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")
		var req struct {
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		hash, _ := services.HashPassword(req.Password)
		db.Model(&models.User{}).Where("id = ?", uid).Update("password", hash)
		c.JSON(http.StatusOK, gin.H{"message": "密码重置成功"})
	}
}
