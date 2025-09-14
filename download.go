package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

const downloadCarrierPadding = 20 * 1024 // 20KB

func downloadHandler(db *sql.DB) http.HandlerFunc {
	client := &http.Client{}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received download request for: %s", r.URL.Path)

		fileIDStr := r.PathValue("id")
		fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
		if err != nil {
			log.Printf("Error: Invalid file ID: %s", fileIDStr)
			http.Error(w, "Invalid file ID", http.StatusBadRequest)
			return
		}
		log.Printf("Attempting to download file with ID: %d", fileID)

		var filename string
		err = db.QueryRow("SELECT filename FROM files WHERE id = ?", fileID).Scan(&filename)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "File not found", http.StatusNotFound)
			} else {
				log.Printf("Error: File not found in DB for ID %d: %v", fileID, err)
				http.Error(w, "Failed to query file", http.StatusInternalServerError)
			}
			return
		}
		log.Printf("Found filename: %s", filename)

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
		buffer := make([]byte, 32*1024)

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
		log.Printf("Finished processing all %d chunks for file ID %d", chunkCount, fileID)
	}
}
