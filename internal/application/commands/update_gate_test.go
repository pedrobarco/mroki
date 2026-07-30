package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testGlobalRetention is the global retention floor used by update_gate tests.
const testGlobalRetention = 720 * time.Hour

// mockGateRepo is a mock implementation of GateRepository for update_gate tests.
type mockGateRepo struct {
	gate    *traffictesting.Gate
	updated *traffictesting.Gate
}

func newMockGateRepo() *mockGateRepo {
	name, _ := traffictesting.ParseGateName("test-gate")
	live, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadow, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(name, live, shadow)
	return &mockGateRepo{gate: gate}
}

func (m *mockGateRepo) Save(_ context.Context, g *traffictesting.Gate) error { return nil }
func (m *mockGateRepo) Delete(_ context.Context, _ traffictesting.GateID) error { return nil }
func (m *mockGateRepo) GetAll(_ context.Context, _ traffictesting.GateFilters, _ traffictesting.GateSort, _ *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
	return nil, errors.New("not implemented")
}

func (m *mockGateRepo) ListRetentions(_ context.Context) ([]traffictesting.GateRetention, error) {
	return nil, errors.New("not implemented")
}

func (m *mockGateRepo) GetByID(_ context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
	if m.gate == nil {
		return nil, traffictesting.ErrGateNotFound
	}
	return m.gate, nil
}

func (m *mockGateRepo) Update(_ context.Context, g *traffictesting.Gate) error {
	m.updated = g
	return nil
}

func TestUpdateGateHandler_valid_redacted_fields(t *testing.T) {
	repo := newMockGateRepo()
	handler := NewUpdateGateHandler(repo, testGlobalRetention)

	cmd := UpdateGateCommand{
		ID: repo.gate.ID.String(),
		RedactedFields: &UpdateRedactedFieldsProps{
			AdditionalFields: []string{"headers.X-Internal-Token", "body.user.password"},
		},
	}

	gate, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, gate)
	assert.Equal(t, []string{"headers.X-Internal-Token", "body.user.password"}, gate.RedactedFields.AdditionalFields)
	assert.Equal(t, gate, repo.updated)
}

func TestUpdateGateHandler_invalid_redacted_fields_missing_prefix(t *testing.T) {
	repo := newMockGateRepo()
	handler := NewUpdateGateHandler(repo, testGlobalRetention)

	cmd := UpdateGateCommand{
		ID: repo.gate.ID.String(),
		RedactedFields: &UpdateRedactedFieldsProps{
			AdditionalFields: []string{"Authorization"},
		},
	}

	gate, err := handler.Handle(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, gate)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidRedactedFields)
}

func TestUpdateGateHandler_invalid_redacted_fields_bare_prefix(t *testing.T) {
	repo := newMockGateRepo()
	handler := NewUpdateGateHandler(repo, testGlobalRetention)

	cmd := UpdateGateCommand{
		ID: repo.gate.ID.String(),
		RedactedFields: &UpdateRedactedFieldsProps{
			AdditionalFields: []string{"headers."},
		},
	}

	gate, err := handler.Handle(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, gate)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidRedactedFields)
}

func TestUpdateGateHandler_partial_update_only_redacted_fields(t *testing.T) {
	repo := newMockGateRepo()
	originalName := repo.gate.Name
	handler := NewUpdateGateHandler(repo, testGlobalRetention)

	cmd := UpdateGateCommand{
		ID: repo.gate.ID.String(),
		RedactedFields: &UpdateRedactedFieldsProps{
			AdditionalFields: []string{"body.token"},
		},
	}

	gate, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	assert.Equal(t, originalName, gate.Name, "name should be unchanged")
	assert.Equal(t, []string{"body.token"}, gate.RedactedFields.AdditionalFields)
}

func TestUpdateGateHandler_combined_name_and_redacted_fields(t *testing.T) {
	repo := newMockGateRepo()
	handler := NewUpdateGateHandler(repo, testGlobalRetention)

	newName := "updated-gate"
	cmd := UpdateGateCommand{
		ID:   repo.gate.ID.String(),
		Name: &newName,
		RedactedFields: &UpdateRedactedFieldsProps{
			AdditionalFields: []string{"headers.X-Secret"},
		},
	}

	gate, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	assert.Equal(t, newName, gate.Name.String())
	assert.Equal(t, []string{"headers.X-Secret"}, gate.RedactedFields.AdditionalFields)
}

func TestUpdateGateHandler_gate_not_found(t *testing.T) {
	repo := &mockGateRepo{gate: nil}
	handler := NewUpdateGateHandler(repo, testGlobalRetention)

	cmd := UpdateGateCommand{
		ID:   traffictesting.NewGateID().String(),
		Name: strPtr("new-name"),
	}

	gate, err := handler.Handle(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, gate)
	assert.ErrorIs(t, err, traffictesting.ErrGateNotFound)
}

func TestUpdateGateHandler_set_custom_retention(t *testing.T) {
	repo := newMockGateRepo()
	handler := NewUpdateGateHandler(repo, testGlobalRetention)

	cmd := UpdateGateCommand{
		ID:        repo.gate.ID.String(),
		Retention: strPtr("1000h"),
	}

	gate, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, gate)
	assert.True(t, gate.Retention.IsSet())
	assert.Equal(t, 1000*time.Hour, gate.Retention.Duration())
	assert.Equal(t, gate, repo.updated)
}

func TestUpdateGateHandler_retention_equal_to_global_is_allowed(t *testing.T) {
	repo := newMockGateRepo()
	handler := NewUpdateGateHandler(repo, testGlobalRetention)

	cmd := UpdateGateCommand{
		ID:        repo.gate.ID.String(),
		Retention: strPtr(testGlobalRetention.String()),
	}

	gate, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	assert.Equal(t, testGlobalRetention, gate.Retention.Duration())
}

func TestUpdateGateHandler_retention_below_global_is_rejected(t *testing.T) {
	repo := newMockGateRepo()
	handler := NewUpdateGateHandler(repo, testGlobalRetention)

	cmd := UpdateGateCommand{
		ID:        repo.gate.ID.String(),
		Retention: strPtr("1h"),
	}

	gate, err := handler.Handle(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, gate)
	assert.ErrorIs(t, err, traffictesting.ErrRetentionBelowMinimum)
}

func TestUpdateGateHandler_invalid_retention_format(t *testing.T) {
	repo := newMockGateRepo()
	handler := NewUpdateGateHandler(repo, testGlobalRetention)

	cmd := UpdateGateCommand{
		ID:        repo.gate.ID.String(),
		Retention: strPtr("not-a-duration"),
	}

	gate, err := handler.Handle(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, gate)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidRetention)
}

func TestUpdateGateHandler_empty_retention_resets_to_global(t *testing.T) {
	repo := newMockGateRepo()
	// Seed the gate with a custom retention so we can verify it is cleared.
	repo.gate.Retention = mustRetention(t, "1000h")
	handler := NewUpdateGateHandler(repo, testGlobalRetention)

	cmd := UpdateGateCommand{
		ID:        repo.gate.ID.String(),
		Retention: strPtr(""),
	}

	gate, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	assert.False(t, gate.Retention.IsSet(), "empty string should reset to global retention")
}

func mustRetention(t *testing.T, s string) traffictesting.Retention {
	t.Helper()
	r, err := traffictesting.ParseRetention(s)
	require.NoError(t, err)
	return r
}

func strPtr(s string) *string { return &s }
