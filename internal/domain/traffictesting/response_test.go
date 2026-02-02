package traffictesting_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
)

func TestNewResponseType_valid_live(t *testing.T) {
	responseType, err := traffictesting.NewResponseType("live")

	assert.NoError(t, err)
	assert.Equal(t, traffictesting.ResponseTypeLive, responseType)
}

func TestNewResponseType_valid_shadow(t *testing.T) {
	responseType, err := traffictesting.NewResponseType("shadow")

	assert.NoError(t, err)
	assert.Equal(t, traffictesting.ResponseTypeShadow, responseType)
}

func TestNewResponseType_invalid(t *testing.T) {
	_, err := traffictesting.NewResponseType("invalid")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid response type")
}

func TestNewResponse_creates_response_with_auto_generated_id(t *testing.T) {
	headers := traffictesting.NewHeaders(http.Header{"Content-Type": []string{"application/json"}})
	body := []byte(`{"status":"ok"}`)
	createdAt := time.Now()
	statusCode, _ := traffictesting.ParseStatusCode(200)

	response, err := traffictesting.NewResponse(traffictesting.ResponseTypeLive, statusCode, headers, body, createdAt)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, response.ID)
	assert.Equal(t, traffictesting.ResponseTypeLive, response.Type)
	assert.Equal(t, 200, response.StatusCode.Int())
	assert.Equal(t, http.Header{"Content-Type": []string{"application/json"}}, response.Headers.HTTPHeader())
	assert.Equal(t, body, response.Body)
	assert.Equal(t, createdAt, response.CreatedAt)
}

func TestNewResponse_with_custom_id(t *testing.T) {
	customID := uuid.New()
	createdAt := time.Now()
	statusCode, _ := traffictesting.ParseStatusCode(500)

	response, err := traffictesting.NewResponse(
		traffictesting.ResponseTypeShadow,
		statusCode,
		traffictesting.NewHeaders(nil),
		nil,
		createdAt,
		traffictesting.WithResponseID(customID),
	)

	assert.NoError(t, err)
	assert.Equal(t, customID, response.ID)
	assert.Equal(t, traffictesting.ResponseTypeShadow, response.Type)
}
