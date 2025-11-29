package assistant_test

import (
	"testing"

	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/testutils"
)

func TestMCPToolName(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	tests := []struct {
		name       string
		serverID   string
		toolName   string
		wantResult string
	}{
		{
			name:       "Simple tool name",
			serverID:   "github",
			toolName:   "search_repos",
			wantResult: "github__search_repos",
		},
		{
			name:       "Server with dots",
			serverID:   "github.enterprise",
			toolName:   "search_repos",
			wantResult: "github_enterprise__search_repos",
		},
		{
			name:       "Tool with underscores",
			serverID:   "customer-db",
			toolName:   "create_customer",
			wantResult: "customer-db__create_customer",
		},
		{
			name:       "Complex server with multiple dots",
			serverID:   "com.example.mcp",
			toolName:   "tool_name",
			wantResult: "com_example_mcp__tool_name",
		},
		{
			name:       "Empty server ID",
			serverID:   "",
			toolName:   "tool",
			wantResult: "",
		},
		{
			name:       "Empty tool name",
			serverID:   "server",
			toolName:   "",
			wantResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assistant.MCPToolName(tt.serverID, tt.toolName)
			if result != tt.wantResult {
				t.Errorf("MCPToolName() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestParseMCPToolName(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	tests := []struct {
		name          string
		formattedName string
		wantServerID  string
		wantToolName  string
		wantOK        bool
	}{
		{
			name:          "Valid simple format",
			formattedName: "github__search_repos",
			wantServerID:  "github",
			wantToolName:  "search_repos",
			wantOK:        true,
		},
		{
			name:          "Server with dots restored",
			formattedName: "github_enterprise__search_repos",
			wantServerID:  "github.enterprise",
			wantToolName:  "search_repos",
			wantOK:        true,
		},
		{
			name:          "Complex server ID with multiple dots",
			formattedName: "com_example_mcp_server__tool_name",
			wantServerID:  "com.example.mcp.server",
			wantToolName:  "tool_name",
			wantOK:        true,
		},
		{
			name:          "Tool name with underscores",
			formattedName: "server__create_new_user",
			wantServerID:  "server",
			wantToolName:  "create_new_user",
			wantOK:        true,
		},
		{
			name:          "Server with hyphens",
			formattedName: "mcp-server__tool",
			wantServerID:  "mcp-server",
			wantToolName:  "tool",
			wantOK:        true,
		},
		{
			name:          "Invalid format - no double underscore",
			formattedName: "invalid",
			wantServerID:  "",
			wantToolName:  "",
			wantOK:        false,
		},
		{
			name:          "Invalid format - empty string",
			formattedName: "",
			wantServerID:  "",
			wantToolName:  "",
			wantOK:        false,
		},
		{
			name:          "Invalid format - only double underscore",
			formattedName: "__",
			wantServerID:  "",
			wantToolName:  "",
			wantOK:        false,
		},
		{
			name:          "Invalid format - ends with double underscore",
			formattedName: "server__",
			wantServerID:  "",
			wantToolName:  "",
			wantOK:        false,
		},
		{
			name:          "Invalid format - starts with double underscore",
			formattedName: "__tool",
			wantServerID:  "",
			wantToolName:  "",
			wantOK:        false,
		},
		{
			name:          "Invalid format - multiple double underscores",
			formattedName: "server__middle__tool",
			wantServerID:  "",
			wantToolName:  "",
			wantOK:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverID, toolName, ok := assistant.ParseMCPToolName(tt.formattedName)
			if serverID != tt.wantServerID {
				t.Errorf("ParseMCPToolName() serverID = %v, want %v", serverID, tt.wantServerID)
			}
			if toolName != tt.wantToolName {
				t.Errorf("ParseMCPToolName() toolName = %v, want %v", toolName, tt.wantToolName)
			}
			if ok != tt.wantOK {
				t.Errorf("ParseMCPToolName() ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

func TestMCPToolName_RoundTrip(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	tests := []struct {
		name     string
		serverID string
		toolName string
	}{
		{
			name:     "Simple IDs",
			serverID: "github",
			toolName: "search_repos",
		},
		{
			name:     "Server with dots",
			serverID: "github.enterprise",
			toolName: "search",
		},
		{
			name:     "Complex server ID",
			serverID: "com.example.mcp.server",
			toolName: "tool_name",
		},
		{
			name:     "Server with dashes",
			serverID: "mcp-server-123",
			toolName: "tool_with_underscores",
		},
		{
			name:     "Mixed dots and dashes",
			serverID: "github.enterprise-prod",
			toolName: "api_call",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Format
			formatted := assistant.MCPToolName(tt.serverID, tt.toolName)
			if formatted == "" {
				t.Fatal("MCPToolName() returned empty string")
			}

			// Parse
			serverID, toolName, ok := assistant.ParseMCPToolName(formatted)

			// Verify round-trip
			if !ok {
				t.Fatal("ParseMCPToolName() failed")
			}
			if serverID != tt.serverID {
				t.Errorf("Round-trip failed: serverID = %v, want %v", serverID, tt.serverID)
			}
			if toolName != tt.toolName {
				t.Errorf("Round-trip failed: toolName = %v, want %v", toolName, tt.toolName)
			}

			t.Logf("✓ Round-trip successful: (%s, %s) → %s → (%s, %s)",
				tt.serverID, tt.toolName, formatted, serverID, toolName)
		})
	}
}
