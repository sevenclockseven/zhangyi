package api

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/models"
)

// 校验备份文件名
var backupNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+\.(sql|db)\.gz$`)

func validateBackupName(name string) bool {
	return backupNameRegex.MatchString(name)
}

func safeBackupPath(backupDir, name string) (string, error) {
	if !validateBackupName(name) {
		return "", fmt.Errorf("非法文件名")
	}
	path := filepath.Join(backupDir, filepath.Base(name))
	absPath, _ := filepath.Abs(path)
	absDir, _ := filepath.Abs(backupDir)
	if !strings.HasPrefix(absPath, absDir+string(os.PathSeparator)) && absPath != absDir {
		return "", fmt.Errorf("路径越界")
	}
	return path, nil
}

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
			if e.IsDir() || (!strings.HasSuffix(e.Name(), ".sql.gz") && !strings.HasSuffix(e.Name(), ".db.gz")) {
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

		if dbDriver == "postgres" {
			// PostgreSQL: use pg_dump via exec
			dsn := os.Getenv("DB_DSN")
			cmd := exec.Command("pg_dump", dsn)
			gzipFile, err := os.Create(path)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "创建备份文件失败: " + err.Error()})
				return
			}
			defer gzipFile.Close()
			gzWriter := gzip.NewWriter(gzipFile)
			defer gzWriter.Close()
			cmd.Stdout = gzWriter
			if out, err := cmd.CombinedOutput(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("备份失败: %s", string(out))})
				return
			}
			gzWriter.Close()
			gzipFile.Close()
		} else {
			// SQLite: use online backup API (pure Go, no sqlite3 needed)
			dbPath := os.Getenv("DB_DSN")
			if dbPath == "" {
				dbPath = "data/zhangyi.db"
			}
			if err := backupSQLite(dbPath, path); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "备份失败: " + err.Error()})
				return
			}
		}

		info, _ := os.Stat(path)
		c.JSON(http.StatusOK, gin.H{"message": "备份成功", "file": filename, "size": info.Size()})
	}
}

// backupSQLite performs an online backup of SQLite database to a gzipped file
func backupSQLite(srcPath, dstPath string) error {
	// Open source database
	srcDB, err := sql.Open("sqlite", srcPath)
	if err != nil {
		return fmt.Errorf("打开源数据库失败: %w", err)
	}
	defer srcDB.Close()

	// VACUUM INTO creates a clean binary copy (no WAL, no fragmentation)
	vacuumSQL := fmt.Sprintf("VACUUM INTO '%s'", dstPath)
	if _, err := srcDB.Exec(vacuumSQL); err != nil {
		return fmt.Errorf("VACUUM INTO 失败: %w", err)
	}

	// Gzip the copied file
	sqliteData, err := os.ReadFile(dstPath)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %w", err)
	}
	os.Remove(dstPath)

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("创建备份文件失败: %w", err)
	}
	defer dstFile.Close()

	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	if _, err := gzWriter.Write(sqliteData); err != nil {
		return fmt.Errorf("写入备份失败: %w", err)
	}

	return nil
}

func downloadBackup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		backupDir := os.Getenv("BACKUP_DIR")
		if backupDir == "" {
			backupDir = "backups"
		}
		path, err := safeBackupPath(backupDir, name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "非法文件名"})
			return
		}

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
		path, err := safeBackupPath(backupDir, name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "非法文件名"})
			return
		}

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
		path, err := safeBackupPath(backupDir, name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "非法文件名"})
			return
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "备份文件不存在"})
			return
		}

		dbDriver := os.Getenv("DB_DRIVER")
		if dbDriver == "" {
			dbDriver = "sqlite"
		}

		dbPath := os.Getenv("DB_DSN")
		if dbPath == "" {
			dbPath = "data/zhangyi.db"
		}

		// 恢复前备份
		preRestoreName := fmt.Sprintf("pre_restore_%s.db.gz", time.Now().Format("2006-01-02_150405"))
		preRestorePath := filepath.Join(backupDir, preRestoreName)

		if dbDriver == "postgres" {
			dsn := os.Getenv("DB_DSN")
			dumpCmd := exec.Command("pg_dump", dsn)
			gzipCmd := exec.Command("gzip")
			gzipFile, _ := os.Create(preRestorePath)
			gzipCmd.Stdout = gzipFile
			dumpCmd.Stdout, _ = gzipCmd.StdinPipe()
			_ = gzipCmd.Start()
			dumpCmd.Run()
			gzipCmd.Wait()
			gzipFile.Close()
		} else {
			// SQLite pre-restore backup using VACUUM INTO
			if err := backupSQLite(dbPath, preRestorePath); err != nil {
				fmt.Printf("Warning: pre-restore backup failed: %v\n", err)
			}
		}

		// 恢复
		if dbDriver == "postgres" {
			dsn := os.Getenv("DB_DSN")
			gunzipCmd := exec.Command("gunzip", "-c", path)
			restoreCmd := exec.Command("psql", dsn)
			restoreCmd.Stdin, _ = gunzipCmd.StdoutPipe()
			_ = gunzipCmd.Start()
			if out, err := restoreCmd.CombinedOutput(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("恢复失败: %s", string(out))})
				return
			}
			gunzipCmd.Wait()
		} else {
			// SQLite: detect backup format and restore
			tempDBPath := dbPath + ".restore_tmp"
			if err := restoreSQLite(dbPath, path, tempDBPath); err != nil {
				os.Remove(tempDBPath)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复失败: " + err.Error()})
				return
			}

			// Close GORM connection pool so it releases file locks
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}

			// Remove old DB files
			os.Remove(dbPath)
			os.Remove(dbPath + "-wal")
			os.Remove(dbPath + "-shm")

			// Rename restored temp DB to actual DB path
			if err := os.Rename(tempDBPath, dbPath); err != nil {
				// Try copy as fallback (cross-device rename)
				input, _ := os.ReadFile(tempDBPath)
				os.WriteFile(dbPath, input, 0644)
				os.Remove(tempDBPath)
			}

			// Reopen GORM connection
			gormDB2, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
				DisableForeignKeyConstraintWhenMigrating: true,
			})
			if err == nil {
				*db = *gormDB2
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "恢复成功，数据已生效", "pre_restore_backup": preRestoreName})
	}
}

// restoreSQLite restores a SQLite database from a gzipped backup.
// Supports both new binary format (.sqlite.gz from VACUUM INTO) and legacy SQL dump (.sql.gz).
// Restores into tempDBPath, caller is responsible for swapping files.
func restoreSQLite(currentDBPath, dumpPath, tempDBPath string) error {
	// Detect format: try gzip first, check if it's raw SQLite or SQL dump
	f, err := os.Open(dumpPath)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %w", err)
	}
	defer f.Close()

	// Read compressed data
	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("解压备份文件失败: %w", err)
	}
	defer gzReader.Close()

	compressedData, err := io.ReadAll(gzReader)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %w", err)
	}

	// Check if it's a binary SQLite file (starts with "SQLite format 3\000")
	if len(compressedData) > 16 && string(compressedData[:16]) == "SQLite format 3\x00" {
		// Binary SQLite backup — write directly to temp path
		if err := os.WriteFile(tempDBPath, compressedData, 0644); err != nil {
			return fmt.Errorf("写入恢复文件失败: %w", err)
		}
		return nil
	}

	// Legacy SQL dump format — execute into temp DB
	tempDB, err := sql.Open("sqlite", tempDBPath)
	if err != nil {
		return fmt.Errorf("创建临时数据库失败: %w", err)
	}
	defer tempDB.Close()

	// Regex to match unquoted datetime values like: 2026-07-22 22:10:39.461760003 +0800 +0800
	reDatetime := regexp.MustCompile(`,(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}[^,)]*?)([,)])`)

	// Execute SQL statements line by line
	lines := strings.Split(string(compressedData), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimRight(line, ";")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Fix unquoted datetime values in INSERT statements
		if strings.HasPrefix(strings.ToUpper(line), "INSERT") {
			line = reDatetime.ReplaceAllString(line, ",'$1'$2")
		}
		if _, err := tempDB.Exec(line); err != nil {
			fmt.Printf("Warning: execute SQL failed: %v\n", err)
		}
	}

	return nil
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

func cleanupOperationLogs(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		keepDays := 90
		if d := c.Query("keep_days"); d != "" {
			fmt.Sscanf(d, "%d", &keepDays)
		}
		if keepDays < 1 {
			keepDays = 1
		}
		if keepDays > 3650 {
			keepDays = 3650
		}

		cutoff := time.Now().AddDate(0, 0, -keepDays)
		result := db.Where("created_at < ?", cutoff).Delete(&models.OperationLog{})

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("已清理 %d 天前的日志，共删除 %d 条", keepDays, result.RowsAffected),
			"deleted": result.RowsAffected,
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
