package main

import (
	"context"
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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func waitForShutdown(server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Wait for an interrupt
	sig := <-quit
	log.Info().Msgf("Caught sig: %+v", sig)
	log.Info().Msg("Starting to shutdown server")

	// Initialize the context with a timeout to ensure the app can make a graceful exit
	// or abort if it takes too long
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown failed")
	} else {
		log.Info().Msg("Server gracefully shutdown")
	}
}

// main is the entrypoint for the Doku Ingester service. It sets up logging,
// initializes the database and observability platforms, starts the HTTP server,
// and handles graceful shutdown.
func main() {
	// Configure global settings for the zerolog logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("Starting Doku Ingester")

	// Load and validate the application configuration
	// cfg, err := utils.LoadConfiguration()
	// if err != nil {
	// 	log.Fatal().Err(err).Msg("Configuration failed to load")
	// }

	// Initialize the backend database connection with loaded configuration
	log.Info().Msg("Initializing connection to the backend database")
	err := db.Init()
	if err != nil {
		log.Error().Msg("Exiting due to error in initializing connection to the backend database")
		os.Exit(1)
	}
	log.Info().Msg("Successfully initialized connection to the backend database")

	// Initialize observability platform if configured
	if os.Getenv("OBSERVABILITY_PLATFORM") != "" {
		log.Info().Msg("Initializing for your Observability Platform")
		err := obsPlatform.Init()
		if err != nil {
			log.Error().Msg("Exiting due to error in initializing for your Observability Platform")
			os.Exit(1)
		}
		log.Info().Msgf("Setup complete for sending data to %s", obsPlatform.ObservabilityPlatform)
	}

	// Cache eviction setup for the authentication process
	auth.InitializeCacheEviction()

	// Initialize the HTTP server routing
	r := mux.NewRouter()
	r.HandleFunc("/api/push", api.DataHandler).Methods("POST")
	r.HandleFunc("/api/keys", api.APIKeyHandler).Methods("GET", "POST", "DELETE")
	r.HandleFunc("/", api.BaseEndpoint).Methods("GET")

	// Get the server's port either from environment variables or use the default value(:9044).
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		log.Warn().Msg("HTTP_PORT environment variable is not set. Using the default port 9044")
		port = "9044"
	}

	// Define and start the HTTP server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Starts the HTTP server in a goroutine and logs any error upon starting.
	go func() {
		log.Info().Msg("Server listening on port " + port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Could not listen on port " + port)
		}
	}()

	waitForShutdown(server)
}
