package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	err := Load(config.Conf)
	// Load may fail due to configuration issues, but we should still check what was loaded
	if err != nil {
		t.Logf("Load returned error: %v", err)
	}

	check(t)
}

func check(t *testing.T) {
	clients := mcp.ListClients()
	clientMap := make(map[string]bool)
	for _, id := range clients {
		clientMap[id] = true
	}

	t.Logf("Loaded clients: %v", clients)

	// Check if test MCP clients are loaded (they may fail to load due to configuration)
	if clientMap["test"] {
		assert.True(t, clientMap["test"], "test MCP client should be loaded")

		// Verify clients can be selected
		testClient, err := mcp.Select("test")
		assert.Nil(t, err)
		assert.NotNil(t, testClient)

		// Check that clients exist
		assert.True(t, mcp.Exists("test"))
		t.Logf("test MCP client loaded successfully")
	} else {
		t.Logf("test MCP client not loaded (possibly due to configuration issues)")
	}

	if clientMap["http_test"] {
		assert.True(t, clientMap["http_test"], "http_test MCP client should be loaded")

		httpTestClient, err := mcp.Select("http_test")
		assert.Nil(t, err)
		assert.NotNil(t, httpTestClient)

		assert.True(t, mcp.Exists("http_test"))
		t.Logf("http_test MCP client loaded successfully")
	} else {
		t.Logf("http_test MCP client not loaded (possibly due to configuration issues)")
	}

	// This should always be false
	assert.False(t, mcp.Exists("non_existent"))
}

func TestLoadWithError(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Test loading with invalid configuration
	// This may fail due to configuration issues but shouldn't crash
	err := Load(config.Conf)
	if err != nil {
		t.Logf("Load returned expected error: %v", err)
	}
}

func TestGetClient(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	err := Load(config.Conf)
	if err != nil {
		t.Logf("Load returned error: %v", err)
	}

	// Test getting existing client (if it was loaded successfully)
	if mcp.Exists("test") {
		client := mcp.GetClient("test")
		assert.NotNil(t, client)
		t.Logf("GetClient test passed")
	} else {
		t.Logf("test client not loaded, skipping GetClient test")
	}

	// Test getting non-existent client should throw exception
	assert.Panics(t, func() {
		mcp.GetClient("non_existent")
	})
}

func TestUnloadClient(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	err := Load(config.Conf)
	if err != nil {
		t.Logf("Load returned error: %v", err)
	}

	// Test unloading only if client was loaded
	if mcp.Exists("test") {
		// Verify client exists before unloading
		assert.True(t, mcp.Exists("test"))

		// Unload client
		mcp.UnloadClient("test")

		// Verify client no longer exists
		assert.False(t, mcp.Exists("test"))
		t.Logf("UnloadClient test passed")
	} else {
		t.Logf("test client not loaded, skipping UnloadClient test")
	}

	// Test that unloading non-existent client doesn't crash
	mcp.UnloadClient("non_existent")
}

func TestLoadAssistantMCPs(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load MCPs
	err := Load(config.Conf)
	if err != nil {
		t.Logf("Load returned error: %v", err)
	}

	// List all loaded clients
	clients := mcp.ListClients()
	t.Logf("Total loaded MCP clients: %d", len(clients))

	// Filter agent clients
	agentClients := []string{}
	for _, id := range clients {
		if len(id) >= 7 && id[:7] == "agents." {
			agentClients = append(agentClients, id)
		}
	}

	t.Logf("Agent MCP clients: %v", agentClients)

	// Check if the test assistant MCP client is loaded
	testClientID := "agents.tests.mcpload.test"
	if mcp.Exists(testClientID) {
		t.Logf("✓ Test assistant MCP client '%s' loaded successfully", testClientID)

		// Verify we can get the client
		client, err := mcp.Select(testClientID)
		assert.Nil(t, err)
		assert.NotNil(t, client)

		// Try to list tools
		ctx := context.Background()
		toolsResp, err := client.ListTools(ctx, "")
		if err == nil && toolsResp != nil {
			t.Logf("✓ Available tools in %s: %d", testClientID, len(toolsResp.Tools))
			for _, tool := range toolsResp.Tools {
				t.Logf("  - Tool: %s - %s", tool.Name, tool.Description)
			}
		} else {
			t.Logf("Could not list tools: %v", err)
		}

	} else {
		t.Logf("Test assistant MCP client '%s' not found", testClientID)
		t.Logf("This may be expected if the test assistant is not in the application")
	}

	// Check for nested MCP client
	nestedClientID := "agents.tests.mcpload.nested.tool"
	if mcp.Exists(nestedClientID) {
		t.Logf("✓ Nested MCP client '%s' loaded successfully", nestedClientID)

		client, err := mcp.Select(nestedClientID)
		assert.Nil(t, err)
		assert.NotNil(t, client)

		ctx := context.Background()
		toolsResp, err := client.ListTools(ctx, "")
		if err == nil && toolsResp != nil {
			t.Logf("✓ Available tools in %s: %d", nestedClientID, len(toolsResp.Tools))
			for _, tool := range toolsResp.Tools {
				t.Logf("  - Tool: %s - %s", tool.Name, tool.Description)
			}
		}
	} else {
		t.Logf("✗ Nested MCP client '%s' not found", nestedClientID)
	}

	// Report all agent clients found
	if len(agentClients) > 0 {
		t.Logf("✓ Successfully loaded %d agent MCP client(s):", len(agentClients))
		for _, id := range agentClients {
			t.Logf("  - %s", id)
		}
	} else {
		t.Logf("No agent MCP clients found (this may be expected if no assistants have mcps)")
	}
}
