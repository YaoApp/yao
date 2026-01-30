package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultImage(t *testing.T) {
	tests := []struct {
		command  string
		expected string
	}{
		{"claude", "yaoapp/sandbox-claude:latest"},
		{"cursor", "yaoapp/sandbox-cursor:latest"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := DefaultImage(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidCommand(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		{"claude", true},
		{"cursor", true},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := IsValidCommand(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOptionsValidation(t *testing.T) {
	// Test that Options struct can be created with all fields
	opts := &Options{
		Command:       "claude",
		Image:         "yaoapp/sandbox-claude:latest",
		MaxMemory:     "4g",
		MaxCPU:        2.0,
		UserID:        "user123",
		ChatID:        "chat456",
		ConnectorHost: "https://api.example.com",
		ConnectorKey:  "key123",
		Model:         "deepseek-v3",
		Arguments: map[string]interface{}{
			"max_turns":       20,
			"permission_mode": "acceptEdits",
		},
	}

	assert.Equal(t, "claude", opts.Command)
	assert.Equal(t, "user123", opts.UserID)
	assert.Equal(t, "chat456", opts.ChatID)
	assert.Equal(t, 20, opts.Arguments["max_turns"])
}

func TestSandboxConfigParsing(t *testing.T) {
	// Test that SandboxConfig can be used for parsing assistant config
	config := &SandboxConfig{
		Command:   "claude",
		Image:     "custom-image:v1",
		MaxMemory: "8g",
		MaxCPU:    4.0,
		Timeout:   "10m",
		Arguments: map[string]interface{}{
			"permission_mode": "bypassPermissions",
		},
	}

	assert.Equal(t, "claude", config.Command)
	assert.Equal(t, "custom-image:v1", config.Image)
	assert.Equal(t, "8g", config.MaxMemory)
	assert.Equal(t, "10m", config.Timeout)
}
