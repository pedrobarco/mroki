package handlers

import (
	"encoding/json"
	"fmt"
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

func CreateRequest(svc *diffing.RequestService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID        uuid.UUID           `json:"id,omitempty"`
			Method    string              `json:"method"`
			Path      string              `json:"path"`
			Headers   map[string][]string `json:"headers"`
			Body      []byte              `json:"body"`
			CreatedAt time.Time           `json:"created_at"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fmt.Printf("Error decoding request body: %v\n", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		gateID, err := uuid.Parse(r.PathValue("gate_id"))
		if err != nil {
			http.Error(w, "Invalid gate ID format", http.StatusBadRequest)
			return
		}

		request, err := svc.Create(r.Context(), diffing.CreateRequestProps{
			ID:        req.ID,
			GateID:    gateID,
			Method:    req.Method,
			Path:      req.Path,
			Headers:   req.Headers,
			Body:      req.Body,
			CreatedAt: req.CreatedAt,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := responseDTO[requestResponseDTO]{
			Data: toRequestResponseDTO(request),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func GetRequestByID(svc *diffing.RequestService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gid := r.PathValue("gate_id")
		if gid == "" {
			http.Error(w, "Gate ID is required", http.StatusBadRequest)
			return
		}

		gateID, err := uuid.Parse(gid)
		if err != nil {
			http.Error(w, "Invalid Gate ID format", http.StatusBadRequest)
			return
		}

		rid := r.PathValue("request_id")
		if rid == "" {
			http.Error(w, "Request ID is required", http.StatusBadRequest)
			return
		}

		requestID, err := uuid.Parse(rid)
		if err != nil {
			http.Error(w, "Invalid Request ID format", http.StatusBadRequest)
			return
		}

		req, err := svc.GetByID(r.Context(), requestID, gateID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := responseDTO[requestResponseDTO]{
			Data: toRequestResponseDTO(req),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func GetAllRequestsByGateID(svc *diffing.RequestService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gid := r.PathValue("gate_id")
		if gid == "" {
			http.Error(w, "Gate ID is required", http.StatusBadRequest)
			return
		}

		gateID, err := uuid.Parse(gid)
		if err != nil {
			http.Error(w, "Invalid Gate ID format", http.StatusBadRequest)
			return
		}

		requests, err := svc.GetAllByGateID(r.Context(), gateID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
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
