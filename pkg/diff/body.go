package diff

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"mime"
	"strings"

	"github.com/pedrobarco/mroki/pkg/jsontree"
)

// BodyKind classifies how a response body is embedded in the diff envelope,
// based on its Content-Type.
type BodyKind int

const (
	// BodyKindJSON embeds the body as a parsed value tree so it is compared as
	// structured JSON.
	BodyKindJSON BodyKind = iota
	// BodyKindText embeds the body as a string so it is compared as raw text.
	BodyKindText
	// BodyKindBinary skips the body, embedding a metadata note in its place.
	BodyKindBinary
)

// ClassifyContentType maps a Content-Type header value to a BodyKind. A missing
// or unparseable Content-Type defaults to JSON to preserve backward-compatible
// behaviour (bodies were previously always treated as JSON).
func ClassifyContentType(contentType string) BodyKind {
	if strings.TrimSpace(contentType) == "" {
		return BodyKindJSON
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return BodyKindJSON
	}
	mediaType = strings.ToLower(mediaType)

	if mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") {
		return BodyKindJSON
	}
	if isTextMediaType(mediaType) {
		return BodyKindText
	}
	return BodyKindBinary
}

// isTextMediaType reports whether a (lower-cased, parameter-stripped) media type
// is a human-readable text format that should be diffed as a raw string.
func isTextMediaType(mediaType string) bool {
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	if strings.HasSuffix(mediaType, "+xml") {
		return true
	}
	switch mediaType {
	case "application/xml",
		"application/javascript",
		"application/ecmascript",
		"application/x-www-form-urlencoded",
		"application/graphql":
		return true
	}
	return false
}

// EmbedBody converts a raw response body into the value used as the envelope's
// "body" field (see BuildEnvelope), choosing the representation from the
// Content-Type:
//
//   - JSON: the parsed value tree. When parsed is non-nil it is used as-is
//     (e.g. the redactor's already-redacted BodyParsed); otherwise body is
//     parsed here. A JSON body that fails to parse falls back to a raw string
//     so a diff still surfaces (best-effort; live traffic is never failed).
//   - Text: the body as a string.
//   - Binary: a metadata note recording contentType, size and a SHA-256 of the
//     bytes, so any change surfaces without embedding non-textual content.
//
// An empty body always yields nil (JSON null) regardless of Content-Type,
// matching the previous behaviour of an absent body.
func EmbedBody(contentType string, body []byte, parsed jsontree.Tree) jsontree.Tree {
	if parsed == nil && len(body) == 0 {
		return nil
	}

	switch ClassifyContentType(contentType) {
	case BodyKindText:
		return string(body)
	case BodyKindBinary:
		return binaryMetadataNote(contentType, body)
	default: // BodyKindJSON
		if parsed != nil {
			return parsed
		}
		var tree jsontree.Tree
		if err := json.Unmarshal(body, &tree); err != nil {
			// Mislabeled body: fall back to raw string so a diff still surfaces.
			return string(body)
		}
		return tree
	}
}

// binaryMetadataNote stands in for a binary body that is not embedded. The
// SHA-256 fingerprint makes any content change diff-detectable (size alone would
// miss same-length edits); size and contentType add human-readable context, and
// reason documents the omission. size is a float64 to match the numbers
// json.Unmarshal produces on the other side of the comparison.
func binaryMetadataNote(contentType string, body []byte) jsontree.Tree {
	sum := sha256.Sum256(body)
	return map[string]any{
		"_mroki": map[string]any{
			"reason":      "binary content not embedded",
			"contentType": contentType,
			"size":        float64(len(body)),
			"sha256":      hex.EncodeToString(sum[:]),
		},
	}
}
