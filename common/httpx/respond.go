package httpx

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response with the provided status code.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload) //nolint:errcheck // best-effort response write
}

// Error writes a JSON error response using a standard envelope.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
