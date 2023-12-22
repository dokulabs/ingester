package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ingester/auth"
	"ingester/db"

	"github.com/rs/zerolog/log"
)

type APIKeyRequest struct {
	Name string `json:"name"`
}

func decodeRequestBody(r *http.Request, dest interface{}) error {
	return json.NewDecoder(r.Body).Decode(dest)
}

func getAuthKey(r *http.Request) string {
	return r.Header.Get("Authorization")
}

func (r *APIKeyRequest) Normalize() {
	r.Name = strings.ToLower(r.Name)
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

func handleAPIKeyErrors(w http.ResponseWriter, err error, name string) {
	if err.Error() == "KEYEXISTS" {
		sendJSONResponse(w, http.StatusConflict, fmt.Sprintf("An API Key with the name '%s' already exists", name))
		return
	} else if err.Error() == "AUTHFAILED" {
		sendJSONResponse(w, http.StatusUnauthorized, "Unauthorized: Please check your API Key and try again")
		return
	} else if err.Error() == "NOTFOUND" {
		sendJSONResponse(w, http.StatusNotFound, fmt.Sprintf("Unable to find API Key with the given name %s", name))
		return
	} else {
		sendJSONResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func generateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	var request APIKeyRequest

	if err := decodeRequestBody(r, &request); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	request.Normalize()

	newAPIKey, err := db.GenerateAPIKey(getAuthKey(r), request.Name)
	if err != nil {
		handleAPIKeyErrors(w, err, request.Name)
		return
	}

	sendJSONResponse(w, http.StatusOK, newAPIKey)
	return
}

// getAPIKeyHandler handles retrieving an existing API key.
func getAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	var request APIKeyRequest

	if err := decodeRequestBody(r, &request); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	request.Normalize()

	apiKey, err := db.GetAPIKeyForName(getAuthKey(r), request.Name)
	if err != nil {
		handleAPIKeyErrors(w, err, request.Name)
		return
	}

	sendJSONResponse(w, http.StatusOK, apiKey)
	return
}

func deleteAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	var request APIKeyRequest

	if err := decodeRequestBody(r, &request); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	request.Normalize()
	err := db.DeleteAPIKey(getAuthKey(r), request.Name)
	if err != nil {
		handleAPIKeyErrors(w, err, request.Name)
		return
	}

	sendJSONResponse(w, http.StatusOK, "API key deleted successfully")
	return
}

func DataHandler(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}

	if err := decodeRequestBody(r, &data); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var err error
	data["orgID"], err = auth.AuthenticateRequest(getAuthKey(r))
	if err != nil {
		handleAPIKeyErrors(w, err, "")
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
