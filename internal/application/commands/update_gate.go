package commands

import (
	"context"
	"fmt"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// UpdateGateCommand represents the intent to update an existing gate.
// All fields are optional — only non-nil fields are applied.
type UpdateGateCommand struct {
	ID             string
	Name           *string
	DiffConfig     *UpdateDiffConfigProps
}

// UpdateDiffConfigProps holds the diff configuration fields for update.
type UpdateDiffConfigProps struct {
	IgnoredFields  []string
	IncludedFields []string
	FloatTolerance float64
}

// UpdateGateHandler handles the UpdateGate command
type UpdateGateHandler struct {
	repo traffictesting.GateRepository
}

// NewUpdateGateHandler creates a new UpdateGateHandler
func NewUpdateGateHandler(repo traffictesting.GateRepository) *UpdateGateHandler {
	return &UpdateGateHandler{repo: repo}
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
		)
		if err != nil {
			return nil, err
		}
		gate.DiffConfig = diffConfig
	}

	// Persist
	if err := h.repo.Update(ctx, gate); err != nil {
		return nil, fmt.Errorf("failed to update gate: %w", err)
	}

	return gate, nil
}
