package diffing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/pedrobarco/mroki/internal/domain/diffing/mocks"
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

	mockRepo := mocks.NewMockGateRepository(ctrl)
	mockRepo.EXPECT().GetAll(gomock.Any()).Return(expectedGates, nil)

	service := diffing.NewGateService(mockRepo)
	gates, err := service.GetAll(context.Background())

	assert.NoError(t, err)
	assert.Len(t, gates, 2)
	assert.Equal(t, expectedGates, gates)
}

func TestGateService_GetAll_repo_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedErr := errors.New("database error")
	mockRepo := mocks.NewMockGateRepository(ctrl)
	mockRepo.EXPECT().GetAll(gomock.Any()).Return(nil, expectedErr)

	service := diffing.NewGateService(mockRepo)
	_, err := service.GetAll(context.Background())

	assert.Error(t, err)
}
