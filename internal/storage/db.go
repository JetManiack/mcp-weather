// Package storage holds this service's persistence layer: GORM models and
// repository functions behind a single Open(dsn).
package storage

import (
	"fmt"

	"gorm.io/gorm"
)

// Open opens a SQLite database at dsn (a file path or ":memory:") and
// migrates the schema. Postgres support can be added by following the
// pattern in mcp-webtools/internal/storage.
func Open(dsn string) (*gorm.DB, error) {
	db, err := openSQLite(dsn)
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&Actor{}, &AgentCredential{}, &ToolCall{}); err != nil {
		return nil, fmt.Errorf("automigrate: %w", err)
	}
	return db, nil
}
