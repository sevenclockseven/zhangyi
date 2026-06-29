//go:build postgres
// +build postgres

package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresDriver PostgreSQL 驱动
type PostgresDriver struct{}

func (d *PostgresDriver) Open(dsn string) gorm.Dialector {
	if dsn == "" {
		dsn = "host=localhost user=zhangyi password=zhangyi dbname=zhangyi port=5432 sslmode=disable"
	}
	return postgres.Open(dsn)
}

func (d *PostgresDriver) Name() string { return "postgres" }

func openPostgresDriver() Driver {
	return &PostgresDriver{}
}
