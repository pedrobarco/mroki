package diffing_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/stretchr/testify/assert"
)

func TestNewRequest_creates_request_with_auto_generated_id(t *testing.T) {
	gateID := diffing.NewGateID()
	headers := http.Header{"Content-Type": []string{"application/json"}}
	body := []byte(`{"test":"data"}`)
	createdAt := time.Now()
	responses := []diffing.Response{}
	diff := diffing.Diff{}

	request, err := diffing.NewRequest(gateID, "POST", "/api/test", headers, body, createdAt, responses, diff)

	assert.NoError(t, err)
	assert.False(t, request.ID.IsZero())
	assert.Equal(t, gateID.String(), request.GateID.String())
	assert.Equal(t, "POST", request.Method)
	assert.Equal(t, "/api/test", request.Path)
	assert.Equal(t, headers, request.Headers)
	assert.Equal(t, body, request.Body)
	assert.Equal(t, createdAt, request.CreatedAt)
}

func TestNewRequest_with_custom_id(t *testing.T) {
	gateID := diffing.NewGateID()
	customID := diffing.NewRequestID()
	createdAt := time.Now()

	request, err := diffing.NewRequest(
		gateID,
		"GET",
		"/api/health",
		nil,
		nil,
		createdAt,
		[]diffing.Response{},
		diffing.Diff{},
		diffing.WithRequestID(customID),
	)

	assert.NoError(t, err)
	assert.Equal(t, customID.String(), request.ID.String())
}
