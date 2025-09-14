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
	client := &http.Client{}

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

		chunkCount := 0
		// Get a buffer from the pool
		bufferPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufferPtr) // Return the buffer to the pool when done
		buffer := *bufferPtr

		for rows.Next() {
			chunkCount++
			var imagePath string
			if err := rows.Scan(&imagePath); err != nil {
				log.Printf("Error: Failed to scan chunk row for file ID %d: %v", fileID, err)
				http.Error(w, "Failed to scan chunk row", http.StatusInternalServerError)
				return
			}
			log.Printf("Processing chunk %d, path: %s", chunkCount, imagePath)

			fullURL := "https://i.111666.best" + imagePath
			req, err := http.NewRequest("GET", fullURL, nil)
			if err != nil {
				log.Printf("Error: Failed to create request for %s: %v", fullURL, err)
				http.Error(w, "Failed to create download request", http.StatusInternalServerError)
				return
			}

			// Add extensive headers to mimic a real browser request
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
			req.Header.Set("Referer", "https://xviewer.pages.dev/")
			req.Header.Set("Sec-Fetch-Dest", "image")
			req.Header.Set("Sec-Fetch-Mode", "no-cors")
			req.Header.Set("Sec-Fetch-Site", "cross-site")

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Error: Failed to download chunk from %s: %v", fullURL, err)
				http.Error(w, "Failed to download chunk", http.StatusInternalServerError)
				return
			}

			bytesSkipped, err := io.CopyN(io.Discard, resp.Body, int64(downloadCarrierPadding))
			log.Printf("Skipped %d bytes for chunk %d", bytesSkipped, chunkCount)
			if err != nil && err != io.EOF {
				log.Printf("Error: Failed to skip carrier data for chunk %d: %v", chunkCount, err)
				resp.Body.Close()
				http.Error(w, "Failed to process chunk", http.StatusInternalServerError)
				return
			}

			bytesWritten, err := io.CopyBuffer(w, resp.Body, buffer)
			if err != nil {
				log.Printf("Error: Failed to stream chunk %d to client: %v", chunkCount, err)
				resp.Body.Close()
				return
			}
			log.Printf("Wrote %d bytes for chunk %d to response", bytesWritten, chunkCount)

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
