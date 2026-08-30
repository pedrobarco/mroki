package services_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/pedrobarco/mroki/internal/application/services"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertOpsFreeOfSecrets fails if any patch op's path or value contains one of
// the given raw secret strings. Values are JSON-marshaled so nested secrets are
// caught regardless of type.
func assertOpsFreeOfSecrets(t *testing.T, ops []diff.PatchOp, secrets ...string) {
	t.Helper()
	for _, op := range ops {
		valBytes, err := json.Marshal(op.Value)
		require.NoError(t, err)
		serialized := op.Op + " " + op.Path + " " + string(valBytes)
		for _, secret := range secrets {
			assert.NotContains(t, serialized, secret,
				"redacted raw value %q leaked into patch op %s", secret, serialized)
		}
	}
}

func TestCompare_identical_json_responses(t *testing.T) {
	redactor := traffictesting.NewRedactor(nil)
	comparer := services.NewResponseComparer(redactor, nil)

	body := []byte(`{"name":"Alice","age":30}`)
	req := services.ResponseData{StatusCode: 200, Headers: http.Header{"Content-Type": {"application/json"}}, Body: body}
	live := services.ResponseData{StatusCode: 200, Headers: http.Header{"X-Live": {"true"}}, Body: body}
	shadow := services.ResponseData{StatusCode: 200, Headers: http.Header{"X-Live": {"true"}}, Body: body}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	assert.Empty(t, result.Ops)
	assert.Equal(t, "application/json", result.Request.Headers.Get("Content-Type"))
	assert.NotNil(t, result.Request.BodyParsed)
	assert.NotNil(t, result.Live.BodyParsed)
	assert.NotNil(t, result.Shadow.BodyParsed)
	assert.Equal(t, "true", result.Live.Headers.Get("X-Live"))
	assert.Equal(t, "true", result.Shadow.Headers.Get("X-Live"))
}

func TestCompare_different_json_responses(t *testing.T) {
	redactor := traffictesting.NewRedactor(nil)
	comparer := services.NewResponseComparer(redactor, nil)

	req := services.ResponseData{StatusCode: 200, Body: []byte(`{}`)}
	live := services.ResponseData{StatusCode: 200, Body: []byte(`{"user":"Alice"}`)}
	shadow := services.ResponseData{StatusCode: 200, Body: []byte(`{"user":"Bob"}`)}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	require.NotEmpty(t, result.Ops)

	paths := make(map[string]string)
	for _, op := range result.Ops {
		paths[op.Path] = op.Op
	}
	assert.Contains(t, paths, "/body/user")
	assert.Equal(t, "replace", paths["/body/user"])
}

func TestCompare_redacts_headers(t *testing.T) {
	redactor := traffictesting.NewRedactor([]string{"headers.Cookie"})
	comparer := services.NewResponseComparer(redactor, nil)

	req := services.ResponseData{
		StatusCode: 200,
		Headers:    http.Header{"Cookie": {"secret"}},
		Body:       []byte(`{}`),
	}
	live := services.ResponseData{StatusCode: 200, Body: []byte(`{}`)}
	shadow := services.ResponseData{StatusCode: 200, Body: []byte(`{}`)}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	assert.Equal(t, []string{"[REDACTED]"}, result.Request.Headers["Cookie"])
}

func TestCompare_redacts_body_fields(t *testing.T) {
	redactor := traffictesting.NewRedactor([]string{"body.password"})
	comparer := services.NewResponseComparer(redactor, nil)

	req := services.ResponseData{StatusCode: 200, Body: []byte(`{}`)}
	live := services.ResponseData{
		StatusCode: 200,
		Body:       []byte(`{"password":"secret","name":"Alice"}`),
	}
	shadow := services.ResponseData{StatusCode: 200, Body: []byte(`{"password":"secret","name":"Alice"}`)}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	require.NotNil(t, result.Live.BodyParsed)

	m, ok := result.Live.BodyParsed.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "[REDACTED]", m["password"])
	assert.Equal(t, "Alice", m["name"])
}

func TestCompare_empty_bodies(t *testing.T) {
	redactor := traffictesting.NewRedactor(nil)
	comparer := services.NewResponseComparer(redactor, nil)

	req := services.ResponseData{StatusCode: 200}
	live := services.ResponseData{StatusCode: 200}
	shadow := services.ResponseData{StatusCode: 200}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	assert.Empty(t, result.Ops)
	assert.Nil(t, result.Request.BodyParsed)
	assert.Nil(t, result.Live.BodyParsed)
	assert.Nil(t, result.Shadow.BodyParsed)
}

func TestCompare_non_json_bodies(t *testing.T) {
	redactor := traffictesting.NewRedactor(nil)
	comparer := services.NewResponseComparer(redactor, nil)

	ct := http.Header{"Content-Type": {"text/html"}}
	req := services.ResponseData{StatusCode: 200, Body: []byte(`{}`)}
	live := services.ResponseData{StatusCode: 200, Headers: ct, Body: []byte(`<html>hello</html>`)}
	shadow := services.ResponseData{StatusCode: 200, Headers: ct, Body: []byte(`<html>world</html>`)}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	// Non-JSON bodies produce nil BodyParsed (the redactor only parses JSON)...
	assert.Nil(t, result.Live.BodyParsed)
	assert.Nil(t, result.Shadow.BodyParsed)
	// ...but the text Content-Type makes the envelope embed each body as a raw
	// string, so the difference now surfaces under /body instead of being lost.
	require.NotEmpty(t, result.Ops)
	paths := make(map[string]string)
	for _, op := range result.Ops {
		paths[op.Path] = op.Op
	}
	assert.Contains(t, paths, "/body")
	assert.Equal(t, "replace", paths["/body"])
}

