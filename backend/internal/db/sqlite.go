package db

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

func Connect() (*sql.DB, error) {
	if err := os.MkdirAll("./data", 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", "./data/0xnet.db")
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

	CREATE TABLE IF NOT EXISTS session_members (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		device_id TEXT NOT NULL,
		device_name TEXT,
		joined_at DATETIME,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE TABLE IF NOT EXISTS join_requests (
		id TEXT PRIMARY KEY,
		session_id TEXT,
		device_id TEXT,
		status TEXT
	);`

	_, err = db.Exec(schema)
	return db, err
}
