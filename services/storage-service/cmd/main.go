package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/database"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}
}

func RunServer(ctx context.Context, srv *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

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

	// The database is mandatory: the service must never start without it.
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		log.Fatalln("POSTGRES_HOST is not set; refusing to start without database")
	}

	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("POSTGRES_USER")
	dbPass := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	connStr := database.BuildDSN(dbUser, dbPass, dbHost, dbPort, dbName, "disable")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := database.InitDB(ctx, connStr)
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer dbPool.Close()
	log.Println("Database connection pool and schema migrations initialized successfully.")

	if err := auth.SeedAdminUser(ctx, dbPool, adminEmail, adminPassword); err != nil {
		log.Printf("Warning: Admin user seeding failed: %v", err)
	} else {
		log.Println("Admin user successfully seeded.")
	}

	const sessionTTL = 24 * time.Hour

	dbAuth := auth.NewDBAuthService(dbPool, sessionTTL)
	dbAuth.StartCleanupWorker(context.Background(), 1*time.Minute)
	authSvc = dbAuth

	authHandler := auth.NewAuthHandlerWithTTL(authSvc, sessionTTL)
	engine := storage.NewDiskEngine(storageDir)
	fileHandler := handler.NewFileHandler(engine, dbPool, quotaBytes)
	folderHandler := handler.NewFolderHandler(dbPool, engine)

	requireAuth := auth.RequireAuth(authSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.HealthHandler)
	mux.Handle("/api/v1/auth/login", auth.RequireSameOrigin(http.HandlerFunc(authHandler.LoginHandler)))
	mux.Handle("/api/v1/auth/logout", auth.RequireSameOrigin(http.HandlerFunc(authHandler.LogoutHandler)))
	mux.Handle("/api/v1/auth/me", requireAuth(http.HandlerFunc(authHandler.MeHandler)))

	mux.Handle("/api/v1/files/upload", requireAuth(auth.RequireSameOrigin(http.HandlerFunc(fileHandler.UploadHandler))))
	mux.Handle("/api/v1/files/download/", requireAuth(http.HandlerFunc(fileHandler.DownloadHandler)))
	mux.Handle("/api/v1/files", requireAuth(http.HandlerFunc(fileHandler.ListHandler)))
	mux.Handle("/api/v1/files/", requireAuth(auth.RequireSameOrigin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		segment := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/files/"), "/")
		if r.Method == http.MethodDelete && segment != "" && !strings.Contains(segment, "/") {
			fileHandler.DeleteHandler(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}))))

	mux.Handle("/api/v1/folders", requireAuth(auth.RequireSameOrigin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			folderHandler.CreateHandler(w, r)
		} else if r.Method == http.MethodGet {
			folderHandler.ListHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))))
	mux.Handle("/api/v1/folders/", requireAuth(auth.RequireSameOrigin(http.HandlerFunc(folderHandler.DeleteHandler))))

	srv := NewServer(":"+port, mux)

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("Storage service starting on port %s (storageDir: %s, defaultQuota: %d bytes)...", port, storageDir, quotaBytes)
	if err := RunServer(stopCtx, srv); err != nil {
		log.Fatalf("Server exited with error: %v", err)
	}
}
