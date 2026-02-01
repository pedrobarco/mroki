package diffing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/pedrobarco/mroki/internal/domain/diffing/mocks"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGateService_Create_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockGateRepository(ctrl)
	mockRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	service := diffing.NewGateService(mockRepo)
	gate, err := service.Create(context.Background(), "http://live.example.com", "http://shadow.example.com")

	assert.NoError(t, err)
	assert.NotNil(t, gate)
	assert.False(t, gate.ID.IsZero())
}

func TestGateService_Create_invalid_live_url(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockRepo)

	_, err := service.Create(context.Background(), "not-a-url", "http://shadow.example.com")

	assert.Error(t, err)
}

func TestGateService_Create_repo_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedErr := errors.New("database error")
	mockRepo := mocks.NewMockGateRepository(ctrl)
	mockRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(expectedErr)

	service := diffing.NewGateService(mockRepo)
	_, err := service.Create(context.Background(), "http://live.example.com", "http://shadow.example.com")

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestGateService_GetByID_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	gateID := diffing.NewGateID()
	expectedGate := &diffing.Gate{ID: gateID}

	mockRepo := mocks.NewMockGateRepository(ctrl)
	mockRepo.EXPECT().GetByID(gomock.Any(), gateID).Return(expectedGate, nil)

	service := diffing.NewGateService(mockRepo)
	gate, err := service.GetByID(context.Background(), gateID.String())

	assert.NoError(t, err)
	assert.Equal(t, expectedGate, gate)
}

func TestGateService_GetByID_invalid_id(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockGateRepository(ctrl)
	service := diffing.NewGateService(mockRepo)

	_, err := service.GetByID(context.Background(), "invalid-uuid")

	assert.Error(t, err)
}

func TestGateService_GetAll_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedGates := []*diffing.Gate{
		{ID: diffing.NewGateID()},
		{ID: diffing.NewGateID()},
	}

	// Service will create pagination.Params from limit=50, offset=0
	params, _ := pagination.NewParams(50, 0)
	pagedResult := pagination.NewPagedResult(expectedGates, int64(2), params)

	mockRepo := mocks.NewMockGateRepository(ctrl)
	mockRepo.EXPECT().GetAll(gomock.Any(), gomock.Any()).Return(pagedResult, nil)

	service := diffing.NewGateService(mockRepo)
	result, err := service.GetAll(context.Background(), 50, 0)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.Total)
	assert.Equal(t, expectedGates, result.Items)
}

func TestGateService_GetAll_repo_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedErr := errors.New("database error")

	mockRepo := mocks.NewMockGateRepository(ctrl)
	mockRepo.EXPECT().GetAll(gomock.Any(), gomock.Any()).Return(nil, expectedErr)

	service := diffing.NewGateService(mockRepo)
	_, err := service.GetAll(context.Background(), 50, 0)

	assert.Error(t, err)
}

func TestGateService_GetAll_with_defaults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedGates := []*diffing.Gate{
		{ID: diffing.NewGateID()},
	}

	// Service will apply defaults: limit=0 -> 50, offset=-10 -> 0
	params, _ := pagination.NewParams(0, -10)
	pagedResult := pagination.NewPagedResult(expectedGates, int64(1), params)

	mockRepo := mocks.NewMockGateRepository(ctrl)
	mockRepo.EXPECT().GetAll(gomock.Any(), gomock.Any()).Return(pagedResult, nil)

	service := diffing.NewGateService(mockRepo)
	result, err := service.GetAll(context.Background(), 0, -10)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 50, result.Limit) // default applied
	assert.Equal(t, 0, result.Offset) // negative corrected to 0
}
