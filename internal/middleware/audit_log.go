package middleware

import (
	"bytes"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/models"
)

func AuditLog(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		c.Next()

		path := c.Request.URL.Path
		module := extractModule(path)
		action := extractAction(c.Request.Method, path)
		username := ""
		if u, exists := c.Get("username"); exists {
			username = u.(string)
		}

		var bookId uint
		bookIdStr := c.Param("id")
		if bookIdStr == "" {
			bookIdStr = c.Param("bookId")
		}
		if bookIdStr != "" {
			for _, ch := range bookIdStr {
				if ch >= '0' && ch <= '9' {
					bookId = bookId*10 + uint(ch-'0')
				}
			}
		}

		var targetId *uint
		for _, param := range []string{"vid", "acid", "uid", "buid", "tid", "aid"} {
			if v := c.Param(param); v != "" {
				var id uint
				for _, ch := range v {
					if ch >= '0' && ch <= '9' {
						id = id*10 + uint(ch-'0')
					}
				}
				targetId = &id
				break
			}
		}

		detail := path

		go func() {
			db.Create(&models.OperationLog{
				BookID:   bookId,
				Module:   module,
				Action:   action,
				TargetID: targetId,
				Detail:   detail,
				Operator: username,
			})
		}()
	}
}

func extractModule(path string) string {
	switch {
	case strings.Contains(path, "/voucher"):
		return "voucher"
	case strings.Contains(path, "/account"):
		return "account"
	case strings.Contains(path, "/book"):
		return "book"
	case strings.Contains(path, "/user") || strings.Contains(path, "/auth"):
		return "user"
	case strings.Contains(path, "/backup") || strings.Contains(path, "/log"):
		return "system"
	case strings.Contains(path, "/aux"):
		return "aux"
	case strings.Contains(path, "/closing"):
		return "closing"
	default:
		return "other"
	}
}

func extractAction(method, path string) string {
	switch method {
	case "POST":
		if strings.Contains(path, "/review") {
			return "review"
		}
		if strings.Contains(path, "/post") {
			return "post"
		}
		if strings.Contains(path, "/restore") {
			return "restore"
		}
		if strings.Contains(path, "/void") {
			return "void"
		}
		return "create"
	case "PUT":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return method
	}
}
