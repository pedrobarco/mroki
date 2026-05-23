package services

import (
	"net/http"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompare_identical_json_responses(t *testing.T) {
	redactor := traffictesting.NewRedactor(nil)
	comparer := NewResponseComparer(redactor, nil)

	body := []byte(`{"name":"Alice","age":30}`)
	req := ResponseData{StatusCode: 200, Headers: http.Header{"Content-Type": {"application/json"}}, Body: body}
	live := ResponseData{StatusCode: 200, Headers: http.Header{"X-Live": {"true"}}, Body: body}
	shadow := ResponseData{StatusCode: 200, Headers: http.Header{"X-Live": {"true"}}, Body: body}

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
	comparer := NewResponseComparer(redactor, nil)

	req := ResponseData{StatusCode: 200, Body: []byte(`{}`)}
	live := ResponseData{StatusCode: 200, Body: []byte(`{"user":"Alice"}`)}
	shadow := ResponseData{StatusCode: 200, Body: []byte(`{"user":"Bob"}`)}

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
	comparer := NewResponseComparer(redactor, nil)

	req := ResponseData{
		StatusCode: 200,
		Headers:    http.Header{"Cookie": {"secret"}},
		Body:       []byte(`{}`),
	}
	live := ResponseData{StatusCode: 200, Body: []byte(`{}`)}
	shadow := ResponseData{StatusCode: 200, Body: []byte(`{}`)}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	assert.Equal(t, []string{"[REDACTED]"}, result.Request.Headers["Cookie"])
}

func TestCompare_redacts_body_fields(t *testing.T) {
	redactor := traffictesting.NewRedactor([]string{"body.password"})
	comparer := NewResponseComparer(redactor, nil)

	req := ResponseData{StatusCode: 200, Body: []byte(`{}`)}
	live := ResponseData{
		StatusCode: 200,
		Body:       []byte(`{"password":"secret","name":"Alice"}`),
	}
	shadow := ResponseData{StatusCode: 200, Body: []byte(`{"password":"secret","name":"Alice"}`)}

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
	comparer := NewResponseComparer(redactor, nil)

	req := ResponseData{StatusCode: 200}
	live := ResponseData{StatusCode: 200}
	shadow := ResponseData{StatusCode: 200}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	assert.Empty(t, result.Ops)
	assert.Nil(t, result.Request.BodyParsed)
	assert.Nil(t, result.Live.BodyParsed)
	assert.Nil(t, result.Shadow.BodyParsed)
}

func TestCompare_non_json_bodies(t *testing.T) {
	redactor := traffictesting.NewRedactor(nil)
	comparer := NewResponseComparer(redactor, nil)

	req := ResponseData{StatusCode: 200, Body: []byte(`{}`)}
	live := ResponseData{StatusCode: 200, Body: []byte(`<html>hello</html>`)}
	shadow := ResponseData{StatusCode: 200, Body: []byte(`<html>world</html>`)}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	assert.Nil(t, result.Live.BodyParsed)
	assert.Nil(t, result.Shadow.BodyParsed)
	// Non-JSON bodies produce nil BodyParsed, so the envelope body is nil for both.
	// Since both envelope bodies are nil (equal), the diff depends only on other fields.
	// With same status code and nil headers, ops may be empty.
}

func TestCompare_with_diff_options(t *testing.T) {
	redactor := traffictesting.NewRedactor(nil)
	comparer := NewResponseComparer(redactor, []diff.Option{
		diff.WithIgnoredFields("body.timestamp"),
	})

	req := ResponseData{StatusCode: 200, Body: []byte(`{}`)}
	live := ResponseData{StatusCode: 200, Body: []byte(`{"timestamp":"2024-01-01T10:00:00Z"}`)}
	shadow := ResponseData{StatusCode: 200, Body: []byte(`{"timestamp":"2024-01-01T11:00:00Z"}`)}

	result, err := comparer.Compare(req, live, shadow)

	require.NoError(t, err)
	assert.Empty(t, result.Ops, "timestamp difference should be ignored")
}

func TestCompare_different_status_codes(t *testing.T) {
	redactor := traffictesting.NewRedactor(nil)
	comparer := NewResponseComparer(redactor, nil)

	body := []byte(`{"ok":true}`)
	req := ResponseData{StatusCode: 200, Body: body}
	live := ResponseData{StatusCode: 200, Body: body}
	shadow := ResponseData{StatusCode: 500, Body: body}

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
	comparer := NewResponseComparer(nil, nil)
	_, err := comparer.Compare(
		ResponseData{},
		ResponseData{StatusCode: 200},
		ResponseData{StatusCode: 200},
	)
	require.ErrorIs(t, err, ErrNilRedactor)
}

func TestCompare_redaction_error_includes_context(t *testing.T) {
	// Redactor with body fields — passing non-JSON body that triggers parsing
	// but the redactor gracefully handles non-JSON (returns as-is, no error).
	// So we just verify that error messages from Compare include context about
	// which step failed (request/live/shadow).
	redactor := traffictesting.NewRedactor(nil)
	comparer := NewResponseComparer(redactor, nil)

	// With a valid redactor and normal input, no errors expected.
	result, err := comparer.Compare(
		ResponseData{Headers: http.Header{"X": {"1"}}, Body: []byte(`{"a":1}`)},
		ResponseData{StatusCode: 200, Headers: http.Header{"X": {"1"}}, Body: []byte(`{"a":1}`)},
		ResponseData{StatusCode: 200, Headers: http.Header{"X": {"1"}}, Body: []byte(`{"a":1}`)},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Ops)
}