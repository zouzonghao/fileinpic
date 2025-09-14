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

	createFilesTableSQL := `CREATE TABLE IF NOT EXISTS files (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"filename" TEXT,
		"filesize" INTEGER,
		"upload_timestamp" DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	statement, err := db.Prepare(createFilesTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec()

	createChunksTableSQL := `CREATE TABLE IF NOT EXISTS chunks (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"file_id" INTEGER,
		"chunk_order" INTEGER,
		"image_path" TEXT,
		"auth_token" TEXT,
		FOREIGN KEY(file_id) REFERENCES files(id)
	);`

	statement, err = db.Prepare(createChunksTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec()

	return db
}