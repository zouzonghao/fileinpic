package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func generateShareToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func shareHandler(db *sql.DB, config *AppConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			FileID   int64  `json:"file_id"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		token, err := generateShareToken()
		if err != nil {
			log.Printf("Failed to generate share token: %v", err)
			http.Error(w, "Failed to generate share token", http.StatusInternalServerError)
			return
		}

		_, err = db.Exec("UPDATE files SET share_password = ?, share_token = ? WHERE id = ?", req.Password, token, req.FileID)
		if err != nil {
			log.Printf("Failed to update file with share info: %v", err)
			http.Error(w, "Failed to update file", http.StatusInternalServerError)
			return
		}

		shareLink := ""
		if config.Host != "" {
			shareLink = config.Host + "/share.html?file=" + token
		} else {
			shareLink = "/share.html?file=" + token
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"share_link": shareLink})
	}
}

func shareInfoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileToken := r.URL.Query().Get("file")
		if fileToken == "" {
			http.Error(w, "Invalid share file token", http.StatusBadRequest)
			return
		}

		var filename string
		var filesize int64
		err := db.QueryRow("SELECT filename, filesize FROM files WHERE share_token = ?", fileToken).Scan(&filename, &filesize)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
			log.Printf("Failed to query file by share token: %v", err)
			http.Error(w, "Failed to query file", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"filename": filename,
			"filesize": filesize,
		})
	}
}

func shareDownloadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileToken := r.URL.Query().Get("file")
		password := r.URL.Query().Get("password")

		if fileToken == "" {
			http.Error(w, "Invalid share file token", http.StatusBadRequest)
			return
		}

		var fileID int64
		var filename, dbPassword string
		err := db.QueryRow("SELECT id, filename, share_password FROM files WHERE share_token = ?", fileToken).Scan(&fileID, &filename, &dbPassword)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
			log.Printf("Failed to query file by share token: %v", err)
			http.Error(w, "Failed to query file", http.StatusInternalServerError)
			return
		}

		if dbPassword != "" && dbPassword != password {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		log.Printf("Starting download for file ID %d via share link", fileID)

		rows, err := db.Query("SELECT image_path FROM chunks WHERE file_id = ? ORDER BY chunk_order ASC", fileID)
		if err != nil {
			log.Printf("Error: Failed to query chunks for file ID %d: %v", fileID, err)
			http.Error(w, "Failed to query chunks", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		w.Header().Set("Content-Type", "application/octet-stream")

		client := &http.Client{}
		buffer := make([]byte, 32*1024)
		chunkCount := 0

		for rows.Next() {
			chunkCount++
			var imagePath string
			if err := rows.Scan(&imagePath); err != nil {
				log.Printf("Error: Failed to scan chunk row for file ID %d: %v", fileID, err)
				http.Error(w, "Failed to scan chunk row", http.StatusInternalServerError)
				return
			}

			fullURL := "https://i.111666.best" + imagePath
			req, err := http.NewRequest("GET", fullURL, nil)
			if err != nil {
				http.Error(w, "Failed to create download request", http.StatusInternalServerError)
				return
			}

			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36")

			resp, err := client.Do(req)
			if err != nil {
				http.Error(w, "Failed to download chunk", http.StatusInternalServerError)
				return
			}

			// Use the robust skipping method from download.go
			_, err = io.CopyN(io.Discard, resp.Body, 20*1024)
			if err != nil && err != io.EOF {
				log.Printf("Error: Failed to skip carrier data for chunk %d: %v", chunkCount, err)
				resp.Body.Close()
				http.Error(w, "Failed to process chunk", http.StatusInternalServerError)
				return
			}

			_, err = io.CopyBuffer(w, resp.Body, buffer)
			if err != nil {
				resp.Body.Close()
				return // Client likely closed connection
			}
			resp.Body.Close()
		}
		log.Printf("Finished processing all %d chunks for file ID %d via share link", chunkCount, fileID)
	}
}

func fileShareDetailsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileID := r.URL.Query().Get("id")
		if fileID == "" {
			http.Error(w, "File ID is required", http.StatusBadRequest)
			return
		}

		var shareToken, sharePassword sql.NullString
		err := db.QueryRow("SELECT share_token, share_password FROM files WHERE id = ?", fileID).Scan(&shareToken, &sharePassword)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
			log.Printf("Failed to query file share details: %v", err)
			http.Error(w, "Failed to query file", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"share_token":    shareToken.String,
			"share_password": sharePassword.String,
		})
	}
}
