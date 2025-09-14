package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type ChunkInfo struct {
	ImagePath string
	AuthToken string
}

func deleteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileIDStr := r.PathValue("id")
		fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
		if err != nil {
			log.Printf("Error parsing file ID '%s': %v", fileIDStr, err)
			http.Error(w, "Invalid file ID", http.StatusBadRequest)
			return
		}

		userAuthToken := r.Header.Get("Auth-Token")
		if userAuthToken == "" {
			log.Printf("Auth-Token header is missing for file ID %d", fileID)
			http.Error(w, "Auth-Token header is required", http.StatusBadRequest)
			return
		}

		rows, err := db.Query("SELECT image_path, auth_token FROM chunks WHERE file_id = ?", fileID)
		if err != nil {
			log.Printf("Failed to query chunks for file ID %d: %v", fileID, err)
			http.Error(w, "Failed to query chunks", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var chunks []ChunkInfo
		for rows.Next() {
			var chunk ChunkInfo
			if err := rows.Scan(&chunk.ImagePath, &chunk.AuthToken); err != nil {
				log.Printf("Failed to scan chunk row for file ID %d: %v", fileID, err)
				http.Error(w, "Failed to scan chunk row", http.StatusInternalServerError)
				return
			}
			chunks = append(chunks, chunk)
		}

		if len(chunks) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "message": "File not found or already deleted."})
			return
		}

		// Validate auth token against the first chunk's token
		if chunks[0].AuthToken != userAuthToken {
			log.Printf("Unauthorized attempt to delete file ID %d with token %s", fileID, userAuthToken)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Delete from external API
		for _, chunk := range chunks {
			if err := deleteImage(chunk.ImagePath, chunk.AuthToken); err != nil {
				log.Printf("Failed to delete image %s: %v", chunk.ImagePath, err)
			}
		}

		// Delete from database
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Failed to start transaction for file ID %d: %v", fileID, err)
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		_, err = tx.Exec("DELETE FROM chunks WHERE file_id = ?", fileID)
		if err != nil {
			tx.Rollback()
			log.Printf("Failed to delete chunks from DB for file ID %d: %v", fileID, err)
			http.Error(w, "Failed to delete chunks from DB", http.StatusInternalServerError)
			return
		}
		_, err = tx.Exec("DELETE FROM files WHERE id = ?", fileID)
		if err != nil {
			tx.Rollback()
			log.Printf("Failed to delete file from DB for file ID %d: %v", fileID, err)
			http.Error(w, "Failed to delete file from DB", http.StatusInternalServerError)
			return
		}
		tx.Commit()

		log.Printf("File with ID %d deleted successfully", fileID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "message": "File deleted successfully."})
	}
}
