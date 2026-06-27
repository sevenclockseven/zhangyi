package api

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/models"
)

// ===== Backup =====

func listBackups(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		backupDir := os.Getenv("BACKUP_DIR")
		if backupDir == "" {
			backupDir = "backups"
		}

		entries, err := os.ReadDir(backupDir)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"data": []gin.H{}})
			return
		}

		backups := []gin.H{}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql.gz") {
				continue
			}
			info, _ := e.Info()
			backups = append(backups, gin.H{
				"name": e.Name(),
				"size": info.Size(),
				"time": info.ModTime(),
			})
		}

		sort.Slice(backups, func(i, j int) bool {
			return backups[j]["time"].(time.Time).Before(backups[i]["time"].(time.Time))
		})

		c.JSON(http.StatusOK, gin.H{"data": backups})
	}
}

func createBackup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		backupDir := os.Getenv("BACKUP_DIR")
		if backupDir == "" {
			backupDir = "backups"
		}
		os.MkdirAll(backupDir, 0755)

		filename := fmt.Sprintf("zhangyi_%s.sql.gz", time.Now().Format("2006-01-02_150405"))
		path := filepath.Join(backupDir, filename)

		dbDriver := os.Getenv("DB_DRIVER")
		if dbDriver == "" {
			dbDriver = "sqlite"
		}

		var cmd *exec.Cmd
		if dbDriver == "postgres" {
			dsn := os.Getenv("DB_DSN")
			cmd = exec.Command("sh", "-c", fmt.Sprintf("pg_dump '%s' | gzip > '%s'", dsn, path))
		} else {
			dbPath := os.Getenv("DB_DSN")
			if dbPath == "" {
				dbPath = "data/zhangyi.db"
			}
			cmd = exec.Command("sh", "-c", fmt.Sprintf("sqlite3 '%s' '.dump' | gzip > '%s'", dbPath, path))
		}

		if out, err := cmd.CombinedOutput(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("备份失败: %s", string(out))})
			return
		}

		info, _ := os.Stat(path)
		c.JSON(http.StatusOK, gin.H{"message": "备份成功", "file": filename, "size": info.Size()})
	}
}

func downloadBackup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		backupDir := os.Getenv("BACKUP_DIR")
		if backupDir == "" {
			backupDir = "backups"
		}
		path := filepath.Join(backupDir, name)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "备份文件不存在"})
			return
		}
		c.File(path)
	}
}

func deleteBackup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		backupDir := os.Getenv("BACKUP_DIR")
		if backupDir == "" {
			backupDir = "backups"
		}
		path := filepath.Join(backupDir, name)

		if err := os.Remove(path); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
	}
}

func restoreBackup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		backupDir := os.Getenv("BACKUP_DIR")
		if backupDir == "" {
			backupDir = "backups"
		}
		path := filepath.Join(backupDir, name)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "备份文件不存在"})
			return
		}

		dbDriver := os.Getenv("DB_DRIVER")
		if dbDriver == "" {
			dbDriver = "sqlite"
		}

		// Create a backup before restoring
		preRestoreName := fmt.Sprintf("pre_restore_%s.sql.gz", time.Now().Format("2006-01-02_150405"))
		preRestorePath := filepath.Join(backupDir, preRestoreName)

		if dbDriver == "postgres" {
			dsn := os.Getenv("DB_DSN")
			exec.Command("sh", "-c", fmt.Sprintf("pg_dump '%s' | gzip > '%s'", dsn, preRestorePath)).Run()
		} else {
			dbPath := os.Getenv("DB_DSN")
			if dbPath == "" {
				dbPath = "data/zhangyi.db"
			}
			exec.Command("sh", "-c", fmt.Sprintf("sqlite3 '%s' '.dump' | gzip > '%s'", dbPath, preRestorePath)).Run()
		}

		// Restore
		var cmd *exec.Cmd
		if dbDriver == "postgres" {
			dsn := os.Getenv("DB_DSN")
			cmd = exec.Command("sh", "-c", fmt.Sprintf("gunzip -c '%s' | psql '%s'", path, dsn))
		} else {
			dbPath := os.Getenv("DB_DSN")
			if dbPath == "" {
				dbPath = "data/zhangyi.db"
			}
			cmd = exec.Command("sh", "-c", fmt.Sprintf("gunzip -c '%s' | sqlite3 '%s'", path, dbPath))
		}

		if out, err := cmd.CombinedOutput(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("恢复失败: %s", string(out))})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "恢复成功，请重启服务", "pre_restore_backup": preRestoreName})
	}
}

// ===== Operation Logs =====

func listOperationLogs(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page := 1
		pageSize := 50
		if p := c.Query("page"); p != "" {
			fmt.Sscanf(p, "%d", &page)
		}
		if ps := c.Query("page_size"); ps != "" {
			fmt.Sscanf(ps, "%d", &pageSize)
		}

		query := db.Model(&models.OperationLog{})

		if v := c.Query("module"); v != "" {
			query = query.Where("module = ?", v)
		}
		if v := c.Query("action"); v != "" {
			query = query.Where("action = ?", v)
		}
		if v := c.Query("operator"); v != "" {
			query = query.Where("operator = ?", v)
		}
		if v := c.Query("book_id"); v != "" {
			query = query.Where("book_id = ?", v)
		}

		var total int64
		query.Count(&total)

		var logs []models.OperationLog
		query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs)

		c.JSON(http.StatusOK, gin.H{
			"data":  logs,
			"total": total,
			"page":  page,
			"size":  pageSize,
		})
	}
}

// ===== Book Users (账套权限管理) =====

func listBookUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookId := c.Param("id")

		var bookUsers []models.BookUser
		db.Preload("User").Where("book_id = ?", bookId).Find(&bookUsers)

		result := []gin.H{}
		for _, bu := range bookUsers {
			result = append(result, gin.H{
				"id":        bu.ID,
				"book_id":   bu.BookID,
				"user_id":   bu.UserID,
				"role":      bu.Role,
				"username":  bu.User.Username,
				"real_name": bu.User.RealName,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

func addBookUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookId := c.Param("id")

		var req struct {
			UserID uint   `json:"user_id" binding:"required"`
			Role   string `json:"role"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Role == "" {
			req.Role = "full"
		}

		var bookIdUint uint
		fmt.Sscanf(bookId, "%d", &bookIdUint)

		bu := models.BookUser{
			BookID: bookIdUint,
			UserID: req.UserID,
			Role:   req.Role,
		}

		if err := db.Create(&bu).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "该用户已有此账套权限"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "添加成功"})
	}
}

func updateBookUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		buid := c.Param("buid")

		var req struct {
			Role string `json:"role" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		db.Model(&models.BookUser{}).Where("id = ?", buid).Update("role", req.Role)
		c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
	}
}

func deleteBookUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		buid := c.Param("buid")
		db.Delete(&models.BookUser{}, buid)
		c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
	}
}

// getBookPermissions 获取用户的账套权限列表
func getBookPermissions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, _ := c.Get("user_id")

		var bookUsers []models.BookUser
		db.Where("user_id = ?", userId).Find(&bookUsers)

		permissions := []gin.H{}
		for _, bu := range bookUsers {
			permissions = append(permissions, gin.H{
				"book_id": bu.BookID,
				"role":    bu.Role,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": permissions})
	}
}
