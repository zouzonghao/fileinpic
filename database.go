package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func initDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatal(err)
	}

	// Create files table
	filesTable := `
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL,
		filesize INTEGER NOT NULL,
		upload_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		share_password TEXT,
		share_token TEXT
	);`
	_, err = db.Exec(filesTable)
	if err != nil {
		log.Fatalf("Failed to create files table: %v", err)
	}

	// Create chunks table
	chunksTable := `
	CREATE TABLE IF NOT EXISTS chunks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_id INTEGER NOT NULL,
		chunk_order INTEGER NOT NULL,
		image_path TEXT NOT NULL,
		auth_token TEXT NOT NULL,
		FOREIGN KEY(file_id) REFERENCES files(id)
	);`
	_, err = db.Exec(chunksTable)
	if err != nil {
		log.Fatalf("Failed to create chunks table: %v", err)
	}

	return db
}
