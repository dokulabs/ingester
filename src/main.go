package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ingester/api"
	"ingester/auth"
	"ingester/db"
	"ingester/obsPlatform"

	"github.com/gorilla/mux"
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

func main() {
	// DB Creditionals Setup
	log.Println("Initializing setup of credentials for the backend Database...")
	db.Init()
	log.Println("Setup complete for credentials of the backend Database")

	// Observability Platform Setup
	if os.Getenv("OBSERVABILITY_PLATFORM") != "DOKU" {
		log.Println("Initializing for your Observability Platform...")
		obsPlatform.Init()
		log.Println("Setup complete for sending data to", obsPlatform.ObservabilityPlatform)
	}

	auth.InitializeCacheEviction()

	// Initialize router
	r := mux.NewRouter()
	r.HandleFunc("/api/push", api.InsertData).Methods("POST")
	r.HandleFunc("/api/keys", api.APIKeyHandler).Methods("GET", "POST", "DELETE")
	r.HandleFunc("/", api.BaseEndpoint).Methods("GET")

	// Start HTTP server
	server := &http.Server{
		Addr:    ":9044",
		Handler: r,
	}

	go func() {
		log.Println("Server listening on port 9044...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on port 9044: %v\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down server...")

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %+v", err)
	}

	log.Println("Server exited properly")
}
