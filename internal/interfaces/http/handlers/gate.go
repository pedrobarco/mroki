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

func DeleteGate(handler *commands.DeleteGateHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("gate_id")
		if id == "" {
			return dto.MissingPathParam("gate_id")
		}

		cmd := commands.DeleteGateCommand{
			ID: id,
		}

		if err := handler.Handle(r.Context(), cmd); err != nil {
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

		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

func UpdateGate(handler *commands.UpdateGateHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("gate_id")
		if id == "" {
			return dto.MissingPathParam("gate_id")
		}

		var req struct {
			Name           *string         `json:"name"`
			DiffConfig     *dto.DiffConfig `json:"diff_config"`
			RedactedFields *[]string       `json:"redacted_fields"`
			Retention      json.RawMessage `json:"retention"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return dto.InvalidRequestBody(err)
		}

		// Decode retention as a tri-state:
		//   - absent key      -> leave the current value unchanged (nil)
		//   - JSON null        -> reset to the global retention floor ("")
		//   - JSON string ""   -> reset to the global retention floor ("")
		//   - JSON string "…"  -> set a custom retention
		retention, err := decodeRetention(req.Retention)
		if err != nil {
			return err
		}

		cmd := commands.UpdateGateCommand{
			ID:        id,
			Name:      req.Name,
			Retention: retention,
		}

		if req.DiffConfig != nil {
			cmd.DiffConfig = &commands.UpdateDiffConfigProps{
				IgnoredFields:  req.DiffConfig.IgnoredFields,
				IncludedFields: req.DiffConfig.IncludedFields,
				FloatTolerance: req.DiffConfig.FloatTolerance,
				SortArrays:     req.DiffConfig.SortArrays,
			}
		}

		if req.RedactedFields != nil {
			cmd.RedactedFields = &commands.UpdateRedactedFieldsProps{
				AdditionalFields: *req.RedactedFields,
			}
		}

		gate, err := handler.Handle(r.Context(), cmd)
		if err != nil {
			switch {
			case errors.Is(err, traffictesting.ErrInvalidGateID):
				return dto.InvalidGateID(id)
			case errors.Is(err, traffictesting.ErrGateNotFound):
				return dto.GateNotFound(id)
			case errors.Is(err, traffictesting.ErrInvalidGateName):
				return dto.InvalidGateName(err)
			case errors.Is(err, traffictesting.ErrInvalidDiffConfig):
				return dto.InvalidDiffConfig(err)
			case errors.Is(err, traffictesting.ErrInvalidRedactedFields):
				return dto.InvalidRedactedFields(err)
			case errors.Is(err, traffictesting.ErrInvalidRetention):
				return dto.InvalidRetention(err)
			case errors.Is(err, traffictesting.ErrRetentionBelowMinimum):
				return dto.RetentionBelowMinimum(err)
			case errors.Is(err, traffictesting.ErrDuplicateGateName):
				return dto.DuplicateGateName(err)
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
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return dto.InvalidResponseBody(err)
		}
		return nil
	}
}

// decodeRetention interprets the raw JSON value of the "retention" field as a
// tri-state command value. An absent field (nil raw) returns nil (unchanged).
// A JSON null or an empty string returns a pointer to "" (reset to the global
// floor). A JSON string returns a pointer to its value (set custom). Any other
// JSON type is rejected as an invalid request body.
func decodeRetention(raw json.RawMessage) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	if string(raw) == "null" {
		empty := ""
		return &empty, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, dto.InvalidRequestBody(err)
	}
	return &s, nil
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
		DiffConfig: dto.DiffConfig{
			IgnoredFields:  gws.Gate.DiffConfig.IgnoredFields,
			IncludedFields: gws.Gate.DiffConfig.IncludedFields,
			FloatTolerance: gws.Gate.DiffConfig.FloatTolerance,
			SortArrays:     gws.Gate.DiffConfig.SortArrays,
		},
		RedactedFields: gws.Gate.RedactedFields.AdditionalFields,
		Retention:      gws.Gate.Retention.String(),
		CreatedAt:      gws.Gate.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Stats: dto.GateStats{
			RequestCount24h: gws.Stats.RequestCount24h,
			DiffCount24h:    gws.Stats.DiffCount24h,
			DiffRate:        gws.Stats.DiffRate,
			LastActive:      lastActive,
		},
	}
}
