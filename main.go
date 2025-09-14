package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host      string `yaml:"host"`
	Password  string `yaml:"password"`
	AuthToken string `yaml:"auth_token"`
}

func loadConfig(path string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)

	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	var config AppConfig

	if *configPath != "" {
		cfg, err := loadConfig(*configPath)
		if err != nil {
			log.Fatal(err)
		}
		config.Host = cfg.Host
		config.Password = cfg.Password
		config.AuthToken = cfg.AuthToken
	} else {
		config.Host = os.Getenv("HOST")
		config.Password = os.Getenv("PASSWORD")
		config.AuthToken = os.Getenv("AUTH_TOKEN")
	}

	if config.Password == "" {
		config.Password = "admin"
	}

	db := initDB("./fileinpic.db")
	defer db.Close()
	log.Println("Database initialized successfully.")

	mux := http.NewServeMux()

	// API routes
	mux.Handle("POST /api/upload", authMiddleware(uploadHandler(db)))
	mux.Handle("GET /api/download/{id}", authMiddleware(downloadHandler(db)))
	mux.Handle("DELETE /api/delete/{id}", authMiddleware(deleteHandler(db)))
	mux.Handle("GET /api/files", authMiddleware(filesHandler(db)))
	mux.Handle("POST /api/share", authMiddleware(shareHandler(db, &config)))
	mux.HandleFunc("GET /api/share/info", shareInfoHandler(db))
	mux.HandleFunc("GET /api/share/download", shareDownloadHandler(db))
	mux.Handle("GET /api/file/share-details", authMiddleware(fileShareDetailsHandler(db)))
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

	fmt.Println("Starting server on :37374")
	if err := http.ListenAndServe(":37374", mux); err != nil {
		log.Fatal(err)
	}
}
