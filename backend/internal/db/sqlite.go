package db

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

func Connect() (*sql.DB, error) {
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "data.db"
	}
	db, err := sql.Open("sqlite", dbName)

	if err != nil {
		return nil, err
	}

	// Create tables if not exist
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		name TEXT,
		host_id TEXT,
		created_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS join_requests (
		id TEXT PRIMARY KEY,
		session_id TEXT,
		device_id TEXT,
		status TEXT
	);

	CREATE TABLE IF NOT EXISTS session_members (
		id TEXT PRIMARY KEY,
		session_id TEXT,
		device_id TEXT,
		device_name TEXT,
		joined_at DATETIME
	);`

	_, err = db.Exec(schema)
	return db, err
}
