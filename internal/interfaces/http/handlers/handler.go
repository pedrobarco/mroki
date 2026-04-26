package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/pedrobarco/mroki/internal/interfaces/http/middleware"
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

		// Clone to avoid modifying shared static error instances
		apiErrCopy := *apiErr

		// Auto-populate instance field for 4xx errors only
		if apiErrCopy.Status >= 400 && apiErrCopy.Status < 500 && apiErrCopy.Instance == "" {
			apiErrCopy.Instance = r.URL.Path
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(apiErrCopy.Status)

		slog.Error("API error",
			slog.String("request.id", middleware.GetRequestID(r.Context())),
			slog.String("error.type", apiErrCopy.Type),
			slog.String("error.title", apiErrCopy.Title),
			slog.Int("error.status", apiErrCopy.Status),
			slog.String("error.detail", apiErrCopy.Detail),
			slog.String("error.instance", apiErrCopy.Instance),
			slog.String("error.error", apiErrCopy.Error()),
		)

		if err := json.NewEncoder(w).Encode(apiErrCopy); err != nil {
			http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
			slog.Error("Failed to encode error response",
				slog.String("error", err.Error()),
			)
			return
		}
	}
}
