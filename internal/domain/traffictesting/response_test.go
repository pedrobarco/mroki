package traffictesting_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
)

func TestNewResponse_creates_response_with_auto_generated_id(t *testing.T) {
	headers := traffictesting.NewHeaders(http.Header{"Content-Type": []string{"application/json"}})
	body := []byte(`{"status":"ok"}`)
	createdAt := time.Now()
	statusCode, _ := traffictesting.ParseStatusCode(200)

	response, err := traffictesting.NewResponse(statusCode, headers, body, int64(142), createdAt)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, response.ID)
	assert.Equal(t, 200, response.StatusCode.Int())
	assert.Equal(t, http.Header{"Content-Type": []string{"application/json"}}, response.Headers.HTTPHeader())
	assert.Equal(t, body, response.Body)
	assert.Equal(t, int64(142), response.LatencyMs)
	assert.Equal(t, createdAt, response.CreatedAt)
}

func TestNewResponse_zero_latency(t *testing.T) {
	statusCode, _ := traffictesting.ParseStatusCode(204)

	response, err := traffictesting.NewResponse(
		statusCode,
		traffictesting.NewHeaders(nil),
		nil,
		int64(0),
		time.Now(),
	)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), response.LatencyMs)
}

func TestNewResponse_with_custom_id(t *testing.T) {
	customID := uuid.New()
	createdAt := time.Now()
	statusCode, _ := traffictesting.ParseStatusCode(500)

	response, err := traffictesting.NewResponse(
		statusCode,
		traffictesting.NewHeaders(nil),
		nil,
		int64(0),
		createdAt,
		traffictesting.WithResponseID(customID),
	)

	assert.NoError(t, err)
	assert.Equal(t, customID, response.ID)
}
