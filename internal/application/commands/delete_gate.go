package commands

import (
	"context"
	"fmt"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// DeleteGateCommand represents the intent to delete a gate
type DeleteGateCommand struct {
	ID string
}

// DeleteGateHandler handles the DeleteGate command
type DeleteGateHandler struct {
	repo traffictesting.GateRepository
}

// NewDeleteGateHandler creates a new DeleteGateHandler
func NewDeleteGateHandler(repo traffictesting.GateRepository) *DeleteGateHandler {
	return &DeleteGateHandler{repo: repo}
}

// Handle executes the DeleteGate command
func (h *DeleteGateHandler) Handle(ctx context.Context, cmd DeleteGateCommand) error {
	id, err := traffictesting.ParseGateID(cmd.ID)
	if err != nil {
		return fmt.Errorf("invalid gate ID: %w", err)
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}
