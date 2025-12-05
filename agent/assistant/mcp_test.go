package assistant_test

import (
	"context"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/mcp"
	mcpTypes "github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
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

// TestMCPToolContextPassing tests that agent context is correctly passed to MCP tools
func TestMCPToolContextPassing(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Get the echo MCP client
	client, err := mcp.Select("echo")
	assert.NoError(t, err, "Failed to select echo MCP client")
	assert.NotNil(t, client, "MCP client should not be nil")

	// Create a test agent context
	authorized := &types.AuthorizedInfo{
		UserID:   "test-user-123",
		TenantID: "test-tenant-456",
	}
	ctx := agentContext.New(context.Background(), authorized, "test-chat-789")
	ctx.AssistantID = "test-assistant-mcptest"
	ctx.Locale = "en"
	ctx.Theme = "dark"

	// Call the echo tool with context
	args := map[string]interface{}{
		"message": "test message from context test",
	}

	// Call the tool - the agent context will be passed as extra parameter
	result, err := client.CallTool(ctx.Context, "echo", args, ctx)
	assert.NoError(t, err, "CallTool should not return error")
	assert.NotNil(t, result, "Result should not be nil")
	assert.False(t, result.IsError, "Result should not be an error")
	assert.Greater(t, len(result.Content), 0, "Result should have content")

	// Parse the result content
	var echoResult map[string]interface{}
	err = jsoniter.Unmarshal([]byte(result.Content[0].Text), &echoResult)
	assert.NoError(t, err, "Failed to parse result content")

	t.Logf("Echo result: %+v", echoResult)

	// Verify the context was received
	contextData, ok := echoResult["context"].(map[string]interface{})
	assert.True(t, ok, "Result should contain context field")
	assert.NotNil(t, contextData, "Context data should not be nil")

	// Verify context has_context flag
	hasContext, ok := contextData["has_context"].(bool)
	assert.True(t, ok, "Context should have has_context field")
	assert.True(t, hasContext, "Context should indicate it has context")

	// Verify chat_id and assistant_id have values (main verification)
	chatID, ok := contextData["chat_id"].(string)
	assert.True(t, ok, "Context should have chat_id field")
	assert.NotEmpty(t, chatID, "chat_id should have a value")
	assert.Equal(t, "test-chat-789", chatID, "chat_id should match")

	assistantID, ok := contextData["assistant_id"].(string)
	assert.True(t, ok, "Context should have assistant_id field")
	assert.NotEmpty(t, assistantID, "assistant_id should have a value")
	assert.Equal(t, "test-assistant-mcptest", assistantID, "assistant_id should match")

	// Verify authorized information
	authorizedData, ok := contextData["authorized"].(map[string]interface{})
	assert.True(t, ok, "Context should have authorized field")
	assert.NotNil(t, authorizedData, "Authorized data should not be nil")

	userID, ok := authorizedData["user_id"].(string)
	assert.True(t, ok, "Authorized should have user_id field")
	assert.Equal(t, "test-user-123", userID, "User ID should match")

	tenantID, ok := authorizedData["tenant_id"].(string)
	assert.True(t, ok, "Authorized should have tenant_id field")
	assert.Equal(t, "test-tenant-456", tenantID, "Tenant ID should match")

	t.Logf("✓ Context successfully passed to MCP tool")
	t.Logf("  - ChatID: %s", chatID)
	t.Logf("  - AssistantID: %s", assistantID)
	t.Logf("  - UserID: %s", userID)
	t.Logf("  - TenantID: %s", tenantID)
}

// TestMCPToolContextPassingParallel tests that agent context is correctly passed in parallel calls
func TestMCPToolContextPassingParallel(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Get the echo MCP client
	client, err := mcp.Select("echo")
	assert.NoError(t, err, "Failed to select echo MCP client")
	assert.NotNil(t, client, "MCP client should not be nil")

	// Create a test agent context
	authorized := &types.AuthorizedInfo{
		UserID:   "parallel-user-123",
		TenantID: "parallel-tenant-456",
	}
	ctx := agentContext.New(context.Background(), authorized, "parallel-chat-789")
	ctx.AssistantID = "test-assistant-parallel"
	ctx.Locale = "zh-CN"

	// Call multiple echo tools in parallel
	toolCalls := []mcpTypes.ToolCall{
		{
			Name: "echo",
			Arguments: map[string]interface{}{
				"message": "parallel message 1",
			},
		},
		{
			Name: "echo",
			Arguments: map[string]interface{}{
				"message": "parallel message 2",
			},
		},
	}

	// Call tools in parallel - the agent context will be passed as extra parameter
	results, err := client.CallToolsParallel(ctx.Context, toolCalls, ctx)
	assert.NoError(t, err, "CallToolsParallel should not return error")
	assert.NotNil(t, results, "Results should not be nil")
	assert.Equal(t, 2, len(results.Results), "Should have 2 results")

	// Verify both results received the context
	for i, result := range results.Results {
		assert.False(t, result.IsError, "Result %d should not be an error", i)
		assert.Greater(t, len(result.Content), 0, "Result %d should have content", i)

		// Parse the result content
		var echoResult map[string]interface{}
		err = jsoniter.Unmarshal([]byte(result.Content[0].Text), &echoResult)
		assert.NoError(t, err, "Failed to parse result %d content", i)

		// Verify the context was received
		contextData, ok := echoResult["context"].(map[string]interface{})
		assert.True(t, ok, "Result %d should contain context field", i)
		assert.NotNil(t, contextData, "Context data %d should not be nil", i)

		hasContext, ok := contextData["has_context"].(bool)
		assert.True(t, ok, "Context %d should have has_context field", i)
		assert.True(t, hasContext, "Context %d should indicate it has context", i)

		// Verify chat_id in parallel call
		chatID, ok := contextData["chat_id"].(string)
		assert.True(t, ok, "Context %d should have chat_id field", i)
		assert.Equal(t, "parallel-chat-789", chatID, "Chat ID in result %d should match", i)

		// Verify authorized information in parallel call
		authorizedData, ok := contextData["authorized"].(map[string]interface{})
		assert.True(t, ok, "Context %d should have authorized field", i)
		if userID, ok := authorizedData["user_id"].(string); ok {
			assert.Equal(t, "parallel-user-123", userID, "User ID in result %d should match", i)
		}

		t.Logf("✓ Result %d successfully received context", i)
	}

	t.Log("✓ Context successfully passed to all parallel MCP tool calls")
}
