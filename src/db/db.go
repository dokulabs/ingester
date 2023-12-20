package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	_ "github.com/lib/pq"
	"ingester/cost"
	"ingester/obsPlatform"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

var (
	once     sync.Once
	db       *sql.DB
	dbConfig DBConfig
)

func getCreateAPIKeysTableSQL(tableName string) string {
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id SERIAL PRIMARY KEY,
		api_key VARCHAR(255) NOT NULL UNIQUE,
		name VARCHAR(50) NOT NULL
	);`, tableName)
}

func getCreateDataTableSQL(tableName string) string {
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		time TIMESTAMPTZ NOT NULL,
		orgID VARCHAR(10) NOT NULL,
		environment VARCHAR(50) NOT NULL,
		endpoint VARCHAR(50) NOT NULL,
		sourceLanguage VARCHAR(50) NOT NULL,
		applicationName VARCHAR(50) NOT NULL,
		completionTokens INTEGER,
		promptTokens INTEGER,
		totalTokens INTEGER,
		finishReason VARCHAR(50),
		requestDuration DOUBLE PRECISION,
		usageCost DOUBLE PRECISION,
		model VARCHAR(50),
		prompt TEXT,
		response TEXT,
		imageSize TEXT,
		revisedPrompt TEXT,
		image TEXT,
		audioVoice TEXT,
		finetuneJobId TEXT,
		finetuneJobStatus TEXT
	);`, tableName)
}

// validFields represent the fields that are expected in the incoming data.
var validFields = []string{
	"orgID",
	"environment",
	"endpoint",
	"sourceLanguage",
	"applicationName",
	"completionTokens",
	"promptTokens",
	"totalTokens",
	"finishReason",
	"requestDuration",
	"usageCost",
	"model",
	"prompt",
	"response",
	"imageSize",
	"revisedPrompt",
	"image",
	"audioVoice",
	"finetuneJobId",
	"finetuneJobStatus",
}

// DBConfig holds the database configuration
type DBConfig struct {
	DBName          string
	User            string
	Password        string
	Host            string
	Port            string
	SSLMode         string
	DataTableName   string
	ApiKeyTableName string
}

// PingDB attempts to ping the database to check if it's alive.
func PingDB() error {
	return db.Ping()
}

