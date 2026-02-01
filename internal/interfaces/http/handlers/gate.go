package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/pedrobarco/mroki/internal/application/commands"
	"github.com/pedrobarco/mroki/internal/application/queries"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

type gateResponseDTO struct {
	ID        string `json:"id"`
	LiveURL   string `json:"live_url"`
	ShadowURL string `json:"shadow_url"`
}

func CreateGate(handler *commands.CreateGateHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var req struct {
			LiveURL   string `json:"live_url"`
			ShadowURL string `json:"shadow_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return InvalidRequestBody(err)
		}

		if req.LiveURL == "" {
			return MissingBodyProperty("live_url")
		}

		if req.ShadowURL == "" {
			return MissingBodyProperty("shadow_url")
		}

		cmd := commands.CreateGateCommand{
			LiveURL:   req.LiveURL,
			ShadowURL: req.ShadowURL,
		}

		gate, err := handler.Handle(r.Context(), cmd)
		if err != nil {
			switch {
			case errors.Is(err, traffictesting.ErrInvalidGateURL):
				return NewError(http.StatusBadRequest, "invalid URL", err)
			default:
				return NewError(http.StatusInternalServerError, "failed to create gate", err)
			}
		}

		response := responseDTO[gateResponseDTO]{
			Data: gateResponseDTO{
				ID:        gate.ID.String(),
				LiveURL:   gate.LiveURL.String(),
				ShadowURL: gate.ShadowURL.String(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return InvalidResponseBody(err)
		}
		return nil
	}
}

func GetGateByID(handler *queries.GetGateHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("gate_id")
		if id == "" {
			return MissingPathParam("gate_id")
		}

		query := queries.GetGateQuery{
			ID: id,
		}

		gate, err := handler.Handle(r.Context(), query)
		if err != nil {
			switch {
			case errors.Is(err, traffictesting.ErrInvalidGateID):
				return NewError(http.StatusBadRequest, "invalid gate ID", err)
			case errors.Is(err, traffictesting.ErrGateNotFound):
				return NewError(http.StatusNotFound, "gate not found", err)
			default:
				return NewError(http.StatusInternalServerError, "failed to retrieve gate", err)
			}
		}

		response := responseDTO[gateResponseDTO]{
			Data: gateResponseDTO{
				ID:        gate.ID.String(),
				LiveURL:   gate.LiveURL.String(),
				ShadowURL: gate.ShadowURL.String(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return InvalidResponseBody(err)
		}
		return nil
	}
}

func GetAllGates(handler *queries.ListGatesHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// Parse HTTP query to primitives
		limit, offset, err := parsePaginationQueryParams(r.URL.Query())
		if err != nil {
			return NewError(http.StatusBadRequest, "invalid pagination parameters", err)
		}

		query := queries.ListGatesQuery{
			Limit:  limit,
			Offset: offset,
		}

		result, err := handler.Handle(r.Context(), query)
		if err != nil {
			// Check if it's a pagination validation error
			switch {
			case errors.Is(err, traffictesting.ErrInvalidPagination):
				return NewError(http.StatusBadRequest, "invalid pagination parameters", err)
			default:
				return NewError(http.StatusInternalServerError, "failed to retrieve gates", err)
			}
		}

		// Map domain entities to DTOs (empty slice for empty results)
		data := make([]gateResponseDTO, 0, len(result.Items))
		for _, gate := range result.Items {
			data = append(data, gateResponseDTO{
				ID:        gate.ID.String(),
				LiveURL:   gate.LiveURL.String(),
				ShadowURL: gate.ShadowURL.String(),
			})
		}

		// Map PagedResult to response DTO
		response := paginatedResponseDTO[[]gateResponseDTO]{
			Data: data,
			Pagination: paginationMetaDTO{
				Limit:   result.Limit,
				Offset:  result.Offset,
				Total:   result.Total,
				HasMore: result.HasMore,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return InvalidResponseBody(err)
		}
		return nil
	}
}
