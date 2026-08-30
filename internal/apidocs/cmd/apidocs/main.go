// Command apidocs generates the Markdown API reference (docs/api/REFERENCE.md)
// from the OpenAPI specification under docs/api/openapi/.
//
// It bundles the multi-file spec into a single self-contained document with
// github.com/pb33f/libopenapi (composed mode, so external $refs are lifted into
// components as internal pointers), then renders that document to Markdown with
// github.com/duh-rpc/openapi-markdown.go.
//
// Usage:
//
//	go run ./internal/apidocs/cmd/apidocs           # write docs/api/REFERENCE.md
//	go run ./internal/apidocs/cmd/apidocs -check     # verify the committed file
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	conv "github.com/duh-rpc/openapi-markdown.go"
	"github.com/pb33f/libopenapi/bundler"
	"github.com/pb33f/libopenapi/datamodel"

	"github.com/pedrobarco/mroki/internal/apidocs"
)

// referenceIntro is the prose that precedes the generated endpoint reference.
// The Markdown converter does not carry over the spec's info.description, so the
// cross-cutting conventions (auth, envelopes, RFC 7807 errors) are supplied here.
const referenceIntro = "" +
	"This file is generated from the OpenAPI specification under `docs/api/openapi/`. " +
	"**Do not edit it by hand** — run `make api-docs` to regenerate it.\n\n" +
	"## Base URL\n\n" +
	"The local development server is available at `http://localhost:8090`.\n\n" +
	"## Authentication\n\n" +
	"Every endpoint except the infrastructure endpoints (`/health/live`, `/health/ready`, " +
	"`/metrics`) requires a bearer token supplied as `Authorization: Bearer <your-api-key>`.\n\n" +
	"## Response envelope\n\n" +
	"Successful responses wrap their payload in a `data` field. List endpoints add a " +
	"`pagination` object (`limit`, `offset`, `total`, `has_more`).\n\n" +
	"## Errors\n\n" +
	"Errors are returned as `application/json` following " +
	"[RFC 7807](https://www.rfc-editor.org/rfc/rfc7807) with the fields `type`, `title`, " +
	"`status`, `detail`, and (for 4xx) `instance`. The `type` is a relative URI such as " +
	"`/errors/not-found`, `/errors/invalid-request-body`, `/errors/invalid-query-param`, " +
	"`/errors/unauthorized`, `/errors/conflict`, `/errors/rate-limit-exceeded`, or " +
	"`/errors/internal-error`. See the `Problem` schema below."

func main() {
	check := flag.Bool("check", false, "verify the committed reference matches the spec instead of writing it")
	flag.Parse()

	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, "apidocs:", err)
		os.Exit(1)
	}
}

func run(check bool) error {
	root := apidocs.RepoRoot()

	bundled, err := bundle(root)
	if err != nil {
		return err
	}

	res, err := conv.Convert(bundled, conv.ConvertOptions{
		Title:               "mroki API Reference",
		Description:         referenceIntro,
		EnableSharedSchemas: true,
	})
	if err != nil {
		return fmt.Errorf("convert spec to markdown: %w", err)
	}
	for _, w := range res.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}

	refPath := filepath.Join(root, apidocs.ReferenceFile)

	if check {
		existing, err := os.ReadFile(refPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", apidocs.ReferenceFile, err)
		}
		if !bytes.Equal(existing, res.Markdown) {
			return fmt.Errorf("%s is out of date; run `make api-docs` and commit the result", apidocs.ReferenceFile)
		}
		fmt.Printf("%s is up to date (%d endpoints, %d tags)\n", apidocs.ReferenceFile, res.EndpointCount, res.TagCount)
		return nil
	}

	if err := os.WriteFile(refPath, res.Markdown, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", apidocs.ReferenceFile, err)
	}
	fmt.Printf("wrote %s (%d endpoints, %d tags)\n", apidocs.ReferenceFile, res.EndpointCount, res.TagCount)
	return nil
}

// bundle reads the multi-file spec and composes it into a single, self-contained
// OpenAPI document. External $refs are lifted into the components section and
// rewritten as internal pointers (which the Markdown converter requires), rather
// than inlined. BasePath is the spec directory so relative file refs resolve.
func bundle(root string) ([]byte, error) {
	specPath := filepath.Join(root, apidocs.SpecFile)

	src, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}

	config := &datamodel.DocumentConfiguration{
		BasePath: filepath.Dir(specPath),
	}

	bundled, err := bundler.BundleBytesComposed(src, config, nil)
	if err != nil {
		return nil, fmt.Errorf("bundle spec: %w", err)
	}
	return bundled, nil
}
