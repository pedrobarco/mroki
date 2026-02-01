package traffictesting_test

import (
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
)

func TestNewGate_creates_gate_with_auto_generated_id(t *testing.T) {
	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")

	gate, err := traffictesting.NewGate(liveURL, shadowURL)

	assert.NoError(t, err)
	assert.False(t, gate.ID.IsZero())
	assert.Equal(t, liveURL.String(), gate.LiveURL.String())
	assert.Equal(t, shadowURL.String(), gate.ShadowURL.String())
}

func TestNewGate_with_custom_id(t *testing.T) {
	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	customID := traffictesting.NewGateID()

	gate, err := traffictesting.NewGate(liveURL, shadowURL, traffictesting.WithGateID(customID))

	assert.NoError(t, err)
	assert.Equal(t, customID.String(), gate.ID.String())
}
