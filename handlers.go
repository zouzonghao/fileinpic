package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type FileInfo struct {
	ID              int64     `json:"id"`
	Filename        string    `json:"filename"`
	Filesize        int64     `json:"filesize"`
	UploadTimestamp time.Time `json:"upload_timestamp"`
}

type AppConfig struct {
	AuthToken string `json:"-"` // Keep this for internal use
	Host      string `json:"host,omitempty"`
	Password  string `json:"-"` // Do not expose password to the frontend
}

type UploadResponse struct {
	Ok  bool   `json:"ok"`
	Src string `json:"src"`
}

func filesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		searchQuery := r.URL.Query().Get("search")

		var rows *sql.Rows
		var err error

		if searchQuery != "" {
			query := "SELECT id, filename, filesize, upload_timestamp FROM files WHERE filename LIKE ? ORDER BY upload_timestamp DESC"
			rows, err = db.Query(query, "%"+searchQuery+"%")
		} else {
			query := "SELECT id, filename, filesize, upload_timestamp FROM files ORDER BY upload_timestamp DESC"
			rows, err = db.Query(query)
		}

		if err != nil {
			http.Error(w, "Failed to query files", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var files []FileInfo
		for rows.Next() {
			var file FileInfo
			if err := rows.Scan(&file.ID, &file.Filename, &file.Filesize, &file.UploadTimestamp); err != nil {
				http.Error(w, "Failed to scan file row", http.StatusInternalServerError)
				return
			}
			files = append(files, file)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(files); err != nil {
			log.Printf("Failed to encode files to JSON: %v", err)
		}
	}
}

func configHandler(appConfig AppConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only expose necessary fields to the frontend
		frontendConfig := struct {
			AuthToken string `json:"authToken,omitempty"`
			Host      string `json:"host,omitempty"`
		}{
			AuthToken: appConfig.AuthToken,
			Host:      appConfig.Host,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(frontendConfig); err != nil {
			log.Printf("Failed to encode config to JSON: %v", err)
			http.Error(w, "Failed to create config response", http.StatusInternalServerError)
		}
	}
}
