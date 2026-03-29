package traffictesting_test

import (
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
)

func TestNewGate_creates_gate_with_auto_generated_id(t *testing.T) {
	name, _ := traffictesting.ParseGateName("test-gate")
	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")

	gate, err := traffictesting.NewGate(name, liveURL, shadowURL)

	assert.NoError(t, err)
	assert.False(t, gate.ID.IsZero())
	assert.Equal(t, "test-gate", gate.Name.String())
	assert.Equal(t, liveURL.String(), gate.LiveURL.String())
	assert.Equal(t, shadowURL.String(), gate.ShadowURL.String())
	assert.False(t, gate.CreatedAt.IsZero())
}

func TestNewGate_with_custom_id(t *testing.T) {
	name, _ := traffictesting.ParseGateName("custom-gate")
	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	customID := traffictesting.NewGateID()

	gate, err := traffictesting.NewGate(name, liveURL, shadowURL, traffictesting.WithGateID(customID))

	assert.NoError(t, err)
	assert.Equal(t, customID.String(), gate.ID.String())
}
