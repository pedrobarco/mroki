package diffing_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/stretchr/testify/assert"
)

func TestNewResponseType_valid_live(t *testing.T) {
	responseType, err := diffing.NewResponseType("live")

	assert.NoError(t, err)
	assert.Equal(t, diffing.ResponseTypeLive, responseType)
}

func TestNewResponseType_valid_shadow(t *testing.T) {
	responseType, err := diffing.NewResponseType("shadow")

	assert.NoError(t, err)
	assert.Equal(t, diffing.ResponseTypeShadow, responseType)
}

func TestNewResponseType_invalid(t *testing.T) {
	_, err := diffing.NewResponseType("invalid")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid response type")
}

func TestNewResponse_creates_response_with_auto_generated_id(t *testing.T) {
	headers := http.Header{"Content-Type": []string{"application/json"}}
	body := []byte(`{"status":"ok"}`)
	createdAt := time.Now()

	response, err := diffing.NewResponse(diffing.ResponseTypeLive, 200, headers, body, createdAt)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, response.ID)
	assert.Equal(t, diffing.ResponseTypeLive, response.Type)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, headers, response.Headers)
	assert.Equal(t, body, response.Body)
	assert.Equal(t, createdAt, response.CreatedAt)
}

func TestNewResponse_with_custom_id(t *testing.T) {
	customID := uuid.New()
	createdAt := time.Now()

	response, err := diffing.NewResponse(
		diffing.ResponseTypeShadow,
		500,
		nil,
		nil,
		createdAt,
		diffing.WithResponseID(customID),
	)

	assert.NoError(t, err)
	assert.Equal(t, customID, response.ID)
	assert.Equal(t, diffing.ResponseTypeShadow, response.Type)
}
