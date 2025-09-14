package main

import (
	"log"
	"net/http"
)

func main() {
	db := initDB("./fileinpic.db")
	defer db.Close()
	log.Println("Database initialized successfully.")

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("POST /api/upload", uploadHandler(db))
	mux.HandleFunc("GET /api/download/{id}", downloadHandler(db))
	mux.HandleFunc("DELETE /api/delete/{id}", deleteHandler(db))
	mux.HandleFunc("GET /api/files", filesHandler(db))
	mux.HandleFunc("GET /api/config", configHandler())

	// Static file server for the frontend
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	log.Println("Starting server on :37374")
	if err := http.ListenAndServe(":37374", mux); err != nil {
		log.Fatal(err)
	}
}
