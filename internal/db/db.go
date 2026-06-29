package db

import (
	"fmt"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/sevenclockseven/zhangyi/internal/models"
	"gorm.io/gorm"
)

// Driver 数据库驱动接口
type Driver interface {
	Open(dsn string) gorm.Dialector
	Name() string
}

// SQLiteDriver SQLite 驱动
type SQLiteDriver struct{}

func (d *SQLiteDriver) Open(dsn string) gorm.Dialector {
	if dsn == "" {
		dsn = "data/zhangyi.db"
	}
	return sqlite.Open(dsn)
}

func (d *SQLiteDriver) Name() string { return "sqlite" }

// InitDB 根据环境变量初始化数据库
func InitDB() (*gorm.DB, string, error) {
	driverName := os.Getenv("DB_DRIVER")
	if driverName == "" {
		driverName = "sqlite"
	}

	var driver Driver
	switch driverName {
	case "postgres":
		driver = openPostgresDriver()
	case "sqlite":
		driver = &SQLiteDriver{}
	default:
		return nil, "", fmt.Errorf("unsupported DB_DRIVER: %s (supported: sqlite, postgres)", driverName)
	}

	dsn := os.Getenv("DB_DSN")

	config := &gorm.Config{}
	if driverName == "sqlite" {
		config.DisableForeignKeyConstraintWhenMigrating = true
	}

	gormDB, err := gorm.Open(driver.Open(dsn), config)
	if err != nil {
		return nil, driverName, fmt.Errorf("gorm.Open failed: %w", err)
	}

	return gormDB, driverName, nil
}

// SetupDB 数据库连接后初始化设置
func SetupDB(db *gorm.DB, driver string) {
	if driver == "sqlite" {
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA foreign_keys=ON")
	}
}

// AllModels 返回所有 GORM 模型
func AllModels() []interface{} {
	return []interface{}{
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
		&models.AssetCategory{},
		&models.AssetCard{},
		&models.AssetTransaction{},
		&models.AssetDepreciation{},
	}
}
