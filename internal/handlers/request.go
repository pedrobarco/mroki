package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
)

type requestResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	ErrInvalidRequestID = errors.New("invalid request ID format")
)

func CreateRequest(svc *diffing.RequestService) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var req struct {
			ID        uuid.UUID           `json:"id,omitempty"`
			Method    string              `json:"method"`
			Path      string              `json:"path"`
			Headers   map[string][]string `json:"headers"`
			Body      string              `json:"body"`
			CreatedAt time.Time           `json:"created_at"`

			Responses []struct {
				ID         uuid.UUID   `json:"id,omitempty"`
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

		gateID, err := uuid.Parse(r.PathValue("gate_id"))
		if err != nil {
			return MissingPathParam("gate_id")
		}

		props := diffing.CreateRequestProps{
			ID:        req.ID,
			GateID:    gateID,
			Method:    req.Method,
			Path:      req.Path,
			Headers:   req.Headers,
			Body:      []byte(req.Body),
			CreatedAt: req.CreatedAt,
		}

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

		request, err := svc.Create(r.Context(), props)
		if err != nil {
			return NewError(http.StatusInternalServerError, "failed to create request", err)
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

		// TODO: refactor to be handled by svc.GetByID
		gateID, err := uuid.Parse(gid)
		if err != nil {
			return NewError(http.StatusBadRequest, ErrInvalidGateID.Error(), err)
		}

		rid := r.PathValue("request_id")
		if rid == "" {
			return MissingPathParam("request_id")
		}

		// TODO: refactor to be handled by svc.GetByID
		requestID, err := uuid.Parse(rid)
		if err != nil {
			return NewError(http.StatusBadRequest, ErrInvalidRequestID.Error(), err)
		}

		req, err := svc.GetByID(r.Context(), requestID, gateID)
		if err != nil {
			return NewError(http.StatusInternalServerError, "failed to retrieve request", err)
		}

		response := responseDTO[requestResponseDTO]{
			Data: toRequestResponseDTO(req),
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

		// TODO: refactor to be handled by svc.GetAllByGateID
		gateID, err := uuid.Parse(gid)
		if err != nil {
			return NewError(http.StatusBadRequest, ErrInvalidGateID.Error(), err)
		}

		requests, err := svc.GetAllByGateID(r.Context(), gateID)
		if err != nil {
			return NewError(http.StatusInternalServerError, "failed to retrieve requests", err)
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
		ID:        req.ID,
		Method:    req.Method,
		Path:      req.Path,
		CreatedAt: req.CreatedAt,
	}
}
