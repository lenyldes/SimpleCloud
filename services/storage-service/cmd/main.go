package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
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

	var authSvc auth.Service
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

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
			authSvc = auth.NewMockAuthService()
		} else {
			defer dbPool.Close()
			log.Println("Database connection pool and schema migrations initialized successfully.")

			if err := auth.SeedAdminUser(ctx, dbPool, adminEmail, adminPassword); err != nil {
				log.Printf("Warning: Admin user seeding failed: %v", err)
			} else {
				log.Println("Admin user successfully seeded.")
			}

			dbAuth := auth.NewDBAuthService(dbPool, 24*time.Hour)
			dbAuth.StartCleanupWorker(context.Background(), 1*time.Minute)
			authSvc = dbAuth
		}
	} else {
		authSvc = auth.NewMockAuthService()
	}

	authHandler := auth.NewAuthHandler(authSvc)
	engine := storage.NewDiskEngine(storageDir)
	fileHandler := handler.NewFileHandler(engine, quotaBytes)
	folderHandler := handler.NewFolderHandler(engine)

	requireAuth := auth.RequireAuth(authSvc)

	http.HandleFunc("/health", handler.HealthHandler)
	http.HandleFunc("/api/v1/auth/login", authHandler.LoginHandler)
	http.HandleFunc("/api/v1/auth/logout", authHandler.LogoutHandler)
	http.HandleFunc("/api/v1/auth/me", authHandler.MeHandler)

	http.Handle("/api/v1/files/upload", requireAuth(http.HandlerFunc(fileHandler.UploadHandler)))
	http.Handle("/api/v1/files/download/", requireAuth(http.HandlerFunc(fileHandler.DownloadHandler)))
	http.Handle("/api/v1/files", requireAuth(http.HandlerFunc(fileHandler.ListHandler)))

	http.Handle("/api/v1/folders", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			folderHandler.CreateHandler(w, r)
		} else if r.Method == http.MethodGet {
			folderHandler.ListHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	http.Handle("/api/v1/folders/", requireAuth(http.HandlerFunc(folderHandler.DeleteHandler)))

	log.Printf("Storage service starting on port %s (storageDir: %s, defaultQuota: %d bytes)...", port, storageDir, quotaBytes)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
