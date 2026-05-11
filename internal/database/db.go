package database

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Connect(dbPath string) (*sql.DB, error) {
	dirPath := filepath.Dir(dbPath)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return nil, err
		}
	}
	// turns out sqlite won't create it on its own
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	concurrencySafe := `PRAGMA journal_mode = WAL; PRAGMA synchronous = NORMAL; PRAGMA busy_timeout = 5000; PRAGMA foreign_keys = ON;`
	if _, err := db.Exec(concurrencySafe); err != nil {
		return nil, err
	}
	return db, nil
}

func Setup(db *sql.DB) error {
	stmt := "CREATE TABLE IF NOT EXISTS Invoices (id INTEGER PRIMARY KEY AUTOINCREMENT, external_id TEXT NOT NULL UNIQUE, raw_json TEXT, raw_xml TEXT, status TEXT NOT NULL DEFAULT 'PENDING', ksef_id TEXT, ksef_error TEXT, upo_xml TEXT, attempt_count INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);"
	_, err := db.Exec(stmt)
	return err
}
