package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/pedrobarco/mroki/internal/domain/diffing/mocks"
	"github.com/pedrobarco/mroki/internal/handlers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCreateGate_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockSvc)

	// Mock service call
	mockSvc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	handler := handlers.CreateGate(service)

	body := `{"live_url":"http://live.example.com","shadow_url":"http://shadow.example.com"}`
	req := httptest.NewRequest("POST", "/gates", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "http://live.example.com", data["live_url"])
	assert.Equal(t, "http://shadow.example.com", data["shadow_url"])
	assert.NotEmpty(t, data["id"])
}

func TestCreateGate_invalid_json(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockSvc)
	handler := handlers.CreateGate(service)

	req := httptest.NewRequest("POST", "/gates", bytes.NewBufferString("{invalid json"))
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestCreateGate_missing_live_url(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockSvc)
	handler := handlers.CreateGate(service)

	body := `{"shadow_url":"http://shadow.example.com"}`
	req := httptest.NewRequest("POST", "/gates", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Contains(t, apiErr.Message, "live_url")
}

func TestCreateGate_missing_shadow_url(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockSvc)
	handler := handlers.CreateGate(service)

	body := `{"live_url":"http://live.example.com"}`
	req := httptest.NewRequest("POST", "/gates", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Contains(t, apiErr.Message, "shadow_url")
}

func TestCreateGate_invalid_url(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockSvc)
	handler := handlers.CreateGate(service)

	body := `{"live_url":"ftp://invalid.com","shadow_url":"http://shadow.example.com"}`
	req := httptest.NewRequest("POST", "/gates", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestGetGateByID_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockSvc)

	gateID := diffing.NewGateID()
	liveURL, _ := diffing.ParseGateURL("http://live.example.com")
	shadowURL, _ := diffing.ParseGateURL("http://shadow.example.com")
	gate, _ := diffing.NewGate(liveURL, shadowURL, diffing.WithGateID(gateID))

	mockSvc.EXPECT().GetByID(gomock.Any(), gateID).Return(gate, nil)

	handler := handlers.GetGateByID(service)

	req := httptest.NewRequest("GET", "/gates/"+gateID.String(), nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, gateID.String(), data["id"])
}

func TestGetGateByID_missing_path_param(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockSvc)
	handler := handlers.GetGateByID(service)

	req := httptest.NewRequest("GET", "/gates/", nil)
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestGetGateByID_not_found(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockSvc)

	gateID := diffing.NewGateID()
	mockSvc.EXPECT().GetByID(gomock.Any(), gateID).Return(nil, diffing.ErrGateNotFound)

	handler := handlers.GetGateByID(service)

	req := httptest.NewRequest("GET", "/gates/"+gateID.String(), nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestGetAllGates_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockSvc)

	liveURL1, _ := diffing.ParseGateURL("http://live1.example.com")
	shadowURL1, _ := diffing.ParseGateURL("http://shadow1.example.com")
	gate1, _ := diffing.NewGate(liveURL1, shadowURL1)

	liveURL2, _ := diffing.ParseGateURL("http://live2.example.com")
	shadowURL2, _ := diffing.ParseGateURL("http://shadow2.example.com")
	gate2, _ := diffing.NewGate(liveURL2, shadowURL2)

	gates := []*diffing.Gate{gate1, gate2}
	mockSvc.EXPECT().GetAll(gomock.Any()).Return(gates, nil)

	handler := handlers.GetAllGates(service)

	req := httptest.NewRequest("GET", "/gates", nil)
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&response)
	data := response["data"].([]interface{})
	assert.Len(t, data, 2)
}

func TestGetAllGates_empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockSvc)

	mockSvc.EXPECT().GetAll(gomock.Any()).Return([]*diffing.Gate{}, nil)

	handler := handlers.GetAllGates(service)

	req := httptest.NewRequest("GET", "/gates", nil)
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&response)
	data := response["data"].([]interface{})
	assert.Len(t, data, 0)
}

func TestGetAllGates_database_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockSvc)

	mockSvc.EXPECT().GetAll(gomock.Any()).Return(nil, errors.New("database error"))

	handler := handlers.GetAllGates(service)

	req := httptest.NewRequest("GET", "/gates", nil)
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
}
