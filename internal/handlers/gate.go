package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
)

type gateResponseDTO struct {
	ID        string `json:"id"`
	LiveURL   string `json:"live_url"`
	ShadowURL string `json:"shadow_url"`
}

var (
	ErrInvalidGateID = errors.New("invalid gate ID format")
)

func CreateGate(svc *diffing.GateService) AppHandler {
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

		gate, err := svc.Create(r.Context(), req.LiveURL, req.ShadowURL)
		if err != nil {
			return NewError(http.StatusInternalServerError, "failed to create gate", err)
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

func GetGateByID(svc *diffing.GateService) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("gate_id")
		if id == "" {
			err := MissingPathParam("gate_id")
			return NewError(http.StatusBadRequest, err.Error(), err)
		}

		// TODO: refactor to be handled by svc.GetByID
		gateID, err := uuid.Parse(id)
		if err != nil {
			return NewError(http.StatusBadRequest, ErrInvalidGateID.Error(), err)
		}

		gate, err := svc.GetByID(r.Context(), gateID)
		if err != nil {
			if errors.Is(err, diffing.ErrGateNotFound) {
				return NewError(http.StatusNotFound, "gate not found", err)
			}
			return NewError(http.StatusInternalServerError, "failed to retrieve gate", err)
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

func GetAllGates(svc *diffing.GateService) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		gates, err := svc.GetAll(r.Context())
		if err != nil {
			return NewError(http.StatusInternalServerError, "failed to retrieve gates", err)
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
			return InvalidResponseBody(err)
		}
		return nil
	}
}
