package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type ChunkInfo struct {
	ImagePath string
	AuthToken string
}

func deleteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 3 {
			http.Error(w, "Invalid URL path", http.StatusBadRequest)
			return
		}
		fileIDStr := pathParts[2]
		fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid file ID", http.StatusBadRequest)
			return
		}

		userAuthToken := r.Header.Get("Auth-Token")
		if userAuthToken == "" {
			http.Error(w, "Auth-Token header is required", http.StatusBadRequest)
			return
		}

		rows, err := db.Query("SELECT image_path, auth_token FROM chunks WHERE file_id = ?", fileID)
		if err != nil {
			http.Error(w, "Failed to query chunks", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var chunks []ChunkInfo
		for rows.Next() {
			var chunk ChunkInfo
			if err := rows.Scan(&chunk.ImagePath, &chunk.AuthToken); err != nil {
				http.Error(w, "Failed to scan chunk row", http.StatusInternalServerError)
				return
			}
			chunks = append(chunks, chunk)
		}

		if len(chunks) == 0 {
			http.Error(w, "File not found or no chunks associated", http.StatusNotFound)
			return
		}

		// Validate auth token against the first chunk's token
		if chunks[0].AuthToken != userAuthToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Delete from external API
		for _, chunk := range chunks {
			if err := deleteImage(chunk.ImagePath, chunk.AuthToken); err != nil {
				// Log the error but try to continue deleting others
				log.Printf("Failed to delete image %s: %v", chunk.ImagePath, err)
			}
		}

		// Delete from database
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		_, err = tx.Exec("DELETE FROM chunks WHERE file_id = ?", fileID)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Failed to delete chunks from DB", http.StatusInternalServerError)
			return
		}
		_, err = tx.Exec("DELETE FROM files WHERE id = ?", fileID)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Failed to delete file from DB", http.StatusInternalServerError)
			return
		}
		tx.Commit()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "message": "File deleted successfully."})
	}
}
