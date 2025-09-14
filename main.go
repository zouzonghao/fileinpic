package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	password := os.Getenv("PASSWORD")
	if password == "" {
		password = "admin"
	}

	config := AppConfig{
		Host:      os.Getenv("HOST"),
		Password:  password,
		AuthToken: os.Getenv("AUTH_TOKEN"),
	}

	db := initDB("./fileinpic.db")
	defer db.Close()
	log.Println("Database initialized successfully.")

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("POST /api/upload", uploadHandler(db))
	mux.HandleFunc("GET /api/download/{id}", downloadHandler(db))
	mux.HandleFunc("DELETE /api/delete/{id}", deleteHandler(db))
	mux.HandleFunc("GET /api/files", filesHandler(db))
	mux.HandleFunc("POST /api/share", shareHandler(db, &config))
	mux.HandleFunc("GET /api/share/info", shareInfoHandler(db))
	mux.HandleFunc("GET /api/share/download", shareDownloadHandler(db))
	mux.HandleFunc("GET /api/file/share-details", fileShareDetailsHandler(db))
	mux.HandleFunc("GET /api/config", configHandler(config))
	mux.HandleFunc("POST /api/login", loginHandler(config))

	// Static file server for the frontend
	fs := http.FileServer(http.Dir("./static"))

	// Public routes
	mux.Handle("/login.html", fs)
	mux.Handle("/share.html", fs)
	mux.Handle("/share.js", fs)
	mux.Handle("/style.css", fs)
	mux.Handle("/app.js", fs) // Needed for login page

	// Protected routes
	mux.Handle("/", authMiddleware(fs))

	log.Println("Starting server on :37374")
	if err := http.ListenAndServe(":37374", mux); err != nil {
		log.Fatal(err)
	}
}
