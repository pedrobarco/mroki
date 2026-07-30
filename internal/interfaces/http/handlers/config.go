package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/pedrobarco/mroki/pkg/dto"
)

// GetConfig returns the read-only, server-wide settings the hub needs. The
// global retention floor is injected at wiring time rather than read from a
// repository, since it is static API configuration.
func GetConfig(retention time.Duration) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		response := dto.Response[dto.Config]{
			Data: dto.Config{
				Retention: retention.String(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return dto.InvalidResponseBody(err)
		}
		return nil
	}
}
