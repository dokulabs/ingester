package utils

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

// CheckEnvVars checks the presence of each environment variable named in envKeys.
// Returns an error if any are missing.
func CheckEnvVars(envKeys []string) error {
	for _, key := range envKeys {
		if os.Getenv(key) == "" {
			log.Error().Msgf("Missing required environment variable: %s", key)
			return fmt.Errorf("missing required environment variable")
		}
	}
	return nil // All environment variables are present
}
