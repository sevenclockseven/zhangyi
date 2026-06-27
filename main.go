package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	sqlite "github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/sevenclockseven/zhangyi/internal/api"
	"github.com/sevenclockseven/zhangyi/internal/models"
)

//go:embed all:web/dist
var webDist embed.FS

func main() {
	// Initialize database
	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		dbDriver = "sqlite"
	}
	dbDsn := os.Getenv("DB_DSN")

	var dialector gorm.Dialector
	switch dbDriver {
	case "postgres":
		if dbDsn == "" {
			dbDsn = "host=localhost user=zhangyi password=zhangyi dbname=zhangyi port=5432 sslmode=disable"
		}
		dialector = postgres.Open(dbDsn)
	default:
		if dbDsn == "" {
			dbDsn = "data/zhangyi.db"
		}
		if err := os.MkdirAll("data", 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}
		dialector = sqlite.Open(dbDsn)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	if dbDriver == "sqlite" || dbDriver == "" {
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA foreign_keys=ON")
	}

	log.Printf("Database: %s", dbDriver)

	// Start scheduled backup
	go func() {
		schedule := os.Getenv("BACKUP_SCHEDULE")
		if schedule == "" || schedule == "disabled" {
			return
		}
		// Simple interval-based backup (every 24h by default)
		interval := 24 * time.Hour
		if schedule == "hourly" {
			interval = 1 * time.Hour
		} else if schedule == "6h" {
			interval = 6 * time.Hour
		}
		time.Sleep(30 * time.Second) // Wait for app to fully start
		log.Printf("Auto backup enabled, interval: %s", interval)
		for {
			time.Sleep(interval)
			backupDir := os.Getenv("BACKUP_DIR")
			if backupDir == "" {
				backupDir = "backups"
			}
			os.MkdirAll(backupDir, 0755)
			filename := fmt.Sprintf("zhangyi_%s.sql.gz", time.Now().Format("2006-01-02_150405"))
			path := filepath.Join(backupDir, filename)
			dbPath := os.Getenv("DB_DSN")
			if dbPath == "" {
				dbPath = "data/zhangyi.db"
			}
			cmd := exec.Command("sh", "-c", fmt.Sprintf("sqlite3 '%s' '.dump' | gzip > '%s'", dbPath, path))
			if err := cmd.Run(); err != nil {
				log.Printf("Auto backup failed: %v", err)
			} else {
				log.Printf("Auto backup created: %s", filename)
				// Cleanup old backups
				keep := 30
				entries, _ := os.ReadDir(backupDir)
				var files []os.DirEntry
				for _, e := range entries {
					if !e.IsDir() && len(e.Name()) > 10 && e.Name()[:8] == "zhangyi_" {
						files = append(files, e)
						}
					}
				if len(files) > keep {
					for i := 0; i < len(files)-keep; i++ {
						os.Remove(filepath.Join(backupDir, files[i].Name()))
						}
					}
				}
			}
	}()

	// Auto migrate
	if err := db.AutoMigrate(
		&models.User{},
		&models.AccountBook{},
		&models.Account{},
		&models.Voucher{},
		&models.VoucherItem{},
		&models.OpeningBalance{},
		&models.AccountBalance{},
		&models.AuxItem{},
		&models.VoucherTemplate{},
		&models.ReportTemplate{},
		&models.OperationLog{},
		&models.BookUser{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// API routes
	// Version is set at build time via Dockerfile
	// Falls back to reading VERSION file, then to git tag
	if v, err := os.ReadFile("VERSION"); err == nil {
		api.AppVersion = strings.TrimSpace(string(v))
	}
	api.RegisterRoutes(r, db)

	// Serve embedded frontend
	distFS, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		log.Fatalf("Failed to create sub FS: %v", err)
	}

	fileServer := http.FileServer(http.FS(distFS))

	// Serve static files and handle SPA routing
	r.NoRoute(func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")

		// Try to serve static file first
		if f, err := distFS.(fs.ReadFileFS).ReadFile(path); err == nil {
			// Detect content type
			ct := http.DetectContentType(f)
			if len(path) > 4 {
				switch {
				case path[len(path)-3:] == ".js":
					ct = "application/javascript"
				case path[len(path)-4:] == ".css":
					ct = "text/css"
				case path[len(path)-4:] == ".svg":
					ct = "image/svg+xml"
				}
			}
			c.Data(http.StatusOK, ct, f)
			return
		}

		// SPA fallback: serve index.html
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// Start server
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	fmt.Printf("🦐 账易启动成功！访问 http://localhost:%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
