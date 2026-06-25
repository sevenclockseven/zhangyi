package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/models"
)

// JWT密钥 - 生产环境应从环境变量读取
var JWTSecret = []byte("zhangyi-jwt-secret-2026")

// Claims JWT声明
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// HashPassword 密码加密
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken 生成JWT
func GenerateToken(user *models.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// ParseToken 解析JWT
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

// Login 登录
func Login(db *gorm.DB, username, password string) (*models.User, string, error) {
	var user models.User
	if err := db.Where("username = ? AND status = ?", username, "active").First(&user).Error; err != nil {
		return nil, "", errors.New("用户名或密码错误")
	}

	if !CheckPassword(password, user.Password) {
		return nil, "", errors.New("用户名或密码错误")
	}

	token, err := GenerateToken(&user)
	if err != nil {
		return nil, "", errors.New("生成token失败")
	}

	// 更新最后登录时间
	db.Model(&user).Update("last_login", time.Now())

	return &user, token, nil
}

// InitAdmin 初始化管理员账号
func InitAdmin(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count == 0 {
		hash, _ := HashPassword("admin123")
		admin := models.User{
			Username: "admin",
			Password: hash,
			RealName: "管理员",
			Role:     "admin",
			Status:   "active",
		}
		db.Create(&admin)
	}
}
