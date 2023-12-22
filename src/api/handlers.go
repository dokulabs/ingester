package api

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"ingester/auth"
	"ingester/db"
	"net/http"
)

func generateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {

	existingAPIKey := r.Header.Get("Authorization")
	var request struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	newAPIKey, err := db.GenerateAPIKey(existingAPIKey, request.Name)
	if err != nil {
		// If the error message is "KEYEXISTS", you can send an http.StatusConflict status code
		if err.Error() == "KEYEXISTS" {
			sendJSONResponse(w, http.StatusConflict, "An API Key with the same name already exists")
			return
		} else if err.Error() == "AUTHFAILED" {
			// For other types of errors, an http.StatusUnauthorized or http.StatusInternalServerError could be appropriate
			sendJSONResponse(w, http.StatusUnauthorized, "Unauthorized: Please check your API key and try again")
			return
		} else {
			sendJSONResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// If the key was successfully generated, return it with an http.StatusOK status code
	sendJSONResponse(w, http.StatusOK, newAPIKey)
}

// getAPIKeyHandler handles retrieving an existing API key.
func getAPIKeyHandler(w http.ResponseWriter, r *http.Request) {

	existingAPIKey := r.Header.Get("Authorization")
	var request struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	apiKey, err := db.GetAPIKeyForName(existingAPIKey, request.Name)
	if err != nil {
		if err.Error() == "AUTHFAILED" {
			sendJSONResponse(w, http.StatusUnauthorized, "Unauthorized: Please check your API key and try again")
			return
		} else if err.Error() == "NOTFOUND" {
			sendJSONResponse(w, http.StatusNotFound, fmt.Sprintf("Unable to find API key with the given name %s", request.Name))
			return
		}
		sendJSONResponse(w, http.StatusNotFound, fmt.Sprintf("Unable to find API key with the given name %s", request.Name))
		return
	}

	sendJSONResponse(w, http.StatusOK, apiKey)
}

func deleteAPIKeyHandler(w http.ResponseWriter, r *http.Request) {

	existingAPIKey := r.Header.Get("Authorization")
	var request struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// The ExistingAPIKey is used for authentication purposes
	err := db.DeleteAPIKey(existingAPIKey, request.Name)
	if err != nil {
		if err.Error() == "AUTHFAILED" {
			sendJSONResponse(w, http.StatusUnauthorized, "Unauthorized: Please check your API key and try again")
			return
		} else if err.Error() == "NOTFOUND" {
			sendJSONResponse(w, http.StatusNotFound, fmt.Sprintf("Unable to find API key with the given name '%s'", request.Name))
			return
		}
		sendJSONResponse(w, http.StatusNotFound, err.Error())
		return
	}
	sendJSONResponse(w, http.StatusOK, "API key deleted successfully")
}

// sendJSONResponse sends a JSON response with the appropriate headers and status code.
func sendJSONResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := struct {
		Status  int         `json:"status"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	}{
		Status:  status,
		Message: message,
	}

	json.NewEncoder(w).Encode(response)
}

func InsertData(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, "Invalid Inputs provided")
		return
	}

	data["orgID"], err = auth.AuthenticateRequest(r.Header.Get("Authorization"))

	if err != nil {
		sendJSONResponse(w, http.StatusUnauthorized, "Unauthorized: Please check your API key and try again")
		return
	}

	// Check if skipResp is true
	skipResp, ok := data["skipResp"].(bool)
	if ok && skipResp == true {
		sendJSONResponse(w, http.StatusAccepted, "Insertion started in background")
		go db.PerformDatabaseInsertion(data) // Running as a goroutine for async processing
		return
	}

	responseMessage, statusCode := db.PerformDatabaseInsertion(data)

	// Respond to the user with the insertion status and any potential errors
	sendJSONResponse(w, statusCode, responseMessage)
}

func APIKeyHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getAPIKeyHandler(w, r)
	case "POST":
		generateAPIKeyHandler(w, r)
	case "DELETE":
		deleteAPIKeyHandler(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func BaseEndpoint(w http.ResponseWriter, r *http.Request) {
	if err := db.PingDB(); err != nil {
		log.Info().Msgf("Health check failed: %v", err) // Log the error
		sendJSONResponse(w, http.StatusServiceUnavailable, "Database is currently not reachable from the server")
		return
	}
	// The database is up and reachable.
	sendJSONResponse(w, http.StatusOK, "Welcome to Doku Ingester - Service operational")
}
