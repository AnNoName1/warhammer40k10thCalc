package handler

import (
	"encoding/json"
	"net/http"
)

// APIError defines a standardized JSON error response.
type APIError struct {
	Message     string `json:"message"`
	RequestUUID string `json:"request_uuid"`
}

// SendError sends a standardized JSON error response.
func SendError(w http.ResponseWriter, reqID string, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	errResp := APIError{
		Message:     message,
		RequestUUID: reqID,
	}

	json.NewEncoder(w).Encode(errResp)
}
