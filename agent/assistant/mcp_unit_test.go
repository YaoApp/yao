//go:build unit

package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/assistant"
)

func TestMCPToolName_Format(t *testing.T) {
	tests := []struct {
		name     string
		serverID string
		toolName string
		expected string
	}{
		{
			name:     "simple server and tool",
			serverID: "echo",
			toolName: "ping",
			expected: "echo__ping",
		},
		{
			name:     "dotted server ID",
			serverID: "github.enterprise",
			toolName: "search",
			expected: "github_enterprise__search",
		},
		{
			name:     "multiple dots in server ID",
			serverID: "com.example.service",
			toolName: "list",
			expected: "com_example_service__list",
		},
		{
			name:     "hyphenated server ID",
			serverID: "my-server",
			toolName: "run",
			expected: "my-server__run",
		},
		{
			name:     "empty server ID returns empty",
			serverID: "",
			toolName: "ping",
			expected: "",
		},
		{
			name:     "empty tool name returns empty",
			serverID: "echo",
			toolName: "",
			expected: "",
		},
		{
			name:     "both empty returns empty",
			serverID: "",
			toolName: "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := assistant.MCPToolName(tc.serverID, tc.toolName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseMCPToolName_Valid(t *testing.T) {
	tests := []struct {
		name           string
		formatted      string
		expectServerID string
		expectToolName string
	}{
		{
			name:           "simple name",
			formatted:      "echo__ping",
			expectServerID: "echo",
			expectToolName: "ping",
		},
		{
			name:           "dotted server restored",
			formatted:      "github_enterprise__search",
			expectServerID: "github.enterprise",
			expectToolName: "search",
		},
		{
			name:           "multiple underscores restored as dots",
			formatted:      "com_example_service__list",
			expectServerID: "com.example.service",
			expectToolName: "list",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			serverID, toolName, ok := assistant.ParseMCPToolName(tc.formatted)
			assert.True(t, ok, "parse should succeed")
			assert.Equal(t, tc.expectServerID, serverID)
			assert.Equal(t, tc.expectToolName, toolName)
		})
	}
}

func TestParseMCPToolName_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		formatted string
	}{
		{name: "empty string", formatted: ""},
		{name: "no separator", formatted: "echo_ping"},
		{name: "only separator", formatted: "__"},
		{name: "missing tool part", formatted: "echo__"},
		{name: "missing server part", formatted: "__ping"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, ok := assistant.ParseMCPToolName(tc.formatted)
			assert.False(t, ok, "parse should fail for %q", tc.formatted)
		})
	}
}

func TestMCPToolName_RoundTrip(t *testing.T) {
	tests := []struct {
		serverID string
		toolName string
	}{
		{"echo", "ping"},
		{"github.enterprise", "search"},
		{"com.example.service", "list-items"},
	}

	for _, tc := range tests {
		formatted := assistant.MCPToolName(tc.serverID, tc.toolName)
		assert.NotEmpty(t, formatted)

		parsedServer, parsedTool, ok := assistant.ParseMCPToolName(formatted)
		assert.True(t, ok, "roundtrip parse should succeed for %s/%s", tc.serverID, tc.toolName)
		assert.Equal(t, tc.serverID, parsedServer, "server ID roundtrip mismatch")
		assert.Equal(t, tc.toolName, parsedTool, "tool name roundtrip mismatch")
	}
}
