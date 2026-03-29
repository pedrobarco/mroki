package client

import (
	"github.com/pedrobarco/mroki/pkg/dto"
)

// Type aliases for backward compatibility
// These allow existing client code to continue using the old names
// while internally using the centralized DTOs

// CapturedRequest represents a complete request/response pair with diff.
// This is what gets sent from proxy to API.
type CapturedRequest = dto.CreateRequestPayload

// CapturedResponse represents a single HTTP response (live or shadow).
type CapturedResponse = dto.ResponsePayload

// CapturedDiff contains the computed difference between responses.
type CapturedDiff = dto.DiffPayload
