package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"ingester/api"
	"ingester/auth"
	"ingester/db"
	"ingester/obsPlatform"

	"github.com/gorilla/mux"
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("Starting Doku Ingester")

	// DB Creditionals Setup
	log.Info().Msg("Initializing connection to the backend Database")
	err := db.Init()
	if err != nil {
		log.Error().Msg("Exiting due to error in initializing connection to the backend Database")
		os.Exit(1)
	}
	log.Info().Msg("Setup complete for credentials of the backend Database")

	// Observability Platform Setup
	if os.Getenv("OBSERVABILITY_PLATFORM") != "DOKU" || os.Getenv("OBSERVABILITY_PLATFORM") != "" {
		log.Info().Msg("Initializing for your Observability Platform")
		obsPlatform.Init()
		log.Info().Msgf("Setup complete for sending data to %s", obsPlatform.ObservabilityPlatform)
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
		log.Info().Msg("Server listening on port 9044...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Msgf("Could not listen on port 9044: %v\n", err)
			os.Exit(1)

		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Info().Msg("Shutting down server")

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Msgf("Server shutdown failed: %+v", err)
	}

	log.Info().Msg("Server exited properly")
}
