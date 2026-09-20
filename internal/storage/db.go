// Package storage holds this service's persistence layer: GORM models and
// repository functions behind a single Open(dsn).
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Open opens a database at dsn and migrates the schema.
// DSNs starting with "postgres://" or "postgresql://" use PostgreSQL;
// everything else is treated as a SQLite file path (use ":memory:" for tests).
func Open(dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		dialector = postgres.Open(dsn)
	} else {
		if dsn != ":memory:" {
			if dir := filepath.Dir(dsn); dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0o750); err != nil {
					return nil, fmt.Errorf("create database directory: %w", err)
				}
			}
		}
		dialector = sqlite.Open(dsn)
	}

	db, err := gorm.Open(dialector, gormConfig())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// SQLite-specific pragmas for write-contention resilience.
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		// Postgres needs no extra pragmas.
	} else {
		if err := db.Exec("PRAGMA busy_timeout = 5000").Error; err != nil {
			return nil, fmt.Errorf("set busy_timeout: %w", err)
		}
		if err := db.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
			return nil, fmt.Errorf("set journal_mode: %w", err)
		}
	}

	if err := db.AutoMigrate(&Actor{}, &AgentCredential{}, &ToolCall{}, &UserIdentity{}, &Session{}); err != nil {
		return nil, fmt.Errorf("automigrate: %w", err)
	}
	return db, nil
}
