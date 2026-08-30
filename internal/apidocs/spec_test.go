package apidocs

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// TestOpenAPISpecLints runs the Speakeasy linter against the committed OpenAPI
// spec. The linter performs full structural validation in addition to the
// configured lint rules, so this single test guards the spec against both
// validation and lint regressions. It runs as part of `go test ./...`, which is
// what CI executes, so no dedicated CI job is required.
func TestOpenAPISpecLints(t *testing.T) {
	root := RepoRoot()

	cmd := exec.Command(
		"go", "tool", "openapi", "spec", "lint",
		"--config", filepath.Join(root, LintConfig),
		filepath.Join(root, SpecFile),
	)
	cmd.Dir = root

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("openapi spec lint failed: %v\n%s", err, out)
	}
}

// TestReferenceUpToDate runs the generator in check mode to verify the committed
// docs/api/REFERENCE.md still matches what the OpenAPI spec would produce. This
// is the same drift guard used by `make api-docs-check` and the pre-commit hook,
// wired into `go test ./...` so CI fails when the reference is out of date. Fix a
// failure by running `make api-docs` and committing the regenerated reference.
func TestReferenceUpToDate(t *testing.T) {
	root := RepoRoot()

	cmd := exec.Command("go", "run", "./internal/apidocs/cmd/apidocs", "-check")
	cmd.Dir = root

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s is out of date; run `make api-docs` and commit the result: %v\n%s",
			ReferenceFile, err, out)
	}
}
