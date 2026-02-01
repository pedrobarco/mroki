package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/pedrobarco/mroki/pkg/dto"
)

type AppHandler func(http.ResponseWriter, *http.Request) error

func (fn AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil {
		apiErr, ok := err.(*dto.APIError)
		if !ok {
			apiErr = dto.NewError(
				http.StatusInternalServerError,
				dto.ErrorTypeInternalError,
				"Internal Server Error",
				"An unknown error occurred. Please try again later.",
				err,
			)
		}

		// Auto-populate instance field for 4xx errors only
		if apiErr.Status >= 400 && apiErr.Status < 500 && apiErr.Instance == "" {
			apiErr.Instance = r.URL.Path
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(apiErr.Status)

		slog.Error("API error",
			slog.String("error.type", apiErr.Type),
			slog.String("error.title", apiErr.Title),
			slog.Int("error.status", apiErr.Status),
			slog.String("error.detail", apiErr.Detail),
			slog.String("error.instance", apiErr.Instance),
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
