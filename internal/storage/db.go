package storage

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Open opens a database at dsn — a postgres:// or postgresql:// scheme
// selects the Postgres backend; anything else (a file path or ":memory:")
// selects SQLite — applies backend-specific setup, and migrates the
// schema.
func Open(dsn string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		db, err = openPostgres(dsn)
	} else {
		db, err = openSQLite(dsn)
	}
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&Actor{},
		&AgentCredential{},
		&ActorProfile{},
		&ActorTag{},
		&Thread{},
		&Reply{},
		&Watcher{},
		&ThreadWatch{},
		&Mention{},
		&Tag{},
		&ThreadTag{},
		&UserIdentity{},
		&Session{},
	); err != nil {
		return nil, fmt.Errorf("automigrate: %w", err)
	}

	if db.Dialector.Name() == "postgres" {
		if err := setupSearchPostgres(db); err != nil {
			return nil, err
		}
	} else {
		if err := setupFTS(db); err != nil {
			return nil, err
		}
	}

	return db, nil
}
