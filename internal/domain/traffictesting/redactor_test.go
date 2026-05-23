package traffictesting_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactor_HeaderOnly(t *testing.T) {
	r := traffictesting.NewRedactor([]string{"headers.Authorization", "headers.Cookie"})
	h := http.Header{"Authorization": {"Bearer secret"}, "Content-Type": {"application/json"}}

	res, err := r.Redact(h, nil)

	require.NoError(t, err)
	assert.Equal(t, "[REDACTED]", res.Headers.Get("Authorization"))
	assert.Equal(t, "", res.Headers.Get("Cookie"), "missing header should not be redacted")
	assert.Equal(t, "application/json", res.Headers.Get("Content-Type"))
	assert.Nil(t, res.Body)
}

func TestRedactor_HeaderOnly_missing_header_not_redacted(t *testing.T) {
	r := traffictesting.NewRedactor([]string{"headers.X-Missing"})
	h := http.Header{"Content-Type": {"text/plain"}}

	res, err := r.Redact(h, nil)

	require.NoError(t, err)
	assert.Equal(t, "", res.Headers.Get("X-Missing"))
	assert.Equal(t, "text/plain", res.Headers.Get("Content-Type"))
}

func TestRedactor_BodyOnly(t *testing.T) {
	r := traffictesting.NewRedactor([]string{"body.password"})
	body := []byte(`{"username":"alice","password":"s3cret"}`)

	res, err := r.Redact(nil, body)

	require.NoError(t, err)
	assert.JSONEq(t, `{"username":"alice","password":"[REDACTED]"}`, string(res.Body))
}

func TestRedactor_Mixed(t *testing.T) {
	r := traffictesting.NewRedactor([]string{"headers.Authorization", "body.token"})
	h := http.Header{"Authorization": {"Bearer x"}}
	body := []byte(`{"token":"abc","data":"ok"}`)

	res, err := r.Redact(h, body)

	require.NoError(t, err)
	assert.Equal(t, "[REDACTED]", res.Headers.Get("Authorization"))
	assert.JSONEq(t, `{"token":"[REDACTED]","data":"ok"}`, string(res.Body))
}

func TestRedactor_NestedBodyPath(t *testing.T) {
	r := traffictesting.NewRedactor([]string{"body.user.address.street"})
	body := []byte(`{"user":{"address":{"street":"123 Main","city":"NY"}}}`)

	res, err := r.Redact(nil, body)

	require.NoError(t, err)
	assert.JSONEq(t, `{"user":{"address":{"street":"[REDACTED]","city":"NY"}}}`, string(res.Body))
}

func TestRedactor_ArrayPath(t *testing.T) {
	r := traffictesting.NewRedactor([]string{"body.users.#.ssn"})
	body := []byte(`{"users":[{"name":"Alice","ssn":"111"},{"name":"Bob","ssn":"222"}]}`)

	res, err := r.Redact(nil, body)

	require.NoError(t, err)
	assert.JSONEq(t, `{"users":[{"name":"Alice","ssn":"[REDACTED]"},{"name":"Bob","ssn":"[REDACTED]"}]}`, string(res.Body))
}

func TestRedactor_MissingFields_silently_skipped(t *testing.T) {
	r := traffictesting.NewRedactor([]string{"body.nonexistent", "headers.X-Gone"})
	h := http.Header{"Content-Type": {"text/plain"}}
	body := []byte(`{"kept":"yes"}`)

	res, err := r.Redact(h, body)

	require.NoError(t, err)
	assert.Equal(t, "text/plain", res.Headers.Get("Content-Type"))
	assert.JSONEq(t, `{"kept":"yes"}`, string(res.Body))
}

func TestRedactor_NilBody(t *testing.T) {
	r := traffictesting.NewRedactor([]string{"body.password"})

	res, err := r.Redact(nil, nil)

	require.NoError(t, err)
	assert.Nil(t, res.Body)
}

func TestRedactor_EmptyBody(t *testing.T) {
	r := traffictesting.NewRedactor([]string{"body.password"})

	res, err := r.Redact(nil, []byte{})

	require.NoError(t, err)
	assert.Empty(t, res.Body)
}

func TestRedactor_NonJSON_body_passthrough(t *testing.T) {
	r := traffictesting.NewRedactor([]string{"body.password"})
	body := []byte("this is not json")

	res, err := r.Redact(nil, body)

	require.NoError(t, err)
	assert.Equal(t, "this is not json", string(res.Body))
}

func TestRedactor_DoesNotMutateOriginalHeaders(t *testing.T) {
	r := traffictesting.NewRedactor([]string{"headers.Authorization"})
	h := http.Header{"Authorization": {"Bearer secret"}}

	_, _ = r.Redact(h, nil)

	assert.Equal(t, "Bearer secret", h.Get("Authorization"), "original headers must not be mutated")
}

func TestRedactor_NoFields(t *testing.T) {
	r := traffictesting.NewRedactor(nil)
	h := http.Header{"Authorization": {"Bearer secret"}}
	body := []byte(`{"password":"s3cret"}`)

	res, err := r.Redact(h, body)

	require.NoError(t, err)
	assert.Equal(t, "Bearer secret", res.Headers.Get("Authorization"))
	assert.JSONEq(t, `{"password":"s3cret"}`, string(res.Body))
}

func TestRedactor_NonCanonicalHeaderKeys(t *testing.T) {
	r := traffictesting.NewRedactor([]string{"headers.Authorization", "headers.X-Api-Key"})

	// Use variables for non-canonical keys to prevent the canonicalheader
	// linter from auto-fixing them to canonical form — that would defeat
	// the purpose of this test.
	authKey := "authorization"
	apiKeyKey := "x-api-key"

	h := http.Header{}
	h[authKey] = []string{"Bearer secret"}
	h[apiKeyKey] = []string{"key123"}
	h["Content-Type"] = []string{"application/json"}

	res, err := r.Redact(h, nil)

	require.NoError(t, err)
	// Clone() preserves the original (non-canonical) keys, so we must assert
	// using the same casing that was in the input map.
	assert.Equal(t, []string{traffictesting.RedactedValue}, res.Headers[authKey])
	assert.Equal(t, []string{traffictesting.RedactedValue}, res.Headers[apiKeyKey])
	assert.Equal(t, []string{"application/json"}, res.Headers["Content-Type"])
}

func BenchmarkRedactor_10MB_20fields(b *testing.B) {
	fields := make([]string, 20)
	for i := range fields {
		fields[i] = fmt.Sprintf("body.field_%d", i)
	}

	obj := make(map[string]interface{})
	for _, f := range fields {
		key := f[len("body."):]
		obj[key] = "sensitive-value-that-should-be-redacted"
	}
	filler := make([]byte, 10*1024*1024)
	for i := range filler {
		filler[i] = byte('A' + rand.Intn(26))
	}
	obj["filler"] = string(filler)

	body, _ := json.Marshal(obj)
	r := traffictesting.NewRedactor(fields)

	b.ResetTimer()
	b.SetBytes(int64(len(body)))
	for i := 0; i < b.N; i++ {
		_, _ = r.Redact(nil, body)
	}
}
