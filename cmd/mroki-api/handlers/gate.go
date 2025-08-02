package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
)

type gateResponseDTO struct {
	ID        string `json:"id"`
	LiveURL   string `json:"live_url"`
	ShadowURL string `json:"shadow_url"`
}

func CreateGate(svc *diffing.GateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			LiveURL   string `json:"live_url"`
			ShadowURL string `json:"shadow_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		gate, err := svc.Create(r.Context(), req.LiveURL, req.ShadowURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func GetGateByID(svc *diffing.GateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "ID is required", http.StatusBadRequest)
			return
		}

		gateID, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid ID format", http.StatusBadRequest)
			return
		}

		gate, err := svc.GetByID(r.Context(), gateID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func GetAllGates(svc *diffing.GateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gates, err := svc.GetAll(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := make([]gateResponseDTO, len(gates))
		for i, gate := range gates {
			data[i] = gateResponseDTO{
				ID:        gate.ID.String(),
				LiveURL:   gate.LiveURL.String(),
				ShadowURL: gate.ShadowURL.String(),
			}
		}

		response := responseDTO[[]gateResponseDTO]{
			Data: data,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
