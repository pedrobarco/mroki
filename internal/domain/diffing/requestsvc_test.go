package diffing_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/pedrobarco/mroki/internal/domain/diffing/mocks"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRequestService_Create_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRequestRepository(ctrl)
	mockRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	service := diffing.NewRequestService(mockRepo)
	gateID := diffing.NewGateID()

	props := diffing.CreateRequestProps{
		GateID:    gateID.String(),
		Method:    "POST",
		Path:      "/api/test",
		Headers:   http.Header{"Content-Type": []string{"application/json"}},
		Body:      []byte(`{"test":"data"}`),
		CreatedAt: time.Now(),
		Responses: []diffing.CreateRequestResponseProps{
			{
				Type:       "live",
				StatusCode: 200,
				Headers:    http.Header{},
				Body:       []byte(`{"status":"ok"}`),
				CreatedAt:  time.Now(),
			},
			{
				Type:       "shadow",
				StatusCode: 200,
				Headers:    http.Header{},
				Body:       []byte(`{"status":"ok"}`),
				CreatedAt:  time.Now(),
			},
		},
		Diff: diffing.CreateRequestDiffProps{
			Content: "no differences",
		},
	}

	request, err := service.Create(context.Background(), props)

	assert.NoError(t, err)
	assert.NotNil(t, request)
	assert.False(t, request.ID.IsZero())
	assert.Equal(t, gateID.String(), request.GateID.String())
}

func TestRequestService_Create_invalid_gate_id(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockRepo)

	props := diffing.CreateRequestProps{
		GateID: "invalid-uuid",
		Method: "POST",
		Path:   "/api/test",
	}

	_, err := service.Create(context.Background(), props)

	assert.Error(t, err)
}

func TestRequestService_Create_invalid_response_type(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockRepo)
	gateID := diffing.NewGateID()

	props := diffing.CreateRequestProps{
		GateID:    gateID.String(),
		Method:    "POST",
		Path:      "/api/test",
		CreatedAt: time.Now(),
		Responses: []diffing.CreateRequestResponseProps{
			{
				Type:       "invalid-type",
				StatusCode: 200,
				CreatedAt:  time.Now(),
			},
		},
	}

	_, err := service.Create(context.Background(), props)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid response type")
}

func TestRequestService_Create_missing_live_response(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockRepo)
	gateID := diffing.NewGateID()

	props := diffing.CreateRequestProps{
		GateID:    gateID.String(),
		Method:    "POST",
		Path:      "/api/test",
		CreatedAt: time.Now(),
		Responses: []diffing.CreateRequestResponseProps{
			{
				Type:       "shadow",
				StatusCode: 200,
				CreatedAt:  time.Now(),
			},
			{
				Type:       "shadow",
				StatusCode: 200,
				CreatedAt:  time.Now(),
			},
		},
		Diff: diffing.CreateRequestDiffProps{Content: "diff"},
	}

	_, err := service.Create(context.Background(), props)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "live response is required")
}

func TestRequestService_Create_repo_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedErr := errors.New("database error")
	mockRepo := mocks.NewMockRequestRepository(ctrl)
	mockRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(expectedErr)

	service := diffing.NewRequestService(mockRepo)
	gateID := diffing.NewGateID()

	props := diffing.CreateRequestProps{
		GateID:    gateID.String(),
		Method:    "POST",
		Path:      "/api/test",
		CreatedAt: time.Now(),
		Responses: []diffing.CreateRequestResponseProps{
			{
				Type:       "live",
				StatusCode: 200,
				CreatedAt:  time.Now(),
			},
			{
				Type:       "shadow",
				StatusCode: 200,
				CreatedAt:  time.Now(),
			},
		},
		Diff: diffing.CreateRequestDiffProps{Content: "diff"},
	}

	_, err := service.Create(context.Background(), props)

	assert.Error(t, err)
}

func TestRequestService_GetByID_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	requestID := diffing.NewRequestID()
	gateID := diffing.NewGateID()
	expectedRequest := &diffing.Request{
		ID:     requestID,
		GateID: gateID,
	}

	mockRepo := mocks.NewMockRequestRepository(ctrl)
	mockRepo.EXPECT().GetByID(gomock.Any(), requestID, gateID).Return(expectedRequest, nil)

	service := diffing.NewRequestService(mockRepo)
	request, err := service.GetByID(context.Background(), requestID.String(), gateID.String())

	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, request)
}

func TestRequestService_GetByID_invalid_request_id(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockRepo)
	gateID := diffing.NewGateID()

	_, err := service.GetByID(context.Background(), "invalid-uuid", gateID.String())

	assert.Error(t, err)
}

func TestRequestService_GetAllByGateID_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	gateID := diffing.NewGateID()
	expectedRequests := []*diffing.Request{
		{ID: diffing.NewRequestID(), GateID: gateID},
		{ID: diffing.NewRequestID(), GateID: gateID},
	}

	params, _ := pagination.NewParams(50, 0)
	pagedResult := pagination.NewPagedResult(expectedRequests, int64(2), params)

	mockRepo := mocks.NewMockRequestRepository(ctrl)
	mockRepo.EXPECT().GetAllByGateID(gomock.Any(), gateID, gomock.Any()).Return(pagedResult, nil)

	service := diffing.NewRequestService(mockRepo)
	result, err := service.GetAllByGateID(context.Background(), gateID.String(), 50, 0)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.Total)
	assert.False(t, result.HasMore)
}

func TestRequestService_GetAllByGateID_with_defaults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	gateID := diffing.NewGateID()
	expectedRequests := []*diffing.Request{
		{ID: diffing.NewRequestID(), GateID: gateID},
	}

	params, _ := pagination.NewParams(50, 0)
	pagedResult := pagination.NewPagedResult(expectedRequests, int64(1), params)

	mockRepo := mocks.NewMockRequestRepository(ctrl)
	mockRepo.EXPECT().GetAllByGateID(gomock.Any(), gateID, gomock.Any()).Return(pagedResult, nil)

	service := diffing.NewRequestService(mockRepo)

	// Pass 0 values - should use defaults
	result, err := service.GetAllByGateID(context.Background(), gateID.String(), 0, 0)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 50, result.Limit) // Default applied
	assert.Equal(t, 0, result.Offset)
}

func TestRequestService_GetAllByGateID_invalid_gate_id(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRequestRepository(ctrl)
	service := diffing.NewRequestService(mockRepo)

	_, err := service.GetAllByGateID(context.Background(), "invalid-uuid", 50, 0)

	assert.Error(t, err)
}
