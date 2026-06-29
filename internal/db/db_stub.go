//go:build !postgres
// +build !postgres

package db

import "fmt"

func openPostgresDriver() Driver {
	panic(fmt.Sprintf("postgres driver not compiled. Build with -tags postgres"))
}
