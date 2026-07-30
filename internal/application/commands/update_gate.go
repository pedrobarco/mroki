package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// UpdateGateCommand represents the intent to update an existing gate.
// All fields are optional — only non-nil fields are applied.
//
// Retention is a tri-state pointer: nil leaves the current value untouched, an
// empty string (including a JSON null, which the HTTP layer maps to "") resets
// the gate to the global retention floor, and a non-empty Go duration string
// sets a custom retention (which must be >= the global floor). Surrounding
// whitespace is trimmed before parsing.
type UpdateGateCommand struct {
	ID          string
	Name        *string
	DiffConfig  *UpdateDiffConfigProps
	RedactedFields *UpdateRedactedFieldsProps
	Retention   *string
}

// UpdateDiffConfigProps holds the diff configuration fields for update.
type UpdateDiffConfigProps struct {
	IgnoredFields  []string
	IncludedFields []string
	FloatTolerance float64
	SortArrays     bool
}

// UpdateRedactedFieldsProps holds the redacted fields configuration for update.
type UpdateRedactedFieldsProps struct {
	AdditionalFields []string
}

// UpdateGateHandler handles the UpdateGate command
type UpdateGateHandler struct {
	repo             traffictesting.GateRepository
	globalRetention  time.Duration
}

// NewUpdateGateHandler creates a new UpdateGateHandler.
// globalRetention is the retention floor: a custom per-gate retention must be
// greater than or equal to this value.
func NewUpdateGateHandler(repo traffictesting.GateRepository, globalRetention time.Duration) *UpdateGateHandler {
	return &UpdateGateHandler{repo: repo, globalRetention: globalRetention}
}

// Handle executes the UpdateGate command
func (h *UpdateGateHandler) Handle(ctx context.Context, cmd UpdateGateCommand) (*traffictesting.Gate, error) {
	// Parse and validate gate ID
	id, err := traffictesting.ParseGateID(cmd.ID)
	if err != nil {
		return nil, err
	}

	// Fetch existing gate
	gate, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply name if provided
	if cmd.Name != nil {
		name, err := traffictesting.ParseGateName(*cmd.Name)
		if err != nil {
			return nil, fmt.Errorf("invalid name: %w", err)
		}
		gate.Name = name
	}

	// Apply diff config if provided
	if cmd.DiffConfig != nil {
		diffConfig, err := traffictesting.NewDiffConfig(
			cmd.DiffConfig.IgnoredFields,
			cmd.DiffConfig.IncludedFields,
			cmd.DiffConfig.FloatTolerance,
			cmd.DiffConfig.SortArrays,
		)
		if err != nil {
			return nil, err
		}
		gate.DiffConfig = diffConfig
	}

	// Apply redacted fields if provided
	if cmd.RedactedFields != nil {
		redactedFields, err := traffictesting.NewRedactedFields(cmd.RedactedFields.AdditionalFields)
		if err != nil {
			return nil, err
		}
		gate.RedactedFields = redactedFields
	}

	// Apply retention if provided. An empty string (after trimming) resets the
	// gate to the global floor; a non-empty duration must be valid and >= the
	// global floor.
	if cmd.Retention != nil {
		v := strings.TrimSpace(*cmd.Retention)
		if v == "" {
			gate.Retention = traffictesting.NoRetention()
		} else {
			retention, err := traffictesting.ParseRetention(v)
			if err != nil {
				return nil, err
			}
			if retention.Duration() < h.globalRetention {
				return nil, fmt.Errorf("%w: %s is below the global minimum of %s",
					traffictesting.ErrRetentionBelowMinimum, retention.Duration(), h.globalRetention)
			}
			gate.Retention = retention
		}
	}

	// Persist
	if err := h.repo.Update(ctx, gate); err != nil {
		return nil, fmt.Errorf("failed to update gate: %w", err)
	}

	return gate, nil
}
