package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// writeJSON encodes v as a JSON 200 OK response.
// Encode failures are logged — the response is already partially flushed
// at that point, so retrying or returning an error has nowhere useful to go.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode JSON response", "error", err)
	}
}
