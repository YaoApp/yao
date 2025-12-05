package assistant_test

import (
	"testing"

	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestBuildRequest_MCP tests MCP tool integration in BuildRequest
func TestBuildRequest_MCP(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.mcptest")
	if err != nil {
		t.Fatalf("Failed to get tests.mcptest assistant: %s", err.Error())
	}

	ctx := newTestContext("chat-test-mcp", "tests.mcptest")

	t.Run("MCPToolsLoaded", func(t *testing.T) {
		inputMessages := []context.Message{{Role: context.RoleUser, Content: "test mcp tools"}}

		// Build LLM request
		_, options, err := agent.BuildRequest(ctx, inputMessages, nil)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Verify that tools are loaded
		if options.Tools == nil {
			t.Fatal("Expected tools to be loaded, got nil")
		}

		if len(options.Tools) == 0 {
			t.Fatal("Expected at least some MCP tools, got empty list")
		}

		// Count MCP tools (should be filtered to only ping and echo)
		mcpToolCount := 0
		var toolNames []string
		for _, toolMap := range options.Tools {
			fn, ok := toolMap["function"].(map[string]interface{})
			if !ok {
				continue
			}
			name, ok := fn["name"].(string)
			if ok {
				toolNames = append(toolNames, name)
				mcpToolCount++
			}
		}

		t.Logf("Found %d MCP tools: %v", mcpToolCount, toolNames)

		// Verify tool count (should be exactly 2: ping and echo)
		if mcpToolCount != 2 {
			t.Errorf("Expected 2 MCP tools (ping, echo), got %d: %v", mcpToolCount, toolNames)
		}

		// Verify specific tools exist
		hasEchoPing := false
		hasEchoEcho := false
		for _, name := range toolNames {
			if name == "echo__ping" {
				hasEchoPing = true
			}
			if name == "echo__echo" {
				hasEchoEcho = true
			}
		}

		if !hasEchoPing {
			t.Error("Expected 'echo__ping' tool to be present")
		}
		if !hasEchoEcho {
			t.Error("Expected 'echo__echo' tool to be present")
		}

		// Verify that 'status' tool is NOT included (filtered out)
		for _, name := range toolNames {
			if name == "echo__status" {
				t.Error("Tool 'echo__status' should be filtered out but was found")
			}
		}

		t.Log("✓ MCP tools loaded and filtered correctly")
	})

	t.Run("MCPSamplesPrompt", func(t *testing.T) {
		inputMessages := []context.Message{{Role: context.RoleUser, Content: "test mcp samples"}}

		// Build LLM request
		finalMessages, _, err := agent.BuildRequest(ctx, inputMessages, nil)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Check if messages contain MCP samples prompt
		// The samples prompt should be added as a system message
		hasMCPSamples := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				if content, ok := msg.Content.(string); ok {
					if len(content) > 50 &&
						(contains(content, "MCP Tool Usage Examples") ||
							contains(content, "echo.ping") ||
							contains(content, "echo.echo")) {
						hasMCPSamples = true
						t.Logf("Found MCP samples prompt (length: %d chars)", len(content))
						break
					}
				}
			}
		}

		// Note: samples may not exist for echo tools, so this is informational
		if hasMCPSamples {
			t.Log("✓ MCP samples prompt included in messages")
		} else {
			t.Log("ℹ No MCP samples prompt found (may not have sample files)")
		}
	})

	t.Run("MCPToolNameFormat", func(t *testing.T) {
		inputMessages := []context.Message{{Role: context.RoleUser, Content: "test tool format"}}

		// Build LLM request
		_, options, err := agent.BuildRequest(ctx, inputMessages, nil)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Verify tool name format: server_id.tool_name
		for _, toolMap := range options.Tools {
			fn, ok := toolMap["function"].(map[string]interface{})
			if !ok {
				continue
			}
			name, ok := fn["name"].(string)
			if ok {
				// Parse tool name
				serverID, toolName, ok := assistant.ParseMCPToolName(name)
				if !ok {
					t.Errorf("Tool name '%s' is not in correct format (server_id.tool_name)", name)
					continue
				}

				// Verify server ID
				if serverID != "echo" {
					t.Errorf("Expected server_id 'echo', got '%s' for tool '%s'", serverID, name)
				}

				// Verify tool name is either ping or echo
				if toolName != "ping" && toolName != "echo" {
					t.Errorf("Expected tool name 'ping' or 'echo', got '%s'", toolName)
				}

				t.Logf("✓ Tool name format correct: %s → (%s, %s)", name, serverID, toolName)
			}
		}
	})

	t.Run("MCPToolSchema", func(t *testing.T) {
		inputMessages := []context.Message{{Role: context.RoleUser, Content: "test tool schema"}}

		// Build LLM request
		_, options, err := agent.BuildRequest(ctx, inputMessages, nil)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Verify tool schema structure
		for _, toolMap := range options.Tools {
			// Verify type field
			if toolType, ok := toolMap["type"].(string); !ok || toolType != "function" {
				t.Errorf("Expected tool type 'function', got: %v", toolMap["type"])
			}

			// Verify function field exists
			fn, ok := toolMap["function"].(map[string]interface{})
			if !ok {
				t.Error("Tool missing 'function' field or wrong type")
				continue
			}

			// Verify required fields
			if _, hasName := fn["name"]; !hasName {
				t.Error("Tool function missing 'name' field")
			}
			if _, hasDesc := fn["description"]; !hasDesc {
				t.Error("Tool function missing 'description' field")
			}
			if _, hasParams := fn["parameters"]; !hasParams {
				t.Error("Tool function missing 'parameters' field")
			}

			t.Logf("✓ Tool schema valid: %v", fn["name"])
		}
	})

	t.Run("MCPHookOverride", func(t *testing.T) {
		// Test that hook can override MCP servers
		// Use tests.mcptest-hook which has a create hook that returns only ["ping"]
		hookAgent, err := assistant.Get("tests.mcptest-hook")
		if err != nil {
			t.Fatalf("Failed to get tests.mcptest-hook assistant: %s", err.Error())
		}

		hookCtx := newTestContext("chat-test-mcp-hook", "tests.mcptest-hook")
		inputMessages := []context.Message{{Role: context.RoleUser, Content: "test hook override"}}

		// Call create hook to get createResponse
		var createResponse *context.HookCreateResponse
		if hookAgent.HookScript != nil {
			createResponse, _, err = hookAgent.HookScript.Create(hookCtx, inputMessages, &context.Options{})
			if err != nil {
				t.Fatalf("Failed to call create hook: %s", err.Error())
			}

			t.Logf("Create hook response: %+v", createResponse)
			if createResponse != nil && len(createResponse.MCPServers) > 0 {
				t.Logf("Hook MCP servers: %+v", createResponse.MCPServers)
			}
		} else {
			t.Fatal("Expected hookAgent to have Script/hook configured")
		}

		// Build LLM request with create hook response
		_, options, err := hookAgent.BuildRequest(hookCtx, inputMessages, createResponse)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Verify that tools are loaded
		if options.Tools == nil {
			t.Fatal("Expected tools to be loaded, got nil")
		}

		// Count MCP tools
		mcpToolCount := 0
		var toolNames []string
		for _, toolMap := range options.Tools {
			fn, ok := toolMap["function"].(map[string]interface{})
			if !ok {
				continue
			}
			name, ok := fn["name"].(string)
			if ok {
				toolNames = append(toolNames, name)
				mcpToolCount++
			}
		}

		t.Logf("Found %d MCP tools after hook override: %v", mcpToolCount, toolNames)

		// Verify tool count (hook should override to only 1: ping)
		if mcpToolCount != 1 {
			t.Errorf("Expected 1 MCP tool (ping only), got %d: %v", mcpToolCount, toolNames)
		}

		// Verify only ping tool exists
		hasEchoPing := false
		hasEchoEcho := false
		for _, name := range toolNames {
			if name == "echo__ping" {
				hasEchoPing = true
			}
			if name == "echo__echo" {
				hasEchoEcho = true
			}
		}

		if !hasEchoPing {
			t.Error("Expected 'echo__ping' tool to be present")
		}
		if hasEchoEcho {
			t.Error("Tool 'echo__echo' should be filtered out by hook override but was found")
		}

		t.Log("✓ Hook successfully overrode MCP servers configuration")
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
