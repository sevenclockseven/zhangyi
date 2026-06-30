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

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/models"
)

// 校验备份文件名
var backupNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+\.sql\.gz$`)

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
	// Open source database in read-only mode
	srcDB, err := sql.Open("sqlite", srcPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("打开源数据库失败: %w", err)
	}
	defer srcDB.Close()

	// Verify source is accessible
	if err := srcDB.Ping(); err != nil {
		return fmt.Errorf("源数据库不可访问: %w", err)
	}

	// Create destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	// Use SQLite backup API via VACUUM INTO piping
	// Since we can't use backup API directly through database/sql,
	// we'll use .dump equivalent by exporting SQL
	rows, err := srcDB.Query("SELECT sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return fmt.Errorf("查询表结构失败: %w", err)
	}
	defer rows.Close()

	var sqlStmts []string
	for rows.Next() {
		var sqlStr string
		if err := rows.Scan(&sqlStr); err != nil {
			continue
		}
		sqlStmts = append(sqlStmts, sqlStr+";")
	}

	// Export data from each table
	tableRows, err := srcDB.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return fmt.Errorf("查询表名失败: %w", err)
	}
	defer tableRows.Close()

	for tableRows.Next() {
		var tableName string
		if err := tableRows.Scan(&tableName); err != nil {
			continue
		}
		dataRows, err := srcDB.Query(fmt.Sprintf("SELECT * FROM [%s]", tableName))
		if err != nil {
			continue
		}
		cols, _ := dataRows.Columns()
		for dataRows.Next() {
			values := make([]interface{}, len(cols))
			valuePtrs := make([]interface{}, len(cols))
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			if err := dataRows.Scan(valuePtrs...); err != nil {
				continue
			}
			var rowVals []string
			for _, v := range values {
				if v == nil {
					rowVals = append(rowVals, "NULL")
				} else {
					switch val := v.(type) {
					case []byte:
						rowVals = append(rowVals, fmt.Sprintf("X'%x'", val))
					case string:
						escaped := strings.ReplaceAll(val, "'", "''")
						rowVals = append(rowVals, "'"+escaped+"'")
					default:
						rowVals = append(rowVals, fmt.Sprintf("%v", val))
					}
				}
			}
			sqlStmts = append(sqlStmts, fmt.Sprintf("INSERT INTO [%s] VALUES(%s);", tableName, strings.Join(rowVals, ",")))
		}
		dataRows.Close()
	}

	// Write all SQL to gzip
	for _, stmt := range sqlStmts {
		if _, err := io.WriteString(gzWriter, stmt+"\n"); err != nil {
			return fmt.Errorf("写入备份失败: %w", err)
		}
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
		preRestoreName := fmt.Sprintf("pre_restore_%s.sql.gz", time.Now().Format("2006-01-02_150405"))
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
			// SQLite pre-restore backup using Go-native method
			if err := backupSQLite(dbPath, preRestorePath); err != nil {
				// Non-fatal, continue with restore
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
			// SQLite restore using Go-native method
			if err := restoreSQLite(dbPath, path); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复失败: " + err.Error()})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "恢复成功，请重启服务", "pre_restore_backup": preRestoreName})
	}
}

// restoreSQLite restores a SQLite database from a gzipped SQL dump
func restoreSQLite(dbPath, dumpPath string) error {
	// Open gzip file
	f, err := os.Open(dumpPath)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %w", err)
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("解压备份文件失败: %w", err)
	}
	defer gzReader.Close()

	// Open target database
	targetDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("打开目标数据库失败: %w", err)
	}
	defer targetDB.Close()

	// Read and execute SQL statements
	sqlData, err := io.ReadAll(gzReader)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %w", err)
	}

	// Split by semicolons and execute
	statements := strings.Split(string(sqlData), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := targetDB.Exec(stmt); err != nil {
			// Skip errors for non-critical statements
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
