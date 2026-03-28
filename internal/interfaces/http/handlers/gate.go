package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/pedrobarco/mroki/internal/application/commands"
	"github.com/pedrobarco/mroki/internal/application/queries"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/dto"
)

// Type alias for backward compatibility

func CreateGate(handler *commands.CreateGateHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var req struct {
			LiveURL   string `json:"live_url"`
			ShadowURL string `json:"shadow_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return dto.InvalidRequestBody(err)
		}

		if req.LiveURL == "" {
			return dto.MissingBodyProperty("live_url")
		}

		if req.ShadowURL == "" {
			return dto.MissingBodyProperty("shadow_url")
		}

		cmd := commands.CreateGateCommand{
			LiveURL:   req.LiveURL,
			ShadowURL: req.ShadowURL,
		}

		gate, err := handler.Handle(r.Context(), cmd)
		if err != nil {
			switch {
			case errors.Is(err, traffictesting.ErrInvalidGateURL):
				return dto.InvalidGateURL(err)
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
			Data: dto.Gate{
				ID:        gate.ID.String(),
				LiveURL:   gate.LiveURL.String(),
				ShadowURL: gate.ShadowURL.String(),
			},
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

		gate, err := handler.Handle(r.Context(), query)
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
			Data: dto.Gate{
				ID:        gate.ID.String(),
				LiveURL:   gate.LiveURL.String(),
				ShadowURL: gate.ShadowURL.String(),
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

func GetAllGates(handler *queries.ListGatesHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// Parse HTTP query to primitives
		limit, offset, err := parsePaginationQueryParams(r.URL.Query())
		if err != nil {
			return dto.InvalidGatePagination(err)
		}

		// Parse filtering and sorting query parameters
		liveURL, shadowURL, sortField, sortOrder := parseGateQueryParams(r.URL.Query())

		query := queries.ListGatesQuery{
			Limit:     limit,
			Offset:    offset,
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
		for _, gate := range result.Items {
			data = append(data, dto.Gate{
				ID:        gate.ID.String(),
				LiveURL:   gate.LiveURL.String(),
				ShadowURL: gate.ShadowURL.String(),
			})
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
