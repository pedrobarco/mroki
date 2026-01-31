package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/pedrobarco/mroki/internal/domain/diffing/mocks"
	"github.com/pedrobarco/mroki/internal/handlers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCreateRequest_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockSvc)

	mockSvc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	handler := handlers.CreateRequest(service)

	body := `{
		"method": "POST",
		"path": "/api/test",
		"headers": {"Content-Type": ["application/json"]},
		"body": "{\"test\":\"data\"}",
		"created_at": "2024-01-01T00:00:00Z",
		"responses": [
			{
				"type": "live",
				"status_code": 200,
				"headers": {},
				"body": "{\"status\":\"ok\"}",
				"created_at": "2024-01-01T00:00:00Z"
			},
			{
				"type": "shadow",
				"status_code": 200,
				"headers": {},
				"body": "{\"status\":\"ok\"}",
				"created_at": "2024-01-01T00:00:00Z"
			}
		],
		"diff": {
			"content": "no differences"
		}
	}`

	gateID := diffing.NewGateID()
	req := httptest.NewRequest("POST", "/gates/"+gateID.String()+"/requests", bytes.NewBufferString(body))
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestCreateRequest_invalid_json(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockSvc)
	handler := handlers.CreateRequest(service)

	gateID := diffing.NewGateID()
	req := httptest.NewRequest("POST", "/gates/"+gateID.String()+"/requests", bytes.NewBufferString("{invalid"))
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestCreateRequest_missing_gate_id(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockSvc)
	handler := handlers.CreateRequest(service)

	body := `{"method":"GET","path":"/test"}`
	req := httptest.NewRequest("POST", "/gates//requests", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Contains(t, apiErr.Message, "gate_id")
}

func TestCreateRequest_invalid_gate_id(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockSvc)
	handler := handlers.CreateRequest(service)

	body := `{
		"method": "GET",
		"path": "/test",
		"headers": {},
		"body": "",
		"created_at": "2024-01-01T00:00:00Z",
		"responses": [
			{"type":"live","status_code":200,"headers":{},"body":"","created_at":"2024-01-01T00:00:00Z"},
			{"type":"shadow","status_code":200,"headers":{},"body":"","created_at":"2024-01-01T00:00:00Z"}
		],
		"diff": {"content":""}
	}`
	req := httptest.NewRequest("POST", "/gates/invalid/requests", bytes.NewBufferString(body))
	req.SetPathValue("gate_id", "invalid")
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestGetRequestByID_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockSvc)

	gateID := diffing.NewGateID()
	requestID := diffing.NewRequestID()
	request := &diffing.Request{
		ID:        requestID,
		GateID:    gateID,
		Method:    "GET",
		Path:      "/test",
		Headers:   http.Header{},
		Body:      []byte{},
		CreatedAt: time.Now(),
		Responses: []diffing.Response{},
		Diff:      diffing.Diff{},
	}

	mockSvc.EXPECT().GetByID(gomock.Any(), requestID, gateID).Return(request, nil)

	handler := handlers.GetRequestByID(service)

	req := httptest.NewRequest("GET", "/gates/"+gateID.String()+"/requests/"+requestID.String(), nil)
	req.SetPathValue("gate_id", gateID.String())
	req.SetPathValue("request_id", requestID.String())
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetRequestByID_missing_gate_id(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockSvc)
	handler := handlers.GetRequestByID(service)

	requestID := diffing.NewRequestID()
	req := httptest.NewRequest("GET", "/gates//requests/"+requestID.String(), nil)
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestGetRequestByID_missing_request_id(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockSvc)
	handler := handlers.GetRequestByID(service)

	gateID := diffing.NewGateID()
	req := httptest.NewRequest("GET", "/gates/"+gateID.String()+"/requests/", nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestGetRequestByID_not_found(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockSvc)

	gateID := diffing.NewGateID()
	requestID := diffing.NewRequestID()
	mockSvc.EXPECT().GetByID(gomock.Any(), requestID, gateID).Return(nil, diffing.ErrRequestNotFound)

	handler := handlers.GetRequestByID(service)

	req := httptest.NewRequest("GET", "/gates/"+gateID.String()+"/requests/"+requestID.String(), nil)
	req.SetPathValue("gate_id", gateID.String())
	req.SetPathValue("request_id", requestID.String())
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestGetAllRequestsByGateID_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockSvc)

	gateID := diffing.NewGateID()
	requests := []*diffing.Request{
		{
			ID:        diffing.NewRequestID(),
			GateID:    gateID,
			Method:    "GET",
			Path:      "/test1",
			CreatedAt: time.Now(),
		},
		{
			ID:        diffing.NewRequestID(),
			GateID:    gateID,
			Method:    "POST",
			Path:      "/test2",
			CreatedAt: time.Now(),
		},
	}

	mockSvc.EXPECT().GetAllByGateID(gomock.Any(), gateID).Return(requests, nil)

	handler := handlers.GetAllRequestsByGateID(service)

	req := httptest.NewRequest("GET", "/gates/"+gateID.String()+"/requests", nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&response)
	data := response["data"].([]interface{})
	assert.Len(t, data, 2)
}

func TestGetAllRequestsByGateID_empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockSvc)

	gateID := diffing.NewGateID()
	mockSvc.EXPECT().GetAllByGateID(gomock.Any(), gateID).Return([]*diffing.Request{}, nil)

	handler := handlers.GetAllRequestsByGateID(service)

	req := httptest.NewRequest("GET", "/gates/"+gateID.String()+"/requests", nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetAllRequestsByGateID_invalid_gate_id(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockSvc)
	handler := handlers.GetAllRequestsByGateID(service)

	req := httptest.NewRequest("GET", "/gates/invalid/requests", nil)
	req.SetPathValue("gate_id", "invalid")
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestGetAllRequestsByGateID_database_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockSvc)

	gateID := diffing.NewGateID()
	mockSvc.EXPECT().GetAllByGateID(gomock.Any(), gateID).Return(nil, errors.New("database error"))

	handler := handlers.GetAllRequestsByGateID(service)

	req := httptest.NewRequest("GET", "/gates/"+gateID.String()+"/requests", nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	err := handler(rec, req)

	assert.Error(t, err)
	apiErr, ok := err.(*handlers.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
}
