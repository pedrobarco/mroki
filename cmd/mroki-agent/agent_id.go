package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const agentIDFile = ".agent_id"

// loadOrGenerateAgentID loads the agent ID from disk, or generates a new one if it doesn't exist.
// The agent ID is stored in a .agent_id file in the current working directory.
// Format: {hostname}-{8-char-uuid}
// Example: web-server-a1b2c3d4
func loadOrGenerateAgentID() (string, error) {
	// Try to read existing agent ID
	data, err := os.ReadFile(agentIDFile)
	if err == nil {
		agentID := strings.TrimSpace(string(data))
		if agentID != "" {
			// Validate it's a proper UUID format
			if _, err := uuid.Parse(agentID); err == nil {
				return agentID, nil
			}
		}
	}

	// Generate new agent ID
	agentID := generateAgentID()

	// Save to disk
	if err := saveAgentID(agentID); err != nil {
		return "", fmt.Errorf("failed to save agent ID: %w", err)
	}

	return agentID, nil
}

// generateAgentID creates a new agent ID with format: {hostname}-{8-char-uuid}
func generateAgentID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Clean hostname: replace invalid characters with hyphens
	hostname = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, hostname)

	// Truncate hostname if too long
	if len(hostname) > 50 {
		hostname = hostname[:50]
	}

	// Generate UUID and create full agent ID
	id := uuid.New()
	return fmt.Sprintf("%s-%s", hostname, id.String())
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