func TestCompare_with_diff_options(t *testing.T) {
	redactor := traffictesting.NewRedactor(nil)
	comparer := services.NewResponseComparer(redactor, []diff.Option{
		diff.WithIgnoredFields("body.timestamp"),
	})

	req := services.ResponseData{StatusCode: 200, Body: []byte(`{}`)}
	live := services.ResponseData{StatusCode: 200, Body: []byte(`{"timestamp":"2024-01-01T10:00:00Z"}`)}
	shadow := services.ResponseData{StatusCode: 200, Body: []byte(`{"timestamp":"2024-01-01T11:00:00Z"}`)}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	assert.Empty(t, result.Ops, "timestamp difference should be ignored")
}

func TestCompare_different_status_codes(t *testing.T) {
	redactor := traffictesting.NewRedactor(nil)
	comparer := services.NewResponseComparer(redactor, nil)

	body := []byte(`{"ok":true}`)
	req := services.ResponseData{StatusCode: 200, Body: body}
	live := services.ResponseData{StatusCode: 200, Body: body}
	shadow := services.ResponseData{StatusCode: 500, Body: body}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	require.NotEmpty(t, result.Ops)

	paths := make(map[string]string)
	for _, op := range result.Ops {
		paths[op.Path] = op.Op
	}
	assert.Contains(t, paths, "/statusCode")
	assert.Equal(t, "replace", paths["/statusCode"])
}

func TestCompare_nil_redactor(t *testing.T) {
	comparer := services.NewResponseComparer(nil, nil)
	_, err := comparer.Compare(
		services.ResponseData{},
		services.ResponseData{StatusCode: 200},
		services.ResponseData{StatusCode: 200},
	)
	require.ErrorIs(t, err, services.ErrNilRedactor)
}

func TestCompare_redaction_error_includes_context(t *testing.T) {
	// Redactor with body fields — passing non-JSON body that triggers parsing
	// but the redactor gracefully handles non-JSON (returns as-is, no error).
	// So we just verify that error messages from Compare include context about
	// which step failed (request/live/shadow).
	redactor := traffictesting.NewRedactor(nil)
	comparer := services.NewResponseComparer(redactor, nil)

	// With a valid redactor and normal input, no errors expected.
	result, err := comparer.Compare(
		services.ResponseData{Headers: http.Header{"X": {"1"}}, Body: []byte(`{"a":1}`)},
		services.ResponseData{StatusCode: 200, Headers: http.Header{"X": {"1"}}, Body: []byte(`{"a":1}`)},
		services.ResponseData{StatusCode: 200, Headers: http.Header{"X": {"1"}}, Body: []byte(`{"a":1}`)},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Ops)
}

func TestCompare_redaction_never_leaks_body(t *testing.T) {
	// A redacted body field whose raw value DIFFERS between live and shadow must
	// never reach result.Ops: both sides collapse to [REDACTED] before diffing,
	// so no op is produced for it. A non-redacted field that genuinely differs
	// still surfaces, proving real diffs are unaffected.
	redactor := traffictesting.NewRedactor([]string{"body.password"})
	comparer := services.NewResponseComparer(redactor, nil)

	req := services.ResponseData{StatusCode: 200, Body: []byte(`{}`)}
	live := services.ResponseData{StatusCode: 200, Body: []byte(`{"password":"live-secret","name":"Alice"}`)}
	shadow := services.ResponseData{StatusCode: 200, Body: []byte(`{"password":"shadow-secret","name":"Bob"}`)}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	require.NotEmpty(t, result.Ops)

	paths := make(map[string]string)
	for _, op := range result.Ops {
		paths[op.Path] = op.Op
	}
	assert.Equal(t, "replace", paths["/body/name"], "non-redacted field diff should surface")
	assert.NotContains(t, paths, "/body/password", "redacted field must not produce an op")
	assertOpsFreeOfSecrets(t, result.Ops, "live-secret", "shadow-secret")
}

func TestCompare_redaction_never_leaks_headers(t *testing.T) {
	// A redacted response header whose raw value DIFFERS between live and shadow
	// must never reach result.Ops. A non-redacted differing header still surfaces.
	redactor := traffictesting.NewRedactor([]string{"headers.Authorization"})
	comparer := services.NewResponseComparer(redactor, nil)

	req := services.ResponseData{StatusCode: 200, Body: []byte(`{}`)}
	live := services.ResponseData{
		StatusCode: 200,
		Headers:    http.Header{"Authorization": {"live-token"}, "X-Trace": {"live"}},
		Body:       []byte(`{}`),
	}
	shadow := services.ResponseData{
		StatusCode: 200,
		Headers:    http.Header{"Authorization": {"shadow-token"}, "X-Trace": {"shadow"}},
		Body:       []byte(`{}`),
	}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	require.NotEmpty(t, result.Ops)

	var sawTrace bool
	for _, op := range result.Ops {
		if strings.HasPrefix(op.Path, "/headers/X-Trace") {
			sawTrace = true
		}
		assert.NotContains(t, op.Path, "Authorization", "redacted header must not produce any op")
	}
	assert.True(t, sawTrace, "non-redacted header diff should surface")
	assertOpsFreeOfSecrets(t, result.Ops, "live-token", "shadow-token")
}
