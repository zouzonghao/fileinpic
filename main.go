package main

import (
	"log"
	"net/http"
)

func main() {
	db := initDB("./fileinpic.db")
	defer db.Close()
	log.Println("Database initialized successfully.")

	// API routes
	http.HandleFunc("/api/upload", uploadHandler(db))
	http.HandleFunc("/api/download/", downloadHandler(db))
	http.HandleFunc("/api/delete/", deleteHandler(db))
	http.HandleFunc("/api/files", filesHandler(db))
	http.HandleFunc("/api/config", configHandler())

	// Static file server for the frontend
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	log.Println("Starting server on :37374")
	if err := http.ListenAndServe(":37374", nil); err != nil {
		log.Fatal(err)
	}
}
