package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/database"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	storageDir := os.Getenv("STORAGE_DIR")
	if storageDir == "" {
		storageDir = "./data/storage"
	}

	quotaStr := os.Getenv("DEFAULT_USER_QUOTA_BYTES")
	quotaBytes, err := strconv.ParseInt(quotaStr, 10, 64)
	if err != nil || quotaBytes <= 0 {
		quotaBytes = 5 * 1024 * 1024 * 1024 // 5GB default
	}

	// Connect to database if connection env is set
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost != "" {
		dbPort := os.Getenv("POSTGRES_PORT")
		if dbPort == "" {
			dbPort = "5432"
		}
		dbUser := os.Getenv("POSTGRES_USER")
		dbPass := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		connStr := "postgres://" + dbUser + ":" + dbPass + "@" + dbHost + ":" + dbPort + "/" + dbName + "?sslmode=disable"

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		dbPool, err := database.InitDB(ctx, connStr)
		if err != nil {
			log.Printf("Warning: Database initialization failed: %v", err)
		} else {
			defer dbPool.Close()
			log.Println("Database connection pool and schema migrations initialized successfully.")
		}
	}

	engine := storage.NewDiskEngine(storageDir)
	fileHandler := handler.NewFileHandler(engine, quotaBytes)

	http.HandleFunc("/health", handler.HealthHandler)
	http.HandleFunc("/api/v1/files/upload", fileHandler.UploadHandler)
	http.HandleFunc("/api/v1/files/download/", fileHandler.DownloadHandler)
	http.HandleFunc("/api/v1/files", fileHandler.ListHandler)

	log.Printf("Storage service starting on port %s (storageDir: %s, defaultQuota: %d bytes)...", port, storageDir, quotaBytes)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
