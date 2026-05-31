package ent

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/ent"
	"github.com/pedrobarco/mroki/ent/schema"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

func mapGateToDomain(raw *ent.Gate) (*traffictesting.Gate, error) {
	id, err := traffictesting.ParseGateID(raw.ID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid gate ID in database: %w", err)
	}

	name, err := traffictesting.ParseGateName(raw.Name)
	if err != nil {
		return nil, fmt.Errorf("invalid gate name in database: %w", err)
	}

	live, err := traffictesting.ParseGateURL(raw.LiveURL)
	if err != nil {
		return nil, fmt.Errorf("invalid live URL in database: %w", err)
	}

	shadow, err := traffictesting.ParseGateURL(raw.ShadowURL)
	if err != nil {
		return nil, fmt.Errorf("invalid shadow URL in database: %w", err)
	}

	ignoredFields := raw.DiffIgnoredFields
	if ignoredFields == nil {
		ignoredFields = []string{}
	}
	includedFields := raw.DiffIncludedFields
	if includedFields == nil {
		includedFields = []string{}
	}

	diffConfig, err := traffictesting.NewDiffConfig(ignoredFields, includedFields, raw.DiffFloatTolerance, raw.DiffSortArrays)
	if err != nil {
		return nil, fmt.Errorf("invalid diff config in database: %w", err)
	}

	redactedFieldList := raw.RedactedFields
	if redactedFieldList == nil {
		redactedFieldList = []string{}
	}
	redactedFields, err := traffictesting.NewRedactedFields(redactedFieldList)
	if err != nil {
		return nil, fmt.Errorf("invalid redacted fields in database: %w", err)
	}

	return traffictesting.NewGate(name, live, shadow,
		traffictesting.WithGateID(id),
		traffictesting.WithGateCreatedAt(raw.CreatedAt),
		traffictesting.WithGateDiffConfig(diffConfig),
		traffictesting.WithGateRedactedFields(redactedFields),
	)
}

func mapRequestToDomain(raw *ent.Request) (*traffictesting.Request, error) {
	id, err := traffictesting.ParseRequestID(raw.ID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid request ID in database: %w", err)
	}

	gateID, err := traffictesting.ParseGateID(raw.GateID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid gate ID in database: %w", err)
	}

	method, err := traffictesting.NewHTTPMethod(raw.Method)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP method in database: %w", err)
	}

	path, err := traffictesting.ParsePath(raw.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid path in database: %w", err)
	}

	headers := traffictesting.NewHeaders(http.Header(raw.Headers))

	return &traffictesting.Request{
		ID:        id,
		GateID:    gateID,
		Method:    method,
		Path:      path,
		RawQuery:  raw.RawQuery,
		Headers:   headers,
		Body:      raw.Body,
		CreatedAt: raw.CreatedAt,
	}, nil
}

func mapResponseToDomain(raw *ent.Response) (traffictesting.Response, error) {
	statusCode, err := traffictesting.ParseStatusCode(int(raw.StatusCode))
	if err != nil {
		return traffictesting.Response{}, fmt.Errorf("invalid status code in database: %w", err)
	}

	headers := traffictesting.NewHeaders(http.Header(raw.Headers))

	return traffictesting.Response{
		ID:         raw.ID,
		StatusCode: statusCode,
		Headers:    headers,
		Body:       raw.Body,
		LatencyMs:  raw.LatencyMs,
		CreatedAt:  raw.CreatedAt,
	}, nil
}

func mapDiffToDomain(raw *ent.Diff) traffictesting.Diff {
	if raw == nil || raw.FromResponseID == uuid.Nil {
		return traffictesting.Diff{}
	}
	return traffictesting.Diff{
		Content: raw.Content,
		Config:  mapDiffConfigSnapshotToDomain(raw.Config),
	}
}

func mapDiffConfigToPersistence(cfg traffictesting.DiffConfig) schema.DiffConfigSnapshot {
	return schema.DiffConfigSnapshot{
		SortArrays:     cfg.SortArrays,
		IgnoredFields:  cfg.IgnoredFields,
		IncludedFields: cfg.IncludedFields,
		FloatTolerance: cfg.FloatTolerance,
	}
}

func mapDiffConfigSnapshotToDomain(s schema.DiffConfigSnapshot) traffictesting.DiffConfig {
	ignoredFields := s.IgnoredFields
	if ignoredFields == nil {
		ignoredFields = []string{}
	}
	includedFields := s.IncludedFields
	if includedFields == nil {
		includedFields = []string{}
	}
	return traffictesting.DiffConfig{
		SortArrays:     s.SortArrays,
		IgnoredFields:  ignoredFields,
		IncludedFields: includedFields,
		FloatTolerance: s.FloatTolerance,
	}
}
