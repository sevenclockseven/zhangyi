package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type loginAttempt struct {
	count    int
	lastSeen time.Time
}

var (
	loginAttempts = make(map[string]*loginAttempt)
	mu           sync.Mutex
)

// LoginRateLimit 登录速率限制：每IP每分钟最多10次
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		attempt, exists := loginAttempts[ip]
		if !exists {
			loginAttempts[ip] = &loginAttempt{count: 1, lastSeen: time.Now()}
			mu.Unlock()
			c.Next()
			return
		}
		if time.Since(attempt.lastSeen) > time.Minute {
			attempt.count = 1
			attempt.lastSeen = time.Now()
			mu.Unlock()
			c.Next()
			return
		}
		attempt.count++
		attempt.lastSeen = time.Now()
		if attempt.count > 10 {
			mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "登录尝试过于频繁，请1分钟后再试"})
			c.Abort()
			return
		}
		mu.Unlock()
		c.Next()
	}
}
