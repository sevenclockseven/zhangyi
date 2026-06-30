package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sevenclockseven/zhangyi/internal/api"
	"github.com/sevenclockseven/zhangyi/internal/db"
)

//go:embed all:web/dist
var webDist embed.FS

func main() {
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database (env-driven: sqlite or postgres)
	gormDB, driverName, err := db.InitDB()
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	db.SetupDB(gormDB, driverName)

	// Auto migrate all models
	if err := gormDB.AutoMigrate(db.AllModels()...); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// API routes
	api.AppVersion = "0.9.2"
	api.RegisterRoutes(r, gormDB)

	// Serve embedded frontend
	distFS, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		log.Fatalf("Failed to create sub FS: %v", err)
	}
	fileServer := http.FileServer(http.FS(distFS))

	r.NoRoute(func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")

		if f, err := distFS.(fs.ReadFileFS).ReadFile(path); err == nil {
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

		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🦐 账易启动！访问 http://localhost:%s (DB: %s)\n", port, driverName)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
