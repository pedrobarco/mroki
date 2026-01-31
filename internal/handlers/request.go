package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/diffing"
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

func CreateRequest(svc *diffing.RequestService) AppHandler {
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

		// Build props with primitive strings - service handles parsing
		props := diffing.CreateRequestProps{
			ID:        req.ID,
			GateID:    gateIDStr,
			AgentID:   req.AgentID,
			Method:    req.Method,
			Path:      req.Path,
			Headers:   req.Headers,
			Body:      []byte(req.Body),
			CreatedAt: req.CreatedAt,
		}

		// Pass response IDs as strings
		for _, resp := range req.Responses {
			props.Responses = append(props.Responses, diffing.CreateRequestResponseProps{
				ID:         resp.ID,
				Type:       resp.Type,
				StatusCode: resp.StatusCode,
				Headers:    resp.Headers,
				Body:       []byte(resp.Body),
				CreatedAt:  resp.CreatedAt,
			})
		}

		props.Diff = diffing.CreateRequestDiffProps{
			Content: req.Diff.Content,
		}

		// Service handles all parsing internally
		request, err := svc.Create(r.Context(), props)
		if err != nil {
			// Map domain errors to HTTP status codes
			switch {
			case errors.Is(err, diffing.ErrInvalidGateID):
				return NewError(http.StatusBadRequest, "invalid gate ID", err)
			case errors.Is(err, diffing.ErrInvalidRequestID):
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

func GetRequestByID(svc *diffing.RequestService) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		gid := r.PathValue("gate_id")
		if gid == "" {
			return MissingPathParam("gate_id")
		}

		rid := r.PathValue("request_id")
		if rid == "" {
			return MissingPathParam("request_id")
		}

		req, err := svc.GetByID(r.Context(), rid, gid)
		if err != nil {
			switch {
			case errors.Is(err, diffing.ErrInvalidRequestID):
				return NewError(http.StatusBadRequest, "invalid request ID", err)
			case errors.Is(err, diffing.ErrInvalidGateID):
				return NewError(http.StatusBadRequest, "invalid gate ID", err)
			case errors.Is(err, diffing.ErrRequestNotFound):
				return NewError(http.StatusNotFound, "request not found", err)
			case errors.Is(err, diffing.ErrGateNotFound):
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

func GetAllRequestsByGateID(svc *diffing.RequestService) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		gid := r.PathValue("gate_id")
		if gid == "" {
			return MissingPathParam("gate_id")
		}

		requests, err := svc.GetAllByGateID(r.Context(), gid)
		if err != nil {
			switch {
			case errors.Is(err, diffing.ErrInvalidGateID):
				return NewError(http.StatusBadRequest, "invalid gate ID", err)
			default:
				return NewError(http.StatusInternalServerError, "failed to retrieve requests", err)
			}
		}

		response := responseDTO[[]requestResponseDTO]{
			Data: make([]requestResponseDTO, len(requests)),
		}

		for i, req := range requests {
			response.Data[i] = toRequestResponseDTO(req)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return InvalidResponseBody(err)
		}

		return nil
	}
}

func toRequestResponseDTO(req *diffing.Request) requestResponseDTO {
	return requestResponseDTO{
		ID:        req.ID.String(),
		Method:    req.Method,
		Path:      req.Path,
		CreatedAt: req.CreatedAt,
	}
}

func toFullRequestResponseDTO(req *diffing.Request) fullRequestResponseDTO {
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
