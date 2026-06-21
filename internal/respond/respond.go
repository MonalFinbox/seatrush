// Package respond centralizes how we write JSON to the wire so every handler
// produces the same success and error shapes.
package respond

import (
	"encoding/json"
	"log"
	"net/http"
)

// JSON writes any payload as JSON with the given status code.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("respond: failed to encode JSON: %v", err)
	}
}

// errorBody is the single error envelope used everywhere: { "error": "..." }.
type errorBody struct {
	Error string `json:"error"`
}

// Error writes a JSON error with the given status code.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, errorBody{Error: message})
}
