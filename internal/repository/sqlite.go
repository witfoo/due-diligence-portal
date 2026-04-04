// Package repository provides the data access layer using SQLite.
package repository

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// DB wraps a sql.DB connection to a SQLite database with WAL mode
// and other production-ready pragmas.
type DB struct {
	*sql.DB
	mu sync.RWMutex
}

// New opens a SQLite database at the given path and configures it
// with WAL mode, foreign keys, and busy timeout.
// Use ":memory:" for in-memory databases (testing).
func New(dbPath string) (*DB, error) {
	dsn := dbPath
	if dbPath != ":memory:" {
		dsn = fmt.Sprintf("file:%s?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=5000&_synchronous=NORMAL", dbPath)
	} else {
		dsn = "file::memory:?_foreign_keys=ON"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite (path=%s): %w", dbPath, err)
	}

	// SQLite works best with a single writer connection.
	db.SetMaxOpenConns(1)

	// Verify the connection works.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite (path=%s): %w", dbPath, err)
	}

	// Set pragmas that need to be executed per-connection.
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = -64000", // 64MB cache
		"PRAGMA temp_store = MEMORY",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			log.Printf("[WARN] Failed to set pragma %q: %v", pragma, err)
		}
	}

	log.Printf("[INFO] SQLite database opened: %s", dbPath)
	return &DB{DB: db}, nil
}

// Close closes the database connection, flushing WAL to main database.
func (db *DB) Close() error {
	// Checkpoint WAL before closing to ensure all data is flushed.
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		log.Printf("[WARN] WAL checkpoint failed: %v", err)
	}
	return db.DB.Close()
}
