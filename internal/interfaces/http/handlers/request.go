package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/pedrobarco/mroki/internal/application/commands"
	"github.com/pedrobarco/mroki/internal/application/queries"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

type requestResponseDTO struct {
	ID        string    `json:"id"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
}

type fullRequestResponseDTO struct {
	ID        string    `json:"id"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`

	Responses []responseResponseDTO `json:"responses"`
	Diff      diffResponseDTO       `json:"diff"`
}

type responseResponseDTO struct {
	ID         string      `json:"id"`
	Type       string      `json:"type"`
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers"`
	Body       string      `json:"body"`
	CreatedAt  time.Time   `json:"created_at"`
}

type diffResponseDTO struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

func CreateRequest(handler *commands.CreateRequestHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var req struct {
			ID        string              `json:"id,omitempty"`
			AgentID   string              `json:"agent_id"`
			Method    string              `json:"method"`
			Path      string              `json:"path"`
			Headers   map[string][]string `json:"headers"`
			Body      string              `json:"body"`
			CreatedAt time.Time           `json:"created_at"`

			Responses []struct {
				ID         string      `json:"id,omitempty"`
				Type       string      `json:"type"`
				StatusCode int         `json:"status_code"`
				Headers    http.Header `json:"headers"`
				Body       string      `json:"body"`
				CreatedAt  time.Time   `json:"created_at"`
			} `json:"responses"`

			Diff struct {
				Content string `json:"content"`
			} `json:"diff"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return InvalidRequestBody(err)
		}

		gateIDStr := r.PathValue("gate_id")
		if gateIDStr == "" {
			return MissingPathParam("gate_id")
		}

		// Build command
		cmd := commands.CreateRequestCommand{
			ID:        req.ID,
			GateID:    gateIDStr,
			AgentID:   req.AgentID,
			Method:    req.Method,
			Path:      req.Path,
			Headers:   req.Headers,
			Body:      []byte(req.Body),
			CreatedAt: req.CreatedAt,
		}

		// Map responses
		for _, resp := range req.Responses {
			cmd.Responses = append(cmd.Responses, commands.CreateRequestResponseProps{
				ID:         resp.ID,
				Type:       resp.Type,
				StatusCode: resp.StatusCode,
				Headers:    resp.Headers,
				Body:       []byte(resp.Body),
				CreatedAt:  resp.CreatedAt,
			})
		}

		cmd.Diff = commands.CreateRequestDiffProps{
			Content: req.Diff.Content,
		}

		// Execute command
		request, err := handler.Handle(r.Context(), cmd)
		if err != nil {
			// Map domain errors to HTTP status codes
			switch {
			case errors.Is(err, traffictesting.ErrInvalidGateID):
				return NewError(http.StatusBadRequest, "invalid gate ID", err)
			case errors.Is(err, traffictesting.ErrInvalidRequestID):
				return NewError(http.StatusBadRequest, "invalid request ID", err)
			default:
				return NewError(http.StatusInternalServerError, "failed to create request", err)
			}
		}

		resp := responseDTO[requestResponseDTO]{
			Data: toRequestResponseDTO(request),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			return InvalidResponseBody(err)
		}

		return nil
	}
}

func GetRequestByID(handler *queries.GetRequestHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		gid := r.PathValue("gate_id")
		if gid == "" {
			return MissingPathParam("gate_id")
		}

		rid := r.PathValue("request_id")
		if rid == "" {
			return MissingPathParam("request_id")
		}

		query := queries.GetRequestQuery{
			ID:     rid,
			GateID: gid,
		}

		req, err := handler.Handle(r.Context(), query)
		if err != nil {
			switch {
			case errors.Is(err, traffictesting.ErrInvalidRequestID):
				return NewError(http.StatusBadRequest, "invalid request ID", err)
			case errors.Is(err, traffictesting.ErrInvalidGateID):
				return NewError(http.StatusBadRequest, "invalid gate ID", err)
			case errors.Is(err, traffictesting.ErrRequestNotFound):
				return NewError(http.StatusNotFound, "request not found", err)
			case errors.Is(err, traffictesting.ErrGateNotFound):
				return NewError(http.StatusNotFound, "gate not found", err)
			default:
				return NewError(http.StatusInternalServerError, "failed to retrieve request", err)
			}
		}

		response := responseDTO[fullRequestResponseDTO]{
			Data: toFullRequestResponseDTO(req),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return InvalidResponseBody(err)
		}

		return nil
	}
}

func GetAllRequestsByGateID(handler *queries.ListRequestsHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		gid := r.PathValue("gate_id")
		if gid == "" {
			return MissingPathParam("gate_id")
		}

		// Parse pagination query parameters
		limit, offset, err := parsePaginationQueryParams(r.URL.Query())
		if err != nil {
			return NewError(http.StatusBadRequest, "invalid pagination parameters", err)
		}

		query := queries.ListRequestsQuery{
			GateID: gid,
			Limit:  limit,
			Offset: offset,
		}

		result, err := handler.Handle(r.Context(), query)
		if err != nil {
			switch {
			case errors.Is(err, traffictesting.ErrInvalidGateID):
				return NewError(http.StatusBadRequest, "invalid gate ID", err)
			case errors.Is(err, traffictesting.ErrInvalidPagination):
				return NewError(http.StatusBadRequest, "invalid pagination parameters", err)
			default:
				return NewError(http.StatusInternalServerError, "failed to retrieve requests", err)
			}
		}

		// Map domain entities to DTOs
		data := make([]requestResponseDTO, 0, len(result.Items))
		for _, req := range result.Items {
			data = append(data, toRequestResponseDTO(req))
		}

		// Use paginated response DTO
		response := paginatedResponseDTO[[]requestResponseDTO]{
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

func toRequestResponseDTO(req *traffictesting.Request) requestResponseDTO {
	return requestResponseDTO{
		ID:        req.ID.String(),
		Method:    req.Method,
		Path:      req.Path,
		CreatedAt: req.CreatedAt,
	}
}

func toFullRequestResponseDTO(req *traffictesting.Request) fullRequestResponseDTO {
	dto := fullRequestResponseDTO{
		ID:        req.ID.String(),
		Method:    req.Method,
		Path:      req.Path,
		CreatedAt: req.CreatedAt,
	}

	for _, resp := range req.Responses {
		dto.Responses = append(dto.Responses, responseResponseDTO{
			ID:         resp.ID.String(),
			Type:       string(resp.Type),
			StatusCode: resp.StatusCode,
			Headers:    resp.Headers,
			Body:       string(resp.Body),
			CreatedAt:  resp.CreatedAt,
		})
	}

	dto.Diff = diffResponseDTO{
		ID:      req.Diff.ID.String(),
		Content: req.Diff.Content,
	}

	return dto
}
