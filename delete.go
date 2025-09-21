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

		rows, err := db.Query("SELECT image_path FROM chunks WHERE file_id = ?", fileID)
		if err != nil {
			log.Printf("Failed to query chunks for file ID %d: %v", fileID, err)
			http.Error(w, "Failed to query chunks", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var chunks []ChunkInfo
		for rows.Next() {
			var chunk ChunkInfo
			if err := rows.Scan(&chunk.ImagePath); err != nil {
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

		// Delete from external API
		for _, chunk := range chunks {
			if err := deleteImage(chunk.ImagePath, ""); err != nil {
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

func apiDeleteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileIDStr := r.PathValue("id")
		fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
		if err != nil {
			log.Printf("Error parsing file ID '%s': %v", fileIDStr, err)
			http.Error(w, "Invalid file ID", http.StatusBadRequest)
			return
		}

		// Check the source of the file before deleting
		var source string
		err = db.QueryRow("SELECT source FROM files WHERE id = ?", fileID).Scan(&source)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
			log.Printf("Failed to query file source for file ID %d: %v", fileID, err)
			http.Error(w, "Failed to query file source", http.StatusInternalServerError)
			return
		}

		if source == "web" {
			log.Printf("API delete forbidden for file ID %d with source 'web'", fileID)
			http.Error(w, "API cannot delete files uploaded from the web UI", http.StatusForbidden)
			return
		}

		// Proceed with deletion if source is not 'web'
		rows, err := db.Query("SELECT image_path FROM chunks WHERE file_id = ?", fileID)
		if err != nil {
			log.Printf("Failed to query chunks for file ID %d: %v", fileID, err)
			http.Error(w, "Failed to query chunks", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var chunks []ChunkInfo
		for rows.Next() {
			var chunk ChunkInfo
			if err := rows.Scan(&chunk.ImagePath); err != nil {
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

		// Delete from external API
		apiKey := r.Header.Get("X-API-KEY")
		for _, chunk := range chunks {
			if err := deleteImage(chunk.ImagePath, apiKey); err != nil {
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

		log.Printf("File with ID %d deleted successfully via API", fileID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "message": "File deleted successfully."})
	}
}