func tableExists(db *sql.DB, tableName string) bool {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM   information_schema.tables 
			WHERE  table_schema = 'public'
			AND    lower(table_name) = lower($1)
		)
	`

	var exists bool
	err := db.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		log.Printf("Error checking table existence: %v\n", err)
		return false
	}

	return exists
}

func createTable(db *sql.DB, tableName string) error {
	var createTableSQL string
	if tableName == dbConfig.ApiKeyTableName {
		createTableSQL = getCreateAPIKeysTableSQL(tableName)
	} else if tableName == dbConfig.DataTableName {
		createTableSQL = getCreateDataTableSQL(tableName)
	}
	if tableExists(db, tableName) == false {
		_, err := db.Exec(createTableSQL)
		if err != nil {
			log.Fatalf("Error creating table: %v", err)

		}

		if tableName == dbConfig.ApiKeyTableName {
			createIndexSQL := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_api_key ON %s (api_key);", tableName)
			_, err = db.Exec(createIndexSQL)
			if err != nil {
				log.Fatalf("Error creating index on 'api_key' column: %v", err)
			}
			log.Printf("Index on 'api_key' column checked/created.")
		}

		// If the table to create is the data table, convert it into a hypertable
		if tableName == dbConfig.DataTableName {
			_, err := db.Exec("SELECT create_hypertable($1, 'time')", tableName)
			if err != nil {
				log.Fatalf("Error creating hypertable: %v", err)
				return err
			}
		}

		log.Printf("Table '%s' created.\n", tableName)
	}
	return nil
}

// Init initializes the database connection.
func initializeDB() {
	once.Do(func() {
		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName, dbConfig.SSLMode)

		var err error
		if db, err = sql.Open("postgres", connStr); err != nil {
			log.Fatalf("Error connecting to the database: %v", err)
		}

		// Set maximum open connections and idle connections.
		maxOpenConns, err := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS"))
		if err != nil {
			maxOpenConns = 10
		}
		db.SetMaxOpenConns(maxOpenConns)

		maxIdleConns, err := strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONNS"))
		if err != nil {
			maxIdleConns = 5
		}
		db.SetMaxIdleConns(maxIdleConns)
	})
}

// insertDataToDB inserts data into the database.
func insertDataToDB(data map[string]interface{}) (string, int) {
	// Calculate usage cost based on the endpoint type
	if data["endpoint"] == "openai.embeddings" {
		data["usageCost"] = cost.CalculateEmbeddingsCost(data["promptTokens"].(float64), data["model"].(string))
	} else if data["endpoint"] == "openai.chat.completions" || data["endpoint"] == "openai.completions" || data["endpoint"] == "anthropic.completions" {
		data["usageCost"] = cost.CalculateChatCost(data["promptTokens"].(float64), data["completionTokens"].(float64), data["model"].(string))
	} else if data["endpoint"] == "openai.images.create" || data["endpoint"] == "openai.images.create.variations" {
		data["usageCost"] = cost.CalculateImageCost(data["model"].(string), data["imageSize"].(string), data["imageQuality"].(string))
	}

	go obsPlatform.SendToPlatform(data)

	// Fill missing fields with nil
	for _, field := range validFields {
		if _, exists := data[field]; !exists {
			data[field] = nil
		}
	}

	// Define the SQL query for data insertion
	query := fmt.Sprintf("INSERT INTO %s (time, orgID, environment, endpoint, sourceLanguage, applicationName, completionTokens, promptTokens, totalTokens, finishReason, requestDuration, usageCost, model, prompt, response, imageSize, revisedPrompt, image, audioVoice, finetuneJobId, finetuneJobStatus) VALUES (NOW(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)", dbConfig.DataTableName)

	// Execute the SQL query
	_, err := db.Exec(query,
		data["orgID"],
		data["environment"],
		data["endpoint"],
		data["sourceLanguage"],
		data["applicationName"],
		data["completionTokens"],
		data["promptTokens"],
		data["totalTokens"],
		data["finishReason"],
		data["requestDuration"],
		data["usageCost"],
		data["model"],
		data["prompt"],
		data["response"],
		data["imageSize"],
		data["revisedPrompt"],
		data["image"],
		data["audioVoice"],
		data["finetuneJobId"],
		data["finetuneJobStatus"],
	)
	if err != nil {
		log.Printf("Database error: %v\n", err)
		// Update the response message and status code for error
		return "Internal Server Error", http.StatusInternalServerError
	}

	return "Data insertion completed", http.StatusCreated
}

func Init() {
	dbConfig = DBConfig{
		DBName:          os.Getenv("DB_NAME"),
		User:            os.Getenv("DB_USER"),
		Password:        os.Getenv("DB_PASSWORD"),
		Host:            os.Getenv("DB_HOST"),
		Port:            os.Getenv("DB_PORT"),
		SSLMode:         os.Getenv("DB_SSLMODE"),
		DataTableName:   os.Getenv("DATA_TABLE_NAME"),
		ApiKeyTableName: os.Getenv("APIKEY_TABLE_NAME"),
	}
	initializeDB()
	err := createTable(db, dbConfig.ApiKeyTableName)
	if err != nil {
		log.Fatalf("Could not create api_keys table: %v", err)
	}

	// Create the data table if it doesn't exist and convert it to a hypertable.
	err = createTable(db, dbConfig.DataTableName)
	if err != nil {
		log.Fatalf("Could not create or convert data table to hypertable: %v", err)
	}
}

// PerformDatabaseInsertion performs the database insertion synchronously.
// It returns the response message and status code.
func PerformDatabaseInsertion(data map[string]interface{}) (string, int) {
	// Call insertDataToDB directly instead of starting a new goroutine.
	responseMessage, statusCode := insertDataToDB(data)

	// The operation is now synchronous. Once insertDataToDB returns, the result is ready to use.
	return responseMessage, statusCode
}

// CheckAPIKey retrieves the name associated with the given API key from the database.
func CheckAPIKey(apiKey string) (string, error) {
	var name string

	query := fmt.Sprintf("SELECT name FROM %s WHERE api_key = $1", dbConfig.ApiKeyTableName)
	err := db.QueryRow(query, apiKey).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", err
		}
		return "", err
	}
	return name, nil
}

// GenerateAPIKey generates a new API key for a given name and stores it in the database.
func GenerateAPIKey(existingAPIKey, name string) (string, error) {
	// If there are any existing API keys, authorize the provided API key before proceeding
	var count int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", dbConfig.ApiKeyTableName)
	err := db.QueryRow(countQuery).Scan(&count)
	if err != nil {
		return "", fmt.Errorf("failed to check API key table: %v", err)
	}

	// Only perform the check if the count is greater than zero
	if count > 0 {
		_, err = CheckAPIKey(existingAPIKey)
		if err != nil {
			return "", fmt.Errorf("Authorization failed: %v", err)
		}
		// Attempt to retrieve any existing key for the given name.
		_, err = GetAPIKeyForName(existingAPIKey, name)
		if err == nil {
			// If no error, an API key already exists with this name.
			return "", fmt.Errorf("A key with this name already exists")
		} else if err != sql.ErrNoRows {
			// If we received an error other than ErrNoRows, it may indicate a serious issue rather than simply "not found."
			return "", fmt.Errorf("unexpected error: %v", err)
		}
	}

	// No existing key found, proceed to generate a new API key
	newAPIKey, _ := GenerateSecureRandomKey()

	// Insert the new API key into the database
	insertQuery := fmt.Sprintf("INSERT INTO %s (api_key, name) VALUES ($1, $2)", dbConfig.ApiKeyTableName)
	_, err = db.Exec(insertQuery, newAPIKey, name)
	if err != nil {
		return "", fmt.Errorf("failed to insert the new API key: %v", err)
	}

	return newAPIKey, nil
}

// GetAPIKeyForName retrieves an API key for a given name from the database.
func GetAPIKeyForName(existingAPIKey, name string) (string, error) {
	_, err := CheckAPIKey(existingAPIKey)
	if err != nil {
		return "", fmt.Errorf("Authorization failed")
	}

	var apiKey string
	query := fmt.Sprintf("SELECT api_key FROM %s WHERE name = $1", dbConfig.ApiKeyTableName)
	err = db.QueryRow(query, name).Scan(&apiKey)
	if err != nil {
		return "", err
	}

	return apiKey, nil
}

func DeleteAPIKey(existingAPIKey, name string) error {
	apiKey, err := GetAPIKeyForName(existingAPIKey, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("API Key with name '%s' not found", name)
		}
		return err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE api_key = $1", dbConfig.ApiKeyTableName)
	_, err = db.Exec(query, apiKey)
	if err != nil {
		return fmt.Errorf("Failed to delete API key")
	}

	return nil
}

// GenerateSecureRandomKey should generate a secure random string to be used as an API key.
func GenerateSecureRandomKey() (string, error) {
	randomPartLength := 40 / 2 // Each byte becomes two hex characters, so we need half as many bytes.

	randomBytes := make([]byte, randomPartLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// In the case of an error, return it as we cannot generate a key.
		return "", err
	}

	// Encode the random bytes as a hex string and prefix with 'dk'.
	randomHexString := hex.EncodeToString(randomBytes)
	apiKey := "dk" + randomHexString

	return apiKey, nil
}
