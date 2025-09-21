package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
)

const (
	maxUploadSize = 7 * 1024 * 1024 // 7 MB
	chunkSize     = 6 * 1024 * 1024 // 6 MB
)

func uploadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authToken := r.Header.Get("Auth-Token")
		if authToken == "" {
			http.Error(w, "Auth-Token header is required", http.StatusBadRequest)
			return
		}

		// It's important to still parse the form to get access to the file handle
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, "Could not parse multipart form", http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "Invalid file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		filename := handler.Filename
		filesize := handler.Size // Use the size from the handler, no need to read the file
		log.Printf("Received file: %s, size: %d bytes", filename, filesize)

		// 1. Save file metadata to DB
		res, err := db.Exec("INSERT INTO files (filename, filesize) VALUES (?, ?)", filename, filesize)
		if err != nil {
			http.Error(w, "Failed to save file metadata", http.StatusInternalServerError)
			return
		}
		fileID, err := res.LastInsertId()
		if err != nil {
			http.Error(w, "Failed to get last insert ID", http.StatusInternalServerError)
			return
		}

		// 2. Process file in chunks
		numChunks := int(math.Ceil(float64(filesize) / float64(chunkSize)))
		log.Printf("Splitting into %d chunks", numChunks)
		chunkBuffer := make([]byte, chunkSize)

		for i := 0; i < numChunks; i++ {
			// Read a chunk from the file stream
			bytesRead, err := io.ReadFull(file, chunkBuffer)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				http.Error(w, "Failed to read file chunk", http.StatusInternalServerError)
				log.Printf("Chunk read error: %v", err)
				return
			}

			// This is the actual chunk data for this iteration
			chunkData := chunkBuffer[:bytesRead]

			// 3. Create carrier PNG
			carrierText := fmt.Sprintf("%s - %d/%d", filename, i+1, numChunks)
			carrierData, err := createCarrierPNG(carrierText)
			if err != nil {
				http.Error(w, "Failed to create carrier PNG", http.StatusInternalServerError)
				return
			}

			// 4. Combine carrier and chunk
			combinedData := append(carrierData, chunkData...)

			// 5. Upload to external API
			imagePath, err := uploadCombinedData(combinedData, authToken)
			if err != nil {
				http.Error(w, "Failed to upload chunk", http.StatusInternalServerError)
				log.Printf("Upload error: %v", err)
				return
			}
			log.Printf("Uploaded chunk %d, image path: %s", i+1, imagePath)

			// 6. Save chunk info to DB
			_, err = db.Exec("INSERT INTO chunks (file_id, chunk_order, image_path, auth_token) VALUES (?, ?, ?, ?)",
				fileID, i, imagePath, authToken)
			if err != nil {
				http.Error(w, "Failed to save chunk metadata", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "message": "File uploaded successfully."})
	}
}

func apiUploadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authToken := r.Header.Get("Auth-Token")
		if authToken == "" {
			http.Error(w, "Auth-Token header is required", http.StatusBadRequest)
			return
		}

		contentDisposition := r.Header.Get("Content-Disposition")
		if contentDisposition == "" {
			http.Error(w, "Content-Disposition header is required", http.StatusBadRequest)
			return
		}

		filename := ""
		if _, err := fmt.Sscanf(contentDisposition, "attachment; filename=\"%s\"", &filename); err != nil {
			http.Error(w, "Invalid Content-Disposition header", http.StatusBadRequest)
			return
		}

		// 1. Save file metadata to DB
		res, err := db.Exec("INSERT INTO files (filename, filesize) VALUES (?, ?)", filename, r.ContentLength)
		if err != nil {
			http.Error(w, "Failed to save file metadata", http.StatusInternalServerError)
			return
		}
		fileID, err := res.LastInsertId()
		if err != nil {
			http.Error(w, "Failed to get last insert ID", http.StatusInternalServerError)
			return
		}

		// 2. Process file in chunks
		numChunks := int(math.Ceil(float64(r.ContentLength) / float64(chunkSize)))
		log.Printf("Splitting into %d chunks", numChunks)
		chunkBuffer := make([]byte, chunkSize)

		for i := 0; i < numChunks; i++ {
			// Read a chunk from the file stream
			bytesRead, err := io.ReadFull(r.Body, chunkBuffer)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				http.Error(w, "Failed to read file chunk", http.StatusInternalServerError)
				log.Printf("Chunk read error: %v", err)
				return
			}

			// This is the actual chunk data for this iteration
			chunkData := chunkBuffer[:bytesRead]

			// 3. Create carrier PNG
			carrierText := fmt.Sprintf("%s - %d/%d", filename, i+1, numChunks)
			carrierData, err := createCarrierPNG(carrierText)
			if err != nil {
				http.Error(w, "Failed to create carrier PNG", http.StatusInternalServerError)
				return
			}

			// 4. Combine carrier and chunk
			combinedData := append(carrierData, chunkData...)

			// 5. Upload to external API
			imagePath, err := uploadCombinedData(combinedData, authToken)
			if err != nil {
				http.Error(w, "Failed to upload chunk", http.StatusInternalServerError)
				log.Printf("Upload error: %v", err)
				return
			}
			log.Printf("Uploaded chunk %d, image path: %s", i+1, imagePath)

			// 6. Save chunk info to DB
			_, err = db.Exec("INSERT INTO chunks (file_id, chunk_order, image_path, auth_token) VALUES (?, ?, ?, ?)",
				fileID, i, imagePath, authToken)
			if err != nil {
				http.Error(w, "Failed to save chunk metadata", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":  true,
			"url": fmt.Sprintf("/api/v1/files/public/download/%d", fileID),
		})
	}
}
