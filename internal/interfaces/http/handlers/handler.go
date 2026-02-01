package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type AppHandler func(http.ResponseWriter, *http.Request) error

func (fn AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil {
		apiErr, ok := err.(*APIError)
		if !ok {
			apiErr = NewError(http.StatusInternalServerError, unknownErrorMessage, err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(apiErr.StatusCode)

		slog.Error("API error",
			slog.String("error.code", apiErr.Code),
			slog.String("error.message", apiErr.Message),
			slog.String("error.error", apiErr.Error()),
		)

		if err := json.NewEncoder(w).Encode(apiErr); err != nil {
			http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
			slog.Error("Failed to encode error response",
				slog.String("error", err.Error()),
			)
			return
		}
	}
}
