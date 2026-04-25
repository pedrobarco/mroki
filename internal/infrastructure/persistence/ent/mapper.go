package ent

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/ent"
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

	diffConfig, err := traffictesting.NewDiffConfig(ignoredFields, includedFields, raw.DiffFloatTolerance)
	if err != nil {
		return nil, fmt.Errorf("invalid diff config in database: %w", err)
	}

	return traffictesting.NewGate(name, live, shadow,
		traffictesting.WithGateID(id),
		traffictesting.WithGateCreatedAt(raw.CreatedAt),
		traffictesting.WithGateDiffConfig(diffConfig),
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
		FromResponseID: raw.FromResponseID,
		ToResponseID:   raw.ToResponseID,
		Content:        raw.Content,
		CreatedAt:      raw.CreatedAt,
	}
}
