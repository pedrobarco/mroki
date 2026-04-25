package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/pedrobarco/mroki/internal/application/commands"
	"github.com/pedrobarco/mroki/internal/application/queries"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/dto"
)

// Type alias for backward compatibility

func CreateGate(handler *commands.CreateGateHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var req struct {
			Name      string `json:"name"`
			LiveURL   string `json:"live_url"`
			ShadowURL string `json:"shadow_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return dto.InvalidRequestBody(err)
		}

		if req.Name == "" {
			return dto.MissingBodyProperty("name")
		}

		if req.LiveURL == "" {
			return dto.MissingBodyProperty("live_url")
		}

		if req.ShadowURL == "" {
			return dto.MissingBodyProperty("shadow_url")
		}

		cmd := commands.CreateGateCommand{
			Name:      req.Name,
			LiveURL:   req.LiveURL,
			ShadowURL: req.ShadowURL,
		}

		gate, err := handler.Handle(r.Context(), cmd)
		if err != nil {
			switch {
			case errors.Is(err, traffictesting.ErrInvalidGateName):
				return dto.InvalidGateName(err)
			case errors.Is(err, traffictesting.ErrInvalidGateURL):
				return dto.InvalidGateURL(err)
			case errors.Is(err, traffictesting.ErrDuplicateGateName):
				return dto.DuplicateGateName(err)
			case errors.Is(err, traffictesting.ErrDuplicateGateURLs):
				return dto.DuplicateGateURLs(err)
			default:
				return dto.NewError(
					http.StatusInternalServerError,
					dto.ErrorTypeInternalError,
					"Internal Server Error",
					"An unknown error occurred. Please try again later.",
					err,
				)
			}
		}

		response := dto.Response[dto.Gate]{
			Data: mapGateToDTO(&queries.GateWithStats{Gate: gate}),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return dto.InvalidResponseBody(err)
		}
		return nil
	}
}

func GetGateByID(handler *queries.GetGateHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("gate_id")
		if id == "" {
			return dto.MissingPathParam("gate_id")
		}

		query := queries.GetGateQuery{
			ID: id,
		}

		result, err := handler.Handle(r.Context(), query)
		if err != nil {
			switch {
			case errors.Is(err, traffictesting.ErrInvalidGateID):
				return dto.InvalidGateID(id)
			case errors.Is(err, traffictesting.ErrGateNotFound):
				return dto.GateNotFound(id)
			default:
				return dto.NewError(
					http.StatusInternalServerError,
					dto.ErrorTypeInternalError,
					"Internal Server Error",
					"An unknown error occurred. Please try again later.",
					err,
				)
			}
		}

		response := dto.Response[dto.Gate]{
			Data: mapGateToDTO(result),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return dto.InvalidResponseBody(err)
		}
		return nil
	}
}

func GetAllGates(handler *queries.ListGatesHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// Parse HTTP query to primitives
		limit, offset, err := parsePaginationQueryParams(r.URL.Query())
		if err != nil {
			return dto.InvalidGatePagination(err)
		}

		// Parse filtering and sorting query parameters
		name, liveURL, shadowURL, sortField, sortOrder := parseGateQueryParams(r.URL.Query())

		query := queries.ListGatesQuery{
			Limit:     limit,
			Offset:    offset,
			Name:      name,
			LiveURL:   liveURL,
			ShadowURL: shadowURL,
			SortField: sortField,
			SortOrder: sortOrder,
		}

		result, err := handler.Handle(r.Context(), query)
		if err != nil {
			switch {
			case errors.Is(err, traffictesting.ErrInvalidPagination):
				return dto.InvalidGatePagination(err)
			case errors.Is(err, traffictesting.ErrInvalidGateSort):
				return dto.InvalidGateSort(err)
			default:
				return dto.NewError(
					http.StatusInternalServerError,
					dto.ErrorTypeInternalError,
					"Internal Server Error",
					"An unknown error occurred. Please try again later.",
					err,
				)
			}
		}

		// Map domain entities to DTOs (empty slice for empty results)
		data := make([]dto.Gate, 0, len(result.Items))
		for _, gws := range result.Items {
			data = append(data, mapGateToDTO(gws))
		}

		// Map PagedResult to response DTO
		response := dto.PaginatedResponse[[]dto.Gate]{
			Data: data,
			Pagination: dto.PaginationMeta{
				Limit:   result.Limit,
				Offset:  result.Offset,
				Total:   result.Total,
				HasMore: result.HasMore,
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


func mapGateToDTO(gws *queries.GateWithStats) dto.Gate {
	var lastActive *string
	if gws.Stats.LastActive != nil {
		t := gws.Stats.LastActive.Format(time.RFC3339)
		lastActive = &t
	}

	return dto.Gate{
		ID:        gws.Gate.ID.String(),
		Name:      gws.Gate.Name.String(),
		LiveURL:   gws.Gate.LiveURL.String(),
		ShadowURL: gws.Gate.ShadowURL.String(),
		CreatedAt: gws.Gate.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Stats: dto.GateStats{
			RequestCount24h: gws.Stats.RequestCount24h,
			DiffCount24h:    gws.Stats.DiffCount24h,
			DiffRate:        gws.Stats.DiffRate,
			LastActive:      lastActive,
		},
	}
}