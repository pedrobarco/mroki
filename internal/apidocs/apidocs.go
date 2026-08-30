// Package apidocs holds the tooling that keeps the mroki API documentation in
// sync with its OpenAPI specification: it lints the spec (exercised by a test
// so CI enforces it) and generates the Markdown API reference from it.
//
// The spec itself lives under docs/api/openapi/ as a multi-file OpenAPI 3.1
// document. The Speakeasy CLI (wired as a `go tool` dependency) validates and
// lints it; the generator in cmd/apidocs bundles it with libopenapi and renders
// the bundled document to Markdown.
package apidocs

import (
	"path/filepath"
	"runtime"
)

// Paths below are relative to the repository root; join them with RepoRoot.
const (
	// SpecFile is the entrypoint of the multi-file OpenAPI 3.1 document.
	SpecFile = "docs/api/openapi/openapi.yaml"
	// LintConfig is the Speakeasy linter configuration for the spec.
	LintConfig = "docs/api/openapi/lint.yaml"
	// ReferenceFile is the generated Markdown API reference.
	ReferenceFile = "docs/api/REFERENCE.md"
)

// RepoRoot returns the absolute path to the repository root, resolved relative
// to this source file so it does not depend on the process working directory.
func RepoRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}
