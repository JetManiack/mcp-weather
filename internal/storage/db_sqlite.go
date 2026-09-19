package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openSQLite(dsn string) (*gorm.DB, error) {
	if dsn != ":memory:" {
		if dir := filepath.Dir(dsn); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				return nil, fmt.Errorf("create database directory: %w", err)
			}
		}
	}

	db, err := gorm.Open(sqlite.Open(dsn), gormConfig())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// SQLite's default is fail-fast on write contention (SQLITE_BUSY). Every MCP
	// tool call writes an audit row, so concurrent agents routinely contend.
	if err := db.Exec("PRAGMA busy_timeout = 5000").Error; err != nil {
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}
	if err := db.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		return nil, fmt.Errorf("set journal_mode: %w", err)
	}
	return db, nil
}
