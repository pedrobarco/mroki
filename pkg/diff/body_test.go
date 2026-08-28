package diff_test

import (
	"testing"

	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        diff.BodyKind
	}{
		{"empty defaults to json", "", diff.BodyKindJSON},
		{"unparseable defaults to json", "???", diff.BodyKindJSON},
		{"application/json", "application/json", diff.BodyKindJSON},
		{"json with charset", "application/json; charset=utf-8", diff.BodyKindJSON},
		{"json suffix", "application/vnd.api+json", diff.BodyKindJSON},
		{"text plain", "text/plain", diff.BodyKindText},
		{"text html with charset", "text/html; charset=utf-8", diff.BodyKindText},
		{"application xml", "application/xml", diff.BodyKindText},
		{"xml suffix", "application/atom+xml", diff.BodyKindText},
		{"form urlencoded", "application/x-www-form-urlencoded", diff.BodyKindText},
		{"octet stream is binary", "application/octet-stream", diff.BodyKindBinary},
		{"image is binary", "image/png", diff.BodyKindBinary},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, diff.ClassifyContentType(tt.contentType))
		})
	}
}

func TestEmbedBody_empty_body_is_nil(t *testing.T) {
	assert.Nil(t, diff.EmbedBody("application/json", nil, nil))
	assert.Nil(t, diff.EmbedBody("text/plain", []byte{}, nil))
	assert.Nil(t, diff.EmbedBody("application/octet-stream", nil, nil))
}

func TestEmbedBody_json_uses_parsed_when_present(t *testing.T) {
	parsed := map[string]any{"a": float64(1)}
	got := diff.EmbedBody("application/json", []byte(`{"ignored":true}`), parsed)
	// parsed tree is returned as-is (e.g. the redactor's already-redacted body).
	assert.Equal(t, parsed, got)
}

func TestEmbedBody_json_parses_body_when_no_parsed(t *testing.T) {
	got := diff.EmbedBody("application/json", []byte(`{"a":1}`), nil)
	assert.Equal(t, map[string]any{"a": float64(1)}, got)
}

func TestEmbedBody_invalid_json_falls_back_to_raw_string(t *testing.T) {
	got := diff.EmbedBody("application/json", []byte(`not json`), nil)
	assert.Equal(t, "not json", got)
}

func TestEmbedBody_text_is_raw_string(t *testing.T) {
	got := diff.EmbedBody("text/plain; charset=utf-8", []byte("hello world"), nil)
	assert.Equal(t, "hello world", got)
}

func TestEmbedBody_binary_is_metadata_note(t *testing.T) {
	got := diff.EmbedBody("application/octet-stream", []byte{0x00, 0x01, 0x02}, nil)

	m, ok := got.(map[string]any)
	require.True(t, ok)
	note, ok := m["_mroki"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "application/octet-stream", note["contentType"])
	// size is float64 so it compares cleanly against json.Unmarshal numbers.
	assert.Equal(t, float64(3), note["size"])
	// sha256 lets same-length byte changes surface as a diff.
	assert.NotEmpty(t, note["sha256"])
}

func TestEmbedBody_binary_same_size_different_bytes_differ_by_hash(t *testing.T) {
	a := diff.EmbedBody("application/octet-stream", []byte{0x00, 0x01, 0x02}, nil).(map[string]any)
	b := diff.EmbedBody("application/octet-stream", []byte{0xFF, 0xFE, 0xFD}, nil).(map[string]any)

	an := a["_mroki"].(map[string]any)
	bn := b["_mroki"].(map[string]any)
	assert.Equal(t, an["size"], bn["size"])
	assert.NotEqual(t, an["sha256"], bn["sha256"])
}
