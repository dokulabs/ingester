package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ingester/api"
	"ingester/auth"
	"ingester/config"
	"ingester/cost"
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

	// Use flag package to parse the configuration file
	configFilePath := flag.String("config", "./config.yml", "Path to the Doku Ingester config file")
	flag.Parse()

	// Load the configuration
	cfg, err := config.LoadConfiguration(*configFilePath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load configuration file")
		log.Info().Msg("Shutting down server")
		os.Exit(1)
	}

	// Initialize the backend database connection with loaded configuration
	log.Info().Msg("Initializing connection to the backend database")
	err = db.Init(*cfg)
	if err != nil {
		log.Error().Msg("Unable to initialize connection to the backend database")
		log.Info().Msg("Shutting down server")
		os.Exit(1)
	}
	log.Info().Msg("Successfully initialized connection to the backend database")

	// Initialize observability platform if configured
	if cfg.ObservabilityPlatform.Enabled == true {
		log.Info().Msg("Initializing for your Observability Platform")
		err := obsPlatform.Init(*cfg)
		if err != nil {
			log.Error().Msg("Exiting due to error in initializing for your Observability Platform")
			log.Info().Msg("Shutting down server")
			os.Exit(1)
		}
		log.Info().Msgf("Setup complete for sending data to %s", obsPlatform.ObservabilityPlatform)
	}

	if cfg.LLMPricing.LocalFile.Path != "" {
		log.Info().Msg("Initializing LLM Pricing Information from JSON file")
		// Load pricing data from JSON file
		if err := cost.LoadPricing(cfg.LLMPricing.LocalFile.Path); err != nil {
			log.Error().Err(err).Msg("Failed to load LLM pricing information")
			log.Info().Msg("Shutting down server")
			os.Exit(1)
		}
		log.Info().Msg("Successfully initialized LLM Pricing Information from JSON file")
	} else {
		log.Error().Msg("COSTING_JSON_FILE_PATH environment variable is not set")
		log.Info().Msg("Shutting down server")
		os.Exit(1)
	}
	// Cache eviction setup for the authentication process
	auth.InitializeCacheEviction()

	// Initialize the HTTP server routing
	r := mux.NewRouter()
	r.HandleFunc("/api/push", api.DataHandler).Methods("POST")
	r.HandleFunc("/api/keys", api.APIKeyHandler).Methods("GET", "POST", "DELETE")
	r.HandleFunc("/", api.BaseEndpoint).Methods("GET")

	// Define and start the HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.IngesterPort,
		Handler: r,
	}

	// Starts the HTTP server in a goroutine and logs any error upon starting.
	go func() {
		log.Info().Msg("Server listening on port " + cfg.IngesterPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Could not listen on port " + cfg.IngesterPort)
		}
	}()

	waitForShutdown(server)
}
