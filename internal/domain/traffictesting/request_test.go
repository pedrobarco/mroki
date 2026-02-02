package traffictesting_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
)

func TestNewRequest_creates_request_with_auto_generated_id(t *testing.T) {
	gateID := traffictesting.NewGateID()
	method, _ := traffictesting.NewHTTPMethod("POST")
	path, _ := traffictesting.ParsePath("/api/test")
	headers := traffictesting.NewHeaders(http.Header{"Content-Type": []string{"application/json"}})
	body := []byte(`{"test":"data"}`)
	createdAt := time.Now()
	responses := []traffictesting.Response{}
	diff := traffictesting.Diff{}

	request, err := traffictesting.NewRequest(gateID, method, path, headers, body, createdAt, responses, diff)

	assert.NoError(t, err)
	assert.False(t, request.ID.IsZero())
	assert.Equal(t, gateID.String(), request.GateID.String())
	assert.Equal(t, "POST", request.Method.String())
	assert.Equal(t, "/api/test", request.Path.String())
	assert.Equal(t, http.Header{"Content-Type": []string{"application/json"}}, request.Headers.HTTPHeader())
	assert.Equal(t, body, request.Body)
	assert.Equal(t, createdAt, request.CreatedAt)
}

func TestNewRequest_with_custom_id(t *testing.T) {
	gateID := traffictesting.NewGateID()
	customID := traffictesting.NewRequestID()
	method, _ := traffictesting.NewHTTPMethod("GET")
	path, _ := traffictesting.ParsePath("/api/health")
	createdAt := time.Now()

	request, err := traffictesting.NewRequest(
		gateID,
		method,
		path,
		traffictesting.NewHeaders(nil),
		nil,
		createdAt,
		[]traffictesting.Response{},
		traffictesting.Diff{},
		traffictesting.WithRequestID(customID),
	)

	assert.NoError(t, err)
	assert.Equal(t, customID.String(), request.ID.String())
}
