package auth

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"ingester/db"
	"sync"
	"time"
)

var (
	// apiKeyCache stores the lookup of API keys and organization IDs.
	apiKeyCache = sync.Map{}
	// CacheEntryDuration defines how long an item should stay in the cache before being re-validated.
	CacheEntryDuration = time.Minute * 10
)

type cacheEntry struct {
	Name      string
	Timestamp time.Time
}

func InitializeCacheEviction() {
	go func() {
		for {
			time.Sleep(CacheEntryDuration)
			evictExpiredEntries()
		}
	}()
}

// EvictExpiredEntries goes through the cache and evicts expired entries.
func evictExpiredEntries() {
	now := time.Now()
	apiKeyCache.Range(func(key, value interface{}) bool {
		if entry, ok := value.(cacheEntry); ok {
			if now.Sub(entry.Timestamp) >= CacheEntryDuration {
				apiKeyCache.Delete(key)
			}
		}
		return true
	})
}

// AuthenticateOrg checks the provided API key against the known keys.
func AuthenticateRequest(apiKey string) (string, error) {
	// Attempt to retrieve the name associated with the API key from the cache.
	if val, ok := apiKeyCache.Load(apiKey); ok {
		if entry, ok := val.(cacheEntry); ok {
			// Check if the cache entry is still within the valid duration.
			if time.Since(entry.Timestamp) < CacheEntryDuration {
				return entry.Name, nil
			}
		}
	}

	// If the key is not in the cache or the cache has expired, call the db to check the API key.
	name, err := db.CheckAPIKey(apiKey)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info().Msgf("Authorization Failed for an API Key")
			return "", err
		}
		return "", err
	}

	// The API key has been successfully authenticated, so cache it.
	apiKeyCache.Store(apiKey, cacheEntry{Name: name, Timestamp: time.Now()})

	return name, nil
}
