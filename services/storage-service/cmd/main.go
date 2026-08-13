package main

import (
	"log"
	"net/http"
	"os"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", handler.HealthHandler)

	log.Printf("Storage service starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
