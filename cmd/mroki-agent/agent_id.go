package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedrobarco/mroki/internal/domain/diffing"
)

const agentIDFile = ".agent_id"

// loadOrGenerateAgentID loads the agent ID from disk, or generates a new one if it doesn't exist.
// The agent ID is stored in a .agent_id file in the current working directory.
// Format: {hostname}-{8-hex-chars}
// Example: web-server-a1b2c3d4, api-prod-550e8400
func loadOrGenerateAgentID() (string, error) {
	// Try to read existing agent ID
	data, err := os.ReadFile(agentIDFile)
	if err == nil {
		agentID := strings.TrimSpace(string(data))
		if agentID != "" {
			// Validate it matches the hybrid format
			if _, err := diffing.ParseAgentID(agentID); err == nil {
				return agentID, nil
			}
			// If validation fails, log a warning and regenerate
			fmt.Printf("Warning: existing agent ID %q is invalid, generating new one\n", agentID)
		}
	}

	// Generate new agent ID using hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "agent"
	}

	agentID := diffing.NewAgentIDWithHostname(hostname)

	// Save to disk
	if err := saveAgentID(agentID.String()); err != nil {
		return "", fmt.Errorf("failed to save agent ID: %w", err)
	}

	return agentID.String(), nil
}

// saveAgentID saves the agent ID to the .agent_id file
func saveAgentID(agentID string) error {
	// Get absolute path for better error messages
	absPath, err := filepath.Abs(agentIDFile)
	if err != nil {
		absPath = agentIDFile
	}

	// Write to file with restrictive permissions (0600 = owner read/write only)
	if err := os.WriteFile(agentIDFile, []byte(agentID+"\n"), 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", absPath, err)
	}

	return nil
}
