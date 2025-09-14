package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

const apiURL = "https://i.111666.best/image"

func uploadCombinedData(data []byte, authToken string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "chunk.png")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Auth-Token", authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var uploadResp UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", err
	}

	return uploadResp.Src, nil
}

func deleteImage(imagePath string, authToken string) error {
	deleteURL := "https://i.111666.best" + imagePath
	// Using GET method as per the latest finding
	req, err := http.NewRequest("GET", deleteURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Auth-Token", authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete image, status code: %d", resp.StatusCode)
	}

	return nil
}
