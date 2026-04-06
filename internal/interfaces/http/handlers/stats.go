package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/pedrobarco/mroki/internal/application/queries"
	"github.com/pedrobarco/mroki/pkg/dto"
)

func GetGlobalStats(handler *queries.GetGlobalStatsHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		stats, err := handler.Handle(r.Context())
		if err != nil {
			return dto.NewError(
				http.StatusInternalServerError,
				dto.ErrorTypeInternalError,
				"Internal Server Error",
				"Failed to retrieve global statistics.",
				err,
			)
		}

		response := dto.Response[dto.GlobalStats]{
			Data: dto.GlobalStats{
				TotalGates:       stats.TotalGates,
				TotalRequests24h: stats.TotalRequests24h,
				TotalDiffRate:    stats.TotalDiffRate,
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
