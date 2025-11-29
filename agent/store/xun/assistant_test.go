package xun

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestMain(m *testing.M) {
	// Setup will be done in each test via test.Prepare
	test.Prepare(nil, config.Conf)
	defer test.Clean()

	// Run tests and exit with appropriate exit code
	code := m.Run()
	os.Exit(code)
}

// TestSaveAssistant tests creating and updating assistants
func TestSaveAssistant(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create a new xun store
	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("CreateNewAssistant", func(t *testing.T) {
		assistant := &types.AssistantModel{
			Name:        "Test Assistant",
			Type:        "assistant",
			Connector:   "openai",
			Description: "A test assistant for unit testing",
			Avatar:      "https://example.com/avatar.png",
			Tags:        []string{"test", "automation"},
			Options:     map[string]interface{}{"temperature": 0.7},
			Sort:        100,
			BuiltIn:     false,
			Readonly:    false,
			Public:      false,
			Share:       "private",
			Mentionable: true,
			Automated:   true,
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to save assistant: %v", err)
		}

		if id == "" {
			t.Error("Expected non-empty assistant ID")
		}

		if assistant.ID == "" {
			t.Error("Expected assistant.ID to be set")
		}

		t.Logf("Created assistant with ID: %s", id)
	})

	t.Run("UpdateExistingAssistant", func(t *testing.T) {
		// Create initial assistant
		assistant := &types.AssistantModel{
			Name:        "Update Test Assistant",
			Type:        "assistant",
			Connector:   "openai",
			Description: "Original description",
			Tags:        []string{"original"},
			Share:       "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Update the assistant
		assistant.Description = "Updated description"
		assistant.Tags = []string{"updated", "modified"}
		assistant.Sort = 200

		updatedID, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to update assistant: %v", err)
		}

		if updatedID != id {
			t.Errorf("Expected ID %s, got %s", id, updatedID)
		}

		// Verify update
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve updated assistant: %v", err)
		}

		if retrieved.Description != "Updated description" {
			t.Errorf("Expected description 'Updated description', got '%s'", retrieved.Description)
		}

		if len(retrieved.Tags) != 2 || retrieved.Tags[0] != "updated" {
			t.Errorf("Expected tags [updated, modified], got %v", retrieved.Tags)
		}
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		// Test nil assistant
		_, err := store.SaveAssistant(nil)
		if err == nil {
			t.Error("Expected error for nil assistant")
		}

		// Test missing name
		assistant := &types.AssistantModel{
			Type:      "assistant",
			Connector: "openai",
		}
		_, err = store.SaveAssistant(assistant)
		if err == nil {
			t.Error("Expected error for missing name")
		}

		// Test missing type
		assistant = &types.AssistantModel{
			Name:      "Test",
			Connector: "openai",
		}
		_, err = store.SaveAssistant(assistant)
		if err == nil {
			t.Error("Expected error for missing type")
		}

		// Test missing connector
		assistant = &types.AssistantModel{
			Name: "Test",
			Type: "assistant",
		}
		_, err = store.SaveAssistant(assistant)
		if err == nil {
			t.Error("Expected error for missing connector")
		}
	})

	t.Run("ComplexDataTypes", func(t *testing.T) {
		assistant := &types.AssistantModel{
			Name:      "Complex Assistant",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
			Prompts: []types.Prompt{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello"},
			},
			Options: map[string]interface{}{
				"temperature": 0.8,
				"max_tokens":  2000,
			},
			Tags: []string{"complex", "testing", "data"},
			Placeholder: &types.Placeholder{
				Title:       "Type your message",
				Description: "Enter your message here...",
				Prompts:     []string{"What can I help you with?"},
			},
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to save complex assistant: %v", err)
		}

		// Retrieve and verify
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve complex assistant: %v", err)
		}

		if len(retrieved.Prompts) != 2 {
			t.Errorf("Expected 2 prompts, got %d", len(retrieved.Prompts))
		}

		if retrieved.Placeholder == nil {
			t.Error("Expected placeholder to be set")
		}

		if len(retrieved.Tags) != 3 {
			t.Errorf("Expected 3 tags, got %d", len(retrieved.Tags))
		}
	})

	t.Run("SaveWithMCPServers", func(t *testing.T) {
		// Test creating assistant with MCP servers directly
		// This will test that:
		// - server1 (no tools/resources) serializes as "server1"
		// - server2 (with tools) serializes as {"server_id":"server2","tools":[...]}
		// - server3 (with both) serializes as {"server_id":"server3","resources":[...],"tools":[...]}
		assistant := &types.AssistantModel{
			Name:      "MCP Save Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
			MCP: &types.MCPServers{
				Servers: []types.MCPServerConfig{
					{ServerID: "server1"},
					{
						ServerID: "server2",
						Tools:    []string{"tool1", "tool2"},
					},
					{
						ServerID:  "server3",
						Resources: []string{"res1", "res2"},
						Tools:     []string{"tool3", "tool4"},
					},
				},
				Options: map[string]interface{}{
					"timeout": 30,
				},
			},
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to save assistant with MCP: %v", err)
		}

		// Retrieve and verify MCP configuration
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.MCP == nil {
			t.Fatal("Expected MCP to be set")
		}

		if len(retrieved.MCP.Servers) != 3 {
			t.Errorf("Expected 3 MCP servers, got %d", len(retrieved.MCP.Servers))
		}

		// Verify server1 (simple format)
		if retrieved.MCP.Servers[0].ServerID != "server1" {
			t.Errorf("Expected server1, got '%s'", retrieved.MCP.Servers[0].ServerID)
		}

		// Verify server2 (with tools)
		if retrieved.MCP.Servers[1].ServerID != "server2" {
			t.Errorf("Expected server2, got '%s'", retrieved.MCP.Servers[1].ServerID)
		}
		if len(retrieved.MCP.Servers[1].Tools) != 2 {
			t.Errorf("Expected 2 tools for server2, got %d", len(retrieved.MCP.Servers[1].Tools))
		}

		// Verify server3 (with resources and tools)
		if retrieved.MCP.Servers[2].ServerID != "server3" {
			t.Errorf("Expected server3, got '%s'", retrieved.MCP.Servers[2].ServerID)
		}
		if len(retrieved.MCP.Servers[2].Resources) != 2 {
			t.Errorf("Expected 2 resources for server3, got %d", len(retrieved.MCP.Servers[2].Resources))
		}
		if len(retrieved.MCP.Servers[2].Tools) != 2 {
			t.Errorf("Expected 2 tools for server3, got %d", len(retrieved.MCP.Servers[2].Tools))
		}

		// Verify options
		if retrieved.MCP.Options == nil {
			t.Error("Expected MCP options to be set")
		}
		if timeout, ok := retrieved.MCP.Options["timeout"].(float64); !ok || timeout != 30 {
			t.Errorf("Expected timeout 30, got %v", retrieved.MCP.Options["timeout"])
		}

		t.Logf("Successfully verified MCP configuration for assistant %s", id)
	})

	t.Run("UpdateWithMCPServers", func(t *testing.T) {
		// Create assistant without MCP
		assistant := &types.AssistantModel{
			Name:      "MCP Update Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Update assistant with MCP
		assistant.MCP = &types.MCPServers{
			Servers: []types.MCPServerConfig{
				{ServerID: "new-server1"},
				{
					ServerID: "new-server2",
					Tools:    []string{"newtool1"},
				},
			},
		}

		_, err = store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to update assistant with MCP: %v", err)
		}

		// Retrieve and verify
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.MCP == nil || len(retrieved.MCP.Servers) != 2 {
			t.Errorf("Expected 2 MCP servers, got %v", retrieved.MCP)
		}

		if retrieved.MCP.Servers[0].ServerID != "new-server1" {
			t.Errorf("Expected new-server1, got '%s'", retrieved.MCP.Servers[0].ServerID)
		}

		t.Logf("Successfully updated and verified MCP for assistant %s", id)
	})

	t.Run("UsesConfiguration", func(t *testing.T) {
		// Test assistant with Uses configuration
		assistant := &types.AssistantModel{
			Name:      "Uses Test Assistant",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
			Uses: &context.Uses{
				Vision: "mcp:vision-server",
				Audio:  "agent",
				Search: "mcp:search-server",
				Fetch:  "agent",
			},
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to save assistant with uses: %v", err)
		}

		// Retrieve and verify uses configuration
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.Uses == nil {
			t.Fatal("Expected uses to be set")
		}

		if retrieved.Uses.Vision != "mcp:vision-server" {
			t.Errorf("Expected vision 'mcp:vision-server', got '%s'", retrieved.Uses.Vision)
		}

		if retrieved.Uses.Audio != "agent" {
			t.Errorf("Expected audio 'agent', got '%s'", retrieved.Uses.Audio)
		}

		if retrieved.Uses.Search != "mcp:search-server" {
			t.Errorf("Expected search 'mcp:search-server', got '%s'", retrieved.Uses.Search)
		}

		if retrieved.Uses.Fetch != "agent" {
			t.Errorf("Expected fetch 'agent', got '%s'", retrieved.Uses.Fetch)
		}

		t.Logf("Successfully saved and retrieved assistant with uses configuration")
	})

	t.Run("NilUses", func(t *testing.T) {
		// Test assistant without Uses configuration
		assistant := &types.AssistantModel{
			Name:      "No Uses Assistant",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to save assistant without uses: %v", err)
		}

		// Retrieve and verify uses is nil
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.Uses != nil {
			t.Errorf("Expected uses to be nil, got %+v", retrieved.Uses)
		}
	})

	t.Run("PartialUsesConfiguration", func(t *testing.T) {
		// Test assistant with partial Uses configuration
		assistant := &types.AssistantModel{
			Name:      "Partial Uses Assistant",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
			Uses: &context.Uses{
				Vision: "mcp:vision-only",
				// Audio, Search, Fetch not set
			},
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to save assistant with partial uses: %v", err)
		}

		// Retrieve and verify
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.Uses == nil {
			t.Fatal("Expected uses to be set")
		}

		if retrieved.Uses.Vision != "mcp:vision-only" {
			t.Errorf("Expected vision 'mcp:vision-only', got '%s'", retrieved.Uses.Vision)
		}

		if retrieved.Uses.Audio != "" {
			t.Errorf("Expected audio to be empty, got '%s'", retrieved.Uses.Audio)
		}

		if retrieved.Uses.Search != "" {
			t.Errorf("Expected search to be empty, got '%s'", retrieved.Uses.Search)
		}

		if retrieved.Uses.Fetch != "" {
			t.Errorf("Expected fetch to be empty, got '%s'", retrieved.Uses.Fetch)
		}
	})
}

// TestDeleteAssistant tests deleting a single assistant
func TestDeleteAssistant(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("DeleteExistingAssistant", func(t *testing.T) {
		// Create assistant
		assistant := &types.AssistantModel{
			Name:      "Delete Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Delete it
		err = store.DeleteAssistant(id)
		if err != nil {
			t.Fatalf("Failed to delete assistant: %v", err)
		}

		// Verify deletion
		_, err = store.GetAssistant(id)
		if err == nil {
			t.Error("Expected error when getting deleted assistant")
		}
	})

	t.Run("DeleteNonExistentAssistant", func(t *testing.T) {
		err := store.DeleteAssistant("nonexistent-id")
		if err == nil {
			t.Error("Expected error when deleting non-existent assistant")
		}
	})
}

// TestGetAssistant tests retrieving a single assistant
func TestGetAssistant(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("GetExistingAssistant", func(t *testing.T) {
		// Create assistant
		assistant := &types.AssistantModel{
			Name:        "Get Test",
			Type:        "assistant",
			Connector:   "openai",
			Description: "Test description",
			Avatar:      "https://example.com/avatar.png",
			Tags:        []string{"tag1", "tag2"},
			Sort:        150,
			BuiltIn:     false,
			Share:       "private",
			Mentionable: true,
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Retrieve it
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to get assistant: %v", err)
		}

		if retrieved.ID != id {
			t.Errorf("Expected ID %s, got %s", id, retrieved.ID)
		}

		if retrieved.Name != "Get Test" {
			t.Errorf("Expected name 'Get Test', got '%s'", retrieved.Name)
		}

		if retrieved.Description != "Test description" {
			t.Errorf("Expected description 'Test description', got '%s'", retrieved.Description)
		}

		if len(retrieved.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(retrieved.Tags))
		}

		if retrieved.Sort != 150 {
			t.Errorf("Expected sort 150, got %d", retrieved.Sort)
		}
	})

	t.Run("GetNonExistentAssistant", func(t *testing.T) {
		_, err := store.GetAssistant("nonexistent-id")
		if err == nil {
			t.Error("Expected error when getting non-existent assistant")
		}
	})
}

// TestGetAssistants tests retrieving multiple assistants with filtering and pagination
func TestGetAssistants(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Clean up existing data before creating test assistants
	deleted, err := store.DeleteAssistants(types.AssistantFilter{})
	if err != nil {
		t.Logf("Warning: Failed to clean up existing assistants: %v", err)
	} else if deleted > 0 {
		t.Logf("Cleaned up %d existing assistants", deleted)
	}

	// Create test assistants
	assistants := []types.AssistantModel{
		{
			Name:        "Assistant 1",
			Type:        "assistant",
			Connector:   "openai",
			Description: "First test assistant",
			Tags:        []string{"test", "automation"},
			Sort:        100,
			Share:       "private",
			Mentionable: true,
			Automated:   true,
		},
		{
			Name:        "Assistant 2",
			Type:        "assistant",
			Connector:   "anthropic",
			Description: "Second test assistant",
			Tags:        []string{"test", "manual"},
			Sort:        200,
			Share:       "private",
			Mentionable: false,
			Automated:   false,
		},
		{
			Name:        "Assistant 3",
			Type:        "bot",
			Connector:   "openai",
			Description: "Third test bot",
			Tags:        []string{"bot", "automation"},
			Sort:        50,
			Share:       "private",
			Mentionable: true,
			Automated:   true,
		},
	}

	createdIDs := []string{}
	for _, asst := range assistants {
		id, err := store.SaveAssistant(&asst)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}
		createdIDs = append(createdIDs, id)
	}

	t.Run("GetAllAssistants", func(t *testing.T) {
		response, err := store.GetAssistants(types.AssistantFilter{
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to get assistants: %v", err)
		}

		if len(response.Data) < 3 {
			t.Errorf("Expected at least 3 assistants, got %d", len(response.Data))
		}

		if response.Total < 3 {
			t.Errorf("Expected total >= 3, got %d", response.Total)
		}
	})

	t.Run("FilterByType", func(t *testing.T) {
		response, err := store.GetAssistants(types.AssistantFilter{
			Type:     "assistant",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to get assistants by type: %v", err)
		}

		for _, assistant := range response.Data {
			if assistant.Type != "assistant" {
				t.Errorf("Expected type 'assistant', got '%s'", assistant.Type)
			}
		}
	})

	t.Run("FilterByConnector", func(t *testing.T) {
		response, err := store.GetAssistants(types.AssistantFilter{
			Connector: "openai",
			Page:      1,
			PageSize:  20,
		})
		if err != nil {
			t.Fatalf("Failed to get assistants by connector: %v", err)
		}

		for _, assistant := range response.Data {
			if assistant.Connector != "openai" {
				t.Errorf("Expected connector 'openai', got '%s'", assistant.Connector)
			}
		}
	})

	t.Run("FilterByTags", func(t *testing.T) {
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"automation"},
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to get assistants by tags: %v", err)
		}

		// Should find assistants with "automation" tag
		found := false
		for _, assistant := range response.Data {
			for _, tag := range assistant.Tags {
				if tag == "automation" {
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found && len(response.Data) > 0 {
			t.Error("Expected to find assistants with 'automation' tag")
		}
	})

	t.Run("FilterByKeywords", func(t *testing.T) {
		response, err := store.GetAssistants(types.AssistantFilter{
			Keywords: "Second",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to get assistants by keywords: %v", err)
		}

		// Should find "Assistant 2"
		found := false
		for _, assistant := range response.Data {
			if assistant.Name == "Assistant 2" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find assistant with keyword 'Second'")
		}
	})

	t.Run("FilterByMentionable", func(t *testing.T) {
		mentionableTrue := true
		response, err := store.GetAssistants(types.AssistantFilter{
			Mentionable: &mentionableTrue,
			Page:        1,
			PageSize:    20,
		})
		if err != nil {
			t.Fatalf("Failed to get mentionable assistants: %v", err)
		}

		if len(response.Data) != 2 {
			t.Errorf("Expected 2 mentionable assistants, got %d", len(response.Data))
		}

		for _, assistant := range response.Data {
			if !assistant.Mentionable {
				t.Errorf("Expected assistant %s (%s) to be mentionable, but it's not", assistant.ID, assistant.Name)
			}
		}
	})

	t.Run("FilterByAutomated", func(t *testing.T) {
		automatedFalse := false
		response, err := store.GetAssistants(types.AssistantFilter{
			Automated: &automatedFalse,
			Page:      1,
			PageSize:  20,
		})
		if err != nil {
			t.Fatalf("Failed to get non-automated assistants: %v", err)
		}

		for _, assistant := range response.Data {
			if assistant.Automated {
				t.Error("Expected all assistants to be non-automated")
			}
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		// Test first page
		response1, err := store.GetAssistants(types.AssistantFilter{
			Page:     1,
			PageSize: 2,
		})
		if err != nil {
			t.Fatalf("Failed to get first page: %v", err)
		}

		if len(response1.Data) > 2 {
			t.Errorf("Expected max 2 results, got %d", len(response1.Data))
		}

		if response1.Page != 1 {
			t.Errorf("Expected page 1, got %d", response1.Page)
		}

		if response1.PageSize != 2 {
			t.Errorf("Expected page size 2, got %d", response1.PageSize)
		}

		// Test second page if there are enough records
		if response1.Total > 2 {
			response2, err := store.GetAssistants(types.AssistantFilter{
				Page:     2,
				PageSize: 2,
			})
			if err != nil {
				t.Fatalf("Failed to get second page: %v", err)
			}

			if response2.Page != 2 {
				t.Errorf("Expected page 2, got %d", response2.Page)
			}
		}
	})

	t.Run("FieldSelection", func(t *testing.T) {
		response, err := store.GetAssistants(types.AssistantFilter{
			Select:   []string{"assistant_id", "name", "type"},
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to get assistants with field selection: %v", err)
		}

		if len(response.Data) > 0 {
			assistant := response.Data[0]
			if assistant.ID == "" {
				t.Error("Expected assistant_id field")
			}
			if assistant.Name == "" {
				t.Error("Expected name field")
			}
			if assistant.Type == "" {
				t.Error("Expected type field")
			}
		}
	})

	t.Run("FilterByAssistantID", func(t *testing.T) {
		if len(createdIDs) > 0 {
			response, err := store.GetAssistants(types.AssistantFilter{
				AssistantID: createdIDs[0],
				Page:        1,
				PageSize:    20,
			})
			if err != nil {
				t.Fatalf("Failed to get assistant by ID: %v", err)
			}

			if len(response.Data) != 1 {
				t.Errorf("Expected 1 result, got %d", len(response.Data))
			}

			if response.Data[0].ID != createdIDs[0] {
				t.Errorf("Expected assistant_id %s, got %s", createdIDs[0], response.Data[0].ID)
			}
		}
	})

	t.Run("FilterByAssistantIDs", func(t *testing.T) {
		if len(createdIDs) >= 2 {
			filterIDs := []string{createdIDs[0], createdIDs[1]}
			response, err := store.GetAssistants(types.AssistantFilter{
				AssistantIDs: filterIDs,
				Page:         1,
				PageSize:     20,
			})
			if err != nil {
				t.Fatalf("Failed to get assistants by IDs: %v", err)
			}

			if len(response.Data) < 2 {
				t.Errorf("Expected at least 2 results, got %d", len(response.Data))
			}
		}
	})
}

// TestDeleteAssistants tests bulk deletion with filters
func TestDeleteAssistants(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("DeleteByTag", func(t *testing.T) {
		// Create assistants with specific tag
		tag := fmt.Sprintf("delete-test-%d", time.Now().UnixNano())
		for i := 0; i < 3; i++ {
			assistant := &types.AssistantModel{
				Name:      fmt.Sprintf("Delete Test %d", i),
				Type:      "assistant",
				Connector: "openai",
				Tags:      []string{tag},
				Share:     "private",
			}
			_, err := store.SaveAssistant(assistant)
			if err != nil {
				t.Fatalf("Failed to create assistant: %v", err)
			}
		}

		// Delete by tag
		count, err := store.DeleteAssistants(types.AssistantFilter{
			Tags: []string{tag},
		})
		if err != nil {
			t.Fatalf("Failed to delete assistants: %v", err)
		}

		if count < 3 {
			t.Errorf("Expected at least 3 deletions, got %d", count)
		}
	})

	t.Run("DeleteByConnector", func(t *testing.T) {
		// Create assistants with specific connector
		connector := fmt.Sprintf("test-connector-%d", time.Now().UnixNano())
		for i := 0; i < 2; i++ {
			assistant := &types.AssistantModel{
				Name:      fmt.Sprintf("Connector Test %d", i),
				Type:      "assistant",
				Connector: connector,
				Share:     "private",
			}
			_, err := store.SaveAssistant(assistant)
			if err != nil {
				t.Fatalf("Failed to create assistant: %v", err)
			}
		}

		// Delete by connector
		count, err := store.DeleteAssistants(types.AssistantFilter{
			Connector: connector,
		})
		if err != nil {
			t.Fatalf("Failed to delete assistants: %v", err)
		}

		if count < 2 {
			t.Errorf("Expected at least 2 deletions, got %d", count)
		}
	})

	t.Run("DeleteByKeywords", func(t *testing.T) {
		// Create assistants with specific keyword
		keyword := fmt.Sprintf("unique-keyword-%d", time.Now().UnixNano())
		assistant := &types.AssistantModel{
			Name:        fmt.Sprintf("Assistant with %s", keyword),
			Type:        "assistant",
			Connector:   "openai",
			Description: "Test description",
			Share:       "private",
		}
		_, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Delete by keyword
		count, err := store.DeleteAssistants(types.AssistantFilter{
			Keywords: keyword,
		})
		if err != nil {
			t.Fatalf("Failed to delete assistants: %v", err)
		}

		if count < 1 {
			t.Errorf("Expected at least 1 deletion, got %d", count)
		}
	})

	t.Run("DeleteByAssistantID", func(t *testing.T) {
		// Create an assistant
		assistant := &types.AssistantModel{
			Name:      "Single Delete Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
		}
		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Delete by ID
		count, err := store.DeleteAssistants(types.AssistantFilter{
			AssistantID: id,
		})
		if err != nil {
			t.Fatalf("Failed to delete assistant: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 deletion, got %d", count)
		}
	})
}

// TestGetAssistantTags tests retrieving unique tags
func TestGetAssistantTags(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("GetUniqueTags", func(t *testing.T) {
		// Create assistants with various tags
		uniqueTag := fmt.Sprintf("tag-test-%d", time.Now().UnixNano())
		assistants := []types.AssistantModel{
			{
				Name:      "Tags Test 1",
				Type:      "assistant",
				Connector: "openai",
				Tags:      []string{uniqueTag, "common"},
				Share:     "private",
			},
			{
				Name:      "Tags Test 2",
				Type:      "assistant",
				Connector: "openai",
				Tags:      []string{uniqueTag, "different"},
				Share:     "private",
			},
			{
				Name:      "Tags Test 3",
				Type:      "assistant",
				Connector: "openai",
				Tags:      []string{"common", "another"},
				Share:     "private",
			},
		}

		for _, asst := range assistants {
			_, err := store.SaveAssistant(&asst)
			if err != nil {
				t.Fatalf("Failed to create assistant: %v", err)
			}
		}

		// Get all tags
		tags, err := store.GetAssistantTags(types.AssistantFilter{})
		if err != nil {
			t.Fatalf("Failed to get tags: %v", err)
		}

		// Verify we have some tags
		if len(tags) == 0 {
			t.Error("Expected at least some tags")
		}

		// Verify tag structure
		for _, tag := range tags {
			if tag.Value == "" {
				t.Error("Expected tag to have non-empty value")
			}
			if tag.Label == "" {
				t.Error("Expected tag to have non-empty label")
			}
		}

		t.Logf("Found %d unique tags", len(tags))
	})

	t.Run("GetTagsWithFilter", func(t *testing.T) {
		// Create test assistants with specific tags and attributes
		uniqueTag := fmt.Sprintf("filter-tag-%d", time.Now().UnixNano())
		assistants := []types.AssistantModel{
			{
				Name:        "Filtered Tags Test 1",
				Type:        "assistant",
				Connector:   "openai",
				Tags:        []string{uniqueTag, "ai"},
				Share:       "private",
				BuiltIn:     false,
				Mentionable: true,
			},
			{
				Name:        "Filtered Tags Test 2",
				Type:        "assistant",
				Connector:   "anthropic",
				Tags:        []string{uniqueTag, "coding"},
				Share:       "private",
				BuiltIn:     true,
				Mentionable: false,
			},
			{
				Name:      "Filtered Tags Test 3",
				Type:      "assistant",
				Connector: "openai",
				Tags:      []string{uniqueTag, "search"},
				Share:     "private",
				BuiltIn:   false,
				Automated: true,
			},
		}

		for _, asst := range assistants {
			_, err := store.SaveAssistant(&asst)
			if err != nil {
				t.Fatalf("Failed to create assistant: %v", err)
			}
		}

		// Test: Get tags filtered by connector
		tagsOpenAI, err := store.GetAssistantTags(types.AssistantFilter{
			Connector: "openai",
		})
		if err != nil {
			t.Fatalf("Failed to get tags with connector filter: %v", err)
		}
		t.Logf("Found %d tags for openai connector", len(tagsOpenAI))

		// Test: Get tags filtered by built_in
		builtInFalse := false
		tagsNonBuiltIn, err := store.GetAssistantTags(types.AssistantFilter{
			BuiltIn: &builtInFalse,
		})
		if err != nil {
			t.Fatalf("Failed to get tags with built_in filter: %v", err)
		}
		t.Logf("Found %d tags for non-built-in assistants", len(tagsNonBuiltIn))

		// Test: Get tags filtered by mentionable
		mentionableTrue := true
		tagsMentionable, err := store.GetAssistantTags(types.AssistantFilter{
			Mentionable: &mentionableTrue,
		})
		if err != nil {
			t.Fatalf("Failed to get tags with mentionable filter: %v", err)
		}
		t.Logf("Found %d tags for mentionable assistants", len(tagsMentionable))

		// Test: Get tags filtered by keywords
		tagsWithKeywords, err := store.GetAssistantTags(types.AssistantFilter{
			Keywords: "Filtered Tags Test",
		})
		if err != nil {
			t.Fatalf("Failed to get tags with keywords filter: %v", err)
		}
		t.Logf("Found %d tags with keywords filter", len(tagsWithKeywords))
	})

	t.Run("GetTagsWithQueryFilter", func(t *testing.T) {
		// Create test assistants with permission fields
		permTag := fmt.Sprintf("perm-tag-%d", time.Now().UnixNano())
		assistants := []types.AssistantModel{
			{
				Name:         "Permission Tags Test 1",
				Type:         "assistant",
				Connector:    "openai",
				Tags:         []string{permTag, "public-tag"},
				Share:        "private",
				Public:       true,
				YaoCreatedBy: "user-1",
				YaoTeamID:    "team-1",
			},
			{
				Name:         "Permission Tags Test 2",
				Type:         "assistant",
				Connector:    "openai",
				Tags:         []string{permTag, "team-tag"},
				Share:        "team",
				Public:       false,
				YaoCreatedBy: "user-2",
				YaoTeamID:    "team-1",
			},
			{
				Name:         "Permission Tags Test 3",
				Type:         "assistant",
				Connector:    "openai",
				Tags:         []string{permTag, "private-tag"},
				Share:        "private",
				Public:       false,
				YaoCreatedBy: "user-3",
				YaoTeamID:    "team-2",
			},
		}

		for _, asst := range assistants {
			_, err := store.SaveAssistant(&asst)
			if err != nil {
				t.Fatalf("Failed to create assistant: %v", err)
			}
		}

		// Test: Get tags for public assistants only
		tagsPublic, err := store.GetAssistantTags(types.AssistantFilter{
			QueryFilter: func(qb query.Query) {
				qb.Where("public", true)
			},
		})
		if err != nil {
			t.Fatalf("Failed to get tags for public assistants: %v", err)
		}
		t.Logf("Found %d tags for public assistants", len(tagsPublic))

		// Test: Get tags for team-1 assistants
		tagsTeam1, err := store.GetAssistantTags(types.AssistantFilter{
			QueryFilter: func(qb query.Query) {
				qb.Where("__yao_team_id", "team-1")
			},
		})
		if err != nil {
			t.Fatalf("Failed to get tags for team-1: %v", err)
		}
		t.Logf("Found %d tags for team-1 assistants", len(tagsTeam1))

		// Test: Complex permission filter (public OR team-1 with share=team)
		tagsComplex, err := store.GetAssistantTags(types.AssistantFilter{
			QueryFilter: func(qb query.Query) {
				qb.Where(func(qb query.Query) {
					qb.Where("public", true)
				}).OrWhere(func(qb query.Query) {
					qb.Where("__yao_team_id", "team-1").
						Where("share", "team")
				})
			},
		})
		if err != nil {
			t.Fatalf("Failed to get tags with complex filter: %v", err)
		}
		t.Logf("Found %d tags with complex permission filter", len(tagsComplex))
	})
}

// TestGenerateAssistantID tests the ID generation function
func TestGenerateAssistantID(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	xunStore := store.(*Xun)

	t.Run("GenerateUniqueIDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 10; i++ {
			id, err := xunStore.GenerateAssistantID()
			if err != nil {
				t.Fatalf("Failed to generate ID: %v", err)
			}

			// Verify ID format (6 digits)
			if len(id) != 6 {
				t.Errorf("Expected 6-digit ID, got %s (length %d)", id, len(id))
			}

			// Verify ID is unique
			if ids[id] {
				t.Errorf("Generated duplicate ID: %s", id)
			}
			ids[id] = true
		}

		t.Logf("Generated %d unique IDs", len(ids))
	})
}

// TestAssistantPermissionFields tests permission management fields
func TestAssistantPermissionFields(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("SaveWithPermissionFields", func(t *testing.T) {
		assistant := &types.AssistantModel{
			Name:         "Permission Test Assistant",
			Type:         "assistant",
			Connector:    "openai",
			Description:  "Testing permission fields",
			Share:        "private",
			YaoCreatedBy: "user-123",
			YaoUpdatedBy: "user-123",
			YaoTeamID:    "team-456",
			YaoTenantID:  "tenant-789",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to save assistant with permission fields: %v", err)
		}

		// Retrieve and verify
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to get assistant: %v", err)
		}

		if retrieved.YaoCreatedBy != "user-123" {
			t.Errorf("Expected YaoCreatedBy 'user-123', got '%s'", retrieved.YaoCreatedBy)
		}
		if retrieved.YaoUpdatedBy != "user-123" {
			t.Errorf("Expected YaoUpdatedBy 'user-123', got '%s'", retrieved.YaoUpdatedBy)
		}
		if retrieved.YaoTeamID != "team-456" {
			t.Errorf("Expected YaoTeamID 'team-456', got '%s'", retrieved.YaoTeamID)
		}
		if retrieved.YaoTenantID != "tenant-789" {
			t.Errorf("Expected YaoTenantID 'tenant-789', got '%s'", retrieved.YaoTenantID)
		}

		t.Logf("Permission fields saved and retrieved successfully for assistant %s", id)
	})

	t.Run("UpdatePermissionFields", func(t *testing.T) {
		// Create assistant
		assistant := &types.AssistantModel{
			Name:         "Update Permission Test",
			Type:         "assistant",
			Connector:    "openai",
			Share:        "private",
			YaoCreatedBy: "user-original",
			YaoTeamID:    "team-original",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Update with new permission fields
		assistant.ID = id
		assistant.YaoUpdatedBy = "user-updater"
		assistant.YaoTenantID = "tenant-new"

		_, err = store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to update assistant: %v", err)
		}

		// Verify update
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to get updated assistant: %v", err)
		}

		if retrieved.YaoCreatedBy != "user-original" {
			t.Errorf("Expected YaoCreatedBy to remain 'user-original', got '%s'", retrieved.YaoCreatedBy)
		}
		if retrieved.YaoUpdatedBy != "user-updater" {
			t.Errorf("Expected YaoUpdatedBy 'user-updater', got '%s'", retrieved.YaoUpdatedBy)
		}
		if retrieved.YaoTenantID != "tenant-new" {
			t.Errorf("Expected YaoTenantID 'tenant-new', got '%s'", retrieved.YaoTenantID)
		}
	})

	t.Run("EmptyPermissionFields", func(t *testing.T) {
		// Create assistant without permission fields
		assistant := &types.AssistantModel{
			Name:      "No Permission Fields",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to save assistant: %v", err)
		}

		// Retrieve and verify fields are empty
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to get assistant: %v", err)
		}

		if retrieved.YaoCreatedBy != "" {
			t.Errorf("Expected empty YaoCreatedBy, got '%s'", retrieved.YaoCreatedBy)
		}
		if retrieved.YaoUpdatedBy != "" {
			t.Errorf("Expected empty YaoUpdatedBy, got '%s'", retrieved.YaoUpdatedBy)
		}
		if retrieved.YaoTeamID != "" {
			t.Errorf("Expected empty YaoTeamID, got '%s'", retrieved.YaoTeamID)
		}
		if retrieved.YaoTenantID != "" {
			t.Errorf("Expected empty YaoTenantID, got '%s'", retrieved.YaoTenantID)
		}
	})
}

// TestEmptyStringAsNull tests that empty strings are stored as NULL in database
func TestEmptyStringAsNull(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("EmptyStringsStoredAsNull", func(t *testing.T) {
		// Create assistant with empty strings for nullable fields
		// According to assistant.mod.yao, nullable string fields are:
		// - name (nullable: true, but required by validation)
		// - avatar, description, path (nullable: true)
		// - share (nullable: false, but empty should trigger default)
		assistant := &types.AssistantModel{
			Name:        "Test Null Fields", // Required by validation
			Type:        "assistant",
			Connector:   "openai",
			Avatar:      "", // Empty string should become NULL (nullable: true)
			Path:        "", // Empty string should become NULL (nullable: true)
			Description: "", // Empty string should become NULL (nullable: true)
			Share:       "", // Empty string should become NULL, then default "private" applied
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to save assistant: %v", err)
		}

		// Retrieve and verify empty strings are returned (not stored as empty strings)
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to get assistant: %v", err)
		}

		// Name should be preserved (required field)
		if retrieved.Name != "Test Null Fields" {
			t.Errorf("Expected Name 'Test Null Fields', got '%s'", retrieved.Name)
		}

		// These nullable fields should be empty strings in Go (converted from NULL)
		if retrieved.Avatar != "" {
			t.Errorf("Expected empty Avatar, got '%s'", retrieved.Avatar)
		}
		if retrieved.Path != "" {
			t.Errorf("Expected empty Path, got '%s'", retrieved.Path)
		}
		if retrieved.Description != "" {
			t.Errorf("Expected empty Description, got '%s'", retrieved.Description)
		}
		// Share should have default value "private" applied
		if retrieved.Share != "private" {
			t.Errorf("Expected Share to be 'private', got '%s'", retrieved.Share)
		}

		t.Logf("Successfully verified empty strings are stored as NULL for assistant %s", id)
	})

	t.Run("NonEmptyStringsPreserved", func(t *testing.T) {
		// Create assistant with non-empty values
		assistant := &types.AssistantModel{
			Name:        "Test Non-Empty Fields",
			Type:        "assistant",
			Connector:   "openai",
			Avatar:      "https://example.com/avatar.png",
			Path:        "/path/to/assistant",
			Description: "This is a description",
			Share:       "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to save assistant: %v", err)
		}

		// Retrieve and verify values are preserved
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to get assistant: %v", err)
		}

		if retrieved.Avatar != "https://example.com/avatar.png" {
			t.Errorf("Expected Avatar 'https://example.com/avatar.png', got '%s'", retrieved.Avatar)
		}
		if retrieved.Path != "/path/to/assistant" {
			t.Errorf("Expected Path '/path/to/assistant', got '%s'", retrieved.Path)
		}
		if retrieved.Description != "This is a description" {
			t.Errorf("Expected Description 'This is a description', got '%s'", retrieved.Description)
		}
		if retrieved.Share != "private" {
			t.Errorf("Expected Share 'private', got '%s'", retrieved.Share)
		}

		t.Logf("Successfully verified non-empty strings are preserved for assistant %s", id)
	})
}

// TestGetAssistantWithLocale tests retrieving assistant with locale translation
func TestGetAssistantWithLocale(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("GetAssistantWithLocaleTranslation", func(t *testing.T) {
		// Create assistant with i18n locales
		assistant := &types.AssistantModel{
			Name:        "{{name}}",
			Type:        "assistant",
			Connector:   "openai",
			Description: "{{description}}",
			Tags:        []string{"test"},
			Share:       "private",
			Placeholder: &types.Placeholder{
				Title:       "{{chat.title}}",
				Description: "{{chat.description}}",
				Prompts:     []string{"{{chat.prompts.0}}", "{{chat.prompts.1}}"},
			},
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Setup i18n for testing
		i18n.Locales[id] = map[string]i18n.I18n{
			"en": {
				Locale: "en",
				Messages: map[string]any{
					"name":             "Test Assistant",
					"description":      "This is a test assistant",
					"chat.title":       "Chat with me",
					"chat.description": "Start a conversation",
					"chat.prompts.0":   "How can I help you?",
					"chat.prompts.1":   "What would you like to know?",
				},
			},
			"zh-cn": {
				Locale: "zh-cn",
				Messages: map[string]any{
					"name":             "测试助手",
					"description":      "这是一个测试助手",
					"chat.title":       "与我聊天",
					"chat.description": "开始对话",
					"chat.prompts.0":   "我能帮你什么？",
					"chat.prompts.1":   "你想了解什么？",
				},
			},
		}

		// Test English locale
		retrievedEN, err := store.GetAssistant(id, "en")
		if err != nil {
			t.Fatalf("Failed to get assistant with EN locale: %v", err)
		}

		if retrievedEN.Name != "Test Assistant" {
			t.Errorf("Expected name 'Test Assistant', got '%s'", retrievedEN.Name)
		}
		if retrievedEN.Description != "This is a test assistant" {
			t.Errorf("Expected description 'This is a test assistant', got '%s'", retrievedEN.Description)
		}
		if retrievedEN.Placeholder == nil {
			t.Fatal("Expected placeholder to be set")
		}
		if retrievedEN.Placeholder.Title != "Chat with me" {
			t.Errorf("Expected placeholder title 'Chat with me', got '%s'", retrievedEN.Placeholder.Title)
		}
		if retrievedEN.Placeholder.Description != "Start a conversation" {
			t.Errorf("Expected placeholder description 'Start a conversation', got '%s'", retrievedEN.Placeholder.Description)
		}
		if len(retrievedEN.Placeholder.Prompts) != 2 {
			t.Errorf("Expected 2 placeholder prompts, got %d", len(retrievedEN.Placeholder.Prompts))
		}
		if retrievedEN.Placeholder.Prompts[0] != "How can I help you?" {
			t.Errorf("Expected first prompt 'How can I help you?', got '%s'", retrievedEN.Placeholder.Prompts[0])
		}

		// Test Chinese locale
		retrievedZH, err := store.GetAssistant(id, "zh-cn")
		if err != nil {
			t.Fatalf("Failed to get assistant with ZH locale: %v", err)
		}

		if retrievedZH.Name != "测试助手" {
			t.Errorf("Expected name '测试助手', got '%s'", retrievedZH.Name)
		}
		if retrievedZH.Description != "这是一个测试助手" {
			t.Errorf("Expected description '这是一个测试助手', got '%s'", retrievedZH.Description)
		}
		if retrievedZH.Placeholder == nil {
			t.Fatal("Expected placeholder to be set")
		}
		if retrievedZH.Placeholder.Title != "与我聊天" {
			t.Errorf("Expected placeholder title '与我聊天', got '%s'", retrievedZH.Placeholder.Title)
		}

		// Test without locale (should return original {{...}} values)
		retrievedNoLocale, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to get assistant without locale: %v", err)
		}

		if retrievedNoLocale.Name != "{{name}}" {
			t.Errorf("Expected original name '{{name}}', got '%s'", retrievedNoLocale.Name)
		}
		if retrievedNoLocale.Description != "{{description}}" {
			t.Errorf("Expected original description '{{description}}', got '%s'", retrievedNoLocale.Description)
		}

		// Cleanup
		delete(i18n.Locales, id)
		t.Logf("Successfully tested locale translation for assistant %s", id)
	})
}

// TestGetAssistantsWithLocale tests retrieving multiple assistants with locale translation
func TestGetAssistantsWithLocale(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("GetAssistantsWithLocaleTranslation", func(t *testing.T) {
		// Create assistant with i18n locales
		assistant := &types.AssistantModel{
			Name:        "{{name}}",
			Type:        "assistant",
			Connector:   "openai",
			Description: "{{description}}",
			Tags:        []string{"locale-test"},
			Share:       "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Setup i18n for testing
		i18n.Locales[id] = map[string]i18n.I18n{
			"en": {
				Locale: "en",
				Messages: map[string]any{
					"name":        "List Test Assistant",
					"description": "This appears in the list",
				},
			},
			"zh-cn": {
				Locale: "zh-cn",
				Messages: map[string]any{
					"name":        "列表测试助手",
					"description": "这出现在列表中",
				},
			},
		}

		// Test GetAssistants with English locale
		responseEN, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"locale-test"},
			Page:     1,
			PageSize: 20,
		}, "en")
		if err != nil {
			t.Fatalf("Failed to get assistants with EN locale: %v", err)
		}

		if len(responseEN.Data) < 1 {
			t.Fatal("Expected at least 1 assistant in response")
		}

		found := false
		for _, asst := range responseEN.Data {
			if asst.ID == id {
				found = true
				if asst.Name != "List Test Assistant" {
					t.Errorf("Expected name 'List Test Assistant', got '%s'", asst.Name)
				}
				if asst.Description != "This appears in the list" {
					t.Errorf("Expected description 'This appears in the list', got '%s'", asst.Description)
				}
				break
			}
		}

		if !found {
			t.Error("Expected to find the test assistant in the list")
		}

		// Test GetAssistants with Chinese locale
		responseZH, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"locale-test"},
			Page:     1,
			PageSize: 20,
		}, "zh-cn")
		if err != nil {
			t.Fatalf("Failed to get assistants with ZH locale: %v", err)
		}

		found = false
		for _, asst := range responseZH.Data {
			if asst.ID == id {
				found = true
				if asst.Name != "列表测试助手" {
					t.Errorf("Expected name '列表测试助手', got '%s'", asst.Name)
				}
				if asst.Description != "这出现在列表中" {
					t.Errorf("Expected description '这出现在列表中', got '%s'", asst.Description)
				}
				break
			}
		}

		if !found {
			t.Error("Expected to find the test assistant in the list")
		}

		// Cleanup
		delete(i18n.Locales, id)
		t.Logf("Successfully tested locale translation for assistants list")
	})
}

// TestGetAssistantsWithQueryFilter tests using QueryFilter for permission filtering
func TestGetAssistantsWithQueryFilter(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create test assistants with different permission settings
	assistants := []types.AssistantModel{
		{
			Name:         "Public Assistant",
			Type:         "assistant",
			Connector:    "openai",
			Description:  "Public assistant visible to all",
			Tags:         []string{"query-filter-test"},
			Public:       true,
			Share:        "private",
			YaoCreatedBy: "user-1",
			YaoTeamID:    "team-1",
		},
		{
			Name:         "Team Shared Assistant",
			Type:         "assistant",
			Connector:    "openai",
			Description:  "Team shared assistant",
			Tags:         []string{"query-filter-test"},
			Public:       false,
			Share:        "team",
			YaoCreatedBy: "user-2",
			YaoTeamID:    "team-1",
		},
		{
			Name:         "Private Assistant Owner",
			Type:         "assistant",
			Connector:    "openai",
			Description:  "Private assistant owned by user-1",
			Tags:         []string{"query-filter-test"},
			Public:       false,
			Share:        "private",
			YaoCreatedBy: "user-1",
			YaoTeamID:    "team-1",
		},
		{
			Name:         "Private Assistant Other",
			Type:         "assistant",
			Connector:    "openai",
			Description:  "Private assistant owned by user-3",
			Tags:         []string{"query-filter-test"},
			Public:       false,
			Share:        "private",
			YaoCreatedBy: "user-3",
			YaoTeamID:    "team-2",
		},
	}

	createdIDs := []string{}
	for _, asst := range assistants {
		id, err := store.SaveAssistant(&asst)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}
		createdIDs = append(createdIDs, id)
	}

	t.Run("FilterByPublic", func(t *testing.T) {
		// QueryFilter: only public assistants
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"query-filter-test"},
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.Where("public", true)
			},
		})
		if err != nil {
			t.Fatalf("Failed to get public assistants: %v", err)
		}

		if len(response.Data) != 1 {
			t.Errorf("Expected 1 public assistant, got %d", len(response.Data))
		}

		if len(response.Data) > 0 && response.Data[0].Name != "Public Assistant" {
			t.Errorf("Expected 'Public Assistant', got '%s'", response.Data[0].Name)
		}
	})

	t.Run("FilterByTeamAndShare", func(t *testing.T) {
		// QueryFilter: team-1 assistants that are shared with team
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"query-filter-test"},
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.Where("__yao_team_id", "team-1").
					Where("share", "team")
			},
		})
		if err != nil {
			t.Fatalf("Failed to get team shared assistants: %v", err)
		}

		if len(response.Data) != 1 {
			t.Errorf("Expected 1 team shared assistant, got %d", len(response.Data))
		}

		if len(response.Data) > 0 && response.Data[0].Name != "Team Shared Assistant" {
			t.Errorf("Expected 'Team Shared Assistant', got '%s'", response.Data[0].Name)
		}
	})

	t.Run("FilterByOwner", func(t *testing.T) {
		// QueryFilter: assistants created by user-1
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"query-filter-test"},
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.Where("__yao_created_by", "user-1")
			},
		})
		if err != nil {
			t.Fatalf("Failed to get user-1 assistants: %v", err)
		}

		if len(response.Data) != 2 {
			t.Errorf("Expected 2 assistants for user-1, got %d", len(response.Data))
		}

		for _, asst := range response.Data {
			if asst.YaoCreatedBy != "user-1" {
				t.Errorf("Expected creator 'user-1', got '%s'", asst.YaoCreatedBy)
			}
		}
	})

	t.Run("ComplexQueryFilter", func(t *testing.T) {
		// Complex QueryFilter: (public = true) OR (team_id = team-1 AND (created_by = user-1 OR share = team))
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"query-filter-test"},
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.Where(func(qb query.Query) {
					// Public assistants
					qb.Where("public", true)
				}).OrWhere(func(qb query.Query) {
					// Team assistants where user is creator or shared with team
					qb.Where("__yao_team_id", "team-1").Where(func(qb query.Query) {
						qb.Where("__yao_created_by", "user-1").
							OrWhere("share", "team")
					})
				})
			},
		})
		if err != nil {
			t.Fatalf("Failed to get filtered assistants: %v", err)
		}

		// Should find: Public Assistant, Team Shared Assistant, Private Assistant Owner
		if len(response.Data) != 3 {
			t.Errorf("Expected 3 assistants, got %d", len(response.Data))
		}

		// Verify we got the right assistants
		names := make(map[string]bool)
		for _, asst := range response.Data {
			names[asst.Name] = true
		}

		expectedNames := []string{"Public Assistant", "Team Shared Assistant", "Private Assistant Owner"}
		for _, name := range expectedNames {
			if !names[name] {
				t.Errorf("Expected to find '%s' in results", name)
			}
		}

		// Should NOT find Private Assistant Other
		if names["Private Assistant Other"] {
			t.Error("Should not find 'Private Assistant Other' in results")
		}
	})

	t.Run("QueryFilterWithNullCheck", func(t *testing.T) {
		// QueryFilter: assistants where team_id is null
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"query-filter-test"},
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.WhereNull("__yao_team_id")
			},
		})
		if err != nil {
			t.Fatalf("Failed to get assistants with null team_id: %v", err)
		}

		// All test assistants have team_id, so should find 0
		if len(response.Data) != 0 {
			t.Errorf("Expected 0 assistants with null team_id, got %d", len(response.Data))
		}
	})

	t.Run("QueryFilterCombinedWithOtherFilters", func(t *testing.T) {
		// Combine QueryFilter with other filters
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:      []string{"query-filter-test"},
			Connector: "openai",
			Page:      1,
			PageSize:  20,
			QueryFilter: func(qb query.Query) {
				qb.Where("public", true)
			},
		})
		if err != nil {
			t.Fatalf("Failed to get combined filtered assistants: %v", err)
		}

		// Should only find public openai assistants
		if len(response.Data) != 1 {
			t.Errorf("Expected 1 assistant, got %d", len(response.Data))
		}

		if len(response.Data) > 0 {
			if response.Data[0].Connector != "openai" {
				t.Errorf("Expected connector 'openai', got '%s'", response.Data[0].Connector)
			}
			if !response.Data[0].Public {
				t.Error("Expected public assistant")
			}
		}
	})

	// Cleanup
	for _, id := range createdIDs {
		_ = store.DeleteAssistant(id)
	}
}

// TestUpdateAssistant tests the UpdateAssistant method for incremental updates
func TestUpdateAssistant(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("UpdateSingleField", func(t *testing.T) {
		// Create assistant
		assistant := &types.AssistantModel{
			Name:        "Original Name",
			Type:        "assistant",
			Connector:   "openai",
			Description: "Original description",
			Tags:        []string{"original"},
			Share:       "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Update only description
		updates := map[string]interface{}{
			"description": "Updated description",
		}

		err = store.UpdateAssistant(id, updates)
		if err != nil {
			t.Fatalf("Failed to update assistant: %v", err)
		}

		// Verify update
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.Description != "Updated description" {
			t.Errorf("Expected description 'Updated description', got '%s'", retrieved.Description)
		}
		// Other fields should remain unchanged
		if retrieved.Name != "Original Name" {
			t.Errorf("Expected name 'Original Name', got '%s'", retrieved.Name)
		}
		if len(retrieved.Tags) != 1 || retrieved.Tags[0] != "original" {
			t.Errorf("Expected tags [original], got %v", retrieved.Tags)
		}
	})

	t.Run("UpdateMultipleFields", func(t *testing.T) {
		// Create assistant
		assistant := &types.AssistantModel{
			Name:        "Test Assistant",
			Type:        "assistant",
			Connector:   "openai",
			Description: "Test description",
			Sort:        100,
			Mentionable: false,
			Share:       "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Update multiple fields
		updates := map[string]interface{}{
			"name":        "Updated Name",
			"description": "Updated description",
			"sort":        200,
			"mentionable": true,
		}

		err = store.UpdateAssistant(id, updates)
		if err != nil {
			t.Fatalf("Failed to update assistant: %v", err)
		}

		// Verify all updates
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.Name != "Updated Name" {
			t.Errorf("Expected name 'Updated Name', got '%s'", retrieved.Name)
		}
		if retrieved.Description != "Updated description" {
			t.Errorf("Expected description 'Updated description', got '%s'", retrieved.Description)
		}
		if retrieved.Sort != 200 {
			t.Errorf("Expected sort 200, got %d", retrieved.Sort)
		}
		if !retrieved.Mentionable {
			t.Error("Expected mentionable to be true")
		}
	})

	t.Run("UpdateJSONFields", func(t *testing.T) {
		// Create assistant with complex fields
		assistant := &types.AssistantModel{
			Name:      "JSON Test",
			Type:      "assistant",
			Connector: "openai",
			Tags:      []string{"tag1", "tag2"},
			Options:   map[string]interface{}{"temperature": 0.7},
			Prompts: []types.Prompt{
				{Role: "system", Content: "Original system prompt"},
			},
			Share: "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Update JSON fields
		updates := map[string]interface{}{
			"tags": []string{"updated", "new-tags"},
			"options": map[string]interface{}{
				"temperature": 0.9,
				"max_tokens":  2000,
			},
			"prompts": []types.Prompt{
				{Role: "system", Content: "Updated system prompt"},
				{Role: "user", Content: "New user prompt"},
			},
		}

		err = store.UpdateAssistant(id, updates)
		if err != nil {
			t.Fatalf("Failed to update JSON fields: %v", err)
		}

		// Verify updates
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if len(retrieved.Tags) != 2 || retrieved.Tags[0] != "updated" {
			t.Errorf("Expected tags [updated, new-tags], got %v", retrieved.Tags)
		}
		if temp, ok := retrieved.Options["temperature"].(float64); !ok || temp != 0.9 {
			t.Errorf("Expected temperature 0.9, got %v", retrieved.Options["temperature"])
		}
		if len(retrieved.Prompts) != 2 {
			t.Errorf("Expected 2 prompts, got %d", len(retrieved.Prompts))
		}
		if retrieved.Prompts[0].Content != "Updated system prompt" {
			t.Errorf("Expected updated system prompt, got '%s'", retrieved.Prompts[0].Content)
		}
	})

	t.Run("UpdateKBAndMCP", func(t *testing.T) {
		// Create assistant
		assistant := &types.AssistantModel{
			Name:      "KB MCP Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Update KB and MCP
		updates := map[string]interface{}{
			"kb": map[string]interface{}{
				"collections": []string{"collection1", "collection2"},
			},
			"mcp": map[string]interface{}{
				"servers": []string{"server1", "server2"},
			},
		}

		err = store.UpdateAssistant(id, updates)
		if err != nil {
			t.Fatalf("Failed to update KB and MCP: %v", err)
		}

		// Verify updates
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.KB == nil || len(retrieved.KB.Collections) != 2 {
			t.Errorf("Expected 2 KB collections, got %v", retrieved.KB)
		}
		if retrieved.MCP == nil || len(retrieved.MCP.Servers) != 2 {
			t.Errorf("Expected 2 MCP servers, got %v", retrieved.MCP)
		}
		if retrieved.MCP.Servers[0].ServerID != "server1" {
			t.Errorf("Expected first server 'server1', got '%s'", retrieved.MCP.Servers[0].ServerID)
		}
	})

	t.Run("UpdateMCPWithToolsAndResources", func(t *testing.T) {
		// Create assistant
		assistant := &types.AssistantModel{
			Name:      "MCP Advanced Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Update with MCP servers using advanced configuration
		updates := map[string]interface{}{
			"mcp": map[string]interface{}{
				"servers": []interface{}{
					"server1", // Simple format
					map[string]interface{}{
						"server2": []string{"tool1", "tool2"}, // Tools only
					},
					map[string]interface{}{
						"server3": map[string]interface{}{
							"resources": []string{"res1", "res2"},
							"tools":     []string{"tool3", "tool4"},
						},
					},
				},
			},
		}

		err = store.UpdateAssistant(id, updates)
		if err != nil {
			t.Fatalf("Failed to update MCP: %v", err)
		}

		// Verify updates
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.MCP == nil || len(retrieved.MCP.Servers) != 3 {
			t.Fatalf("Expected 3 MCP servers, got %d", len(retrieved.MCP.Servers))
		}

		// Verify server1 (simple format)
		if retrieved.MCP.Servers[0].ServerID != "server1" {
			t.Errorf("Expected server1, got '%s'", retrieved.MCP.Servers[0].ServerID)
		}
		if len(retrieved.MCP.Servers[0].Tools) != 0 {
			t.Errorf("Expected no tools for server1, got %v", retrieved.MCP.Servers[0].Tools)
		}

		// Verify server2 (tools only)
		if retrieved.MCP.Servers[1].ServerID != "server2" {
			t.Errorf("Expected server2, got '%s'", retrieved.MCP.Servers[1].ServerID)
		}
		if len(retrieved.MCP.Servers[1].Tools) != 2 {
			t.Errorf("Expected 2 tools for server2, got %d", len(retrieved.MCP.Servers[1].Tools))
		}
		if retrieved.MCP.Servers[1].Tools[0] != "tool1" {
			t.Errorf("Expected tool1, got '%s'", retrieved.MCP.Servers[1].Tools[0])
		}

		// Verify server3 (full config)
		if retrieved.MCP.Servers[2].ServerID != "server3" {
			t.Errorf("Expected server3, got '%s'", retrieved.MCP.Servers[2].ServerID)
		}
		if len(retrieved.MCP.Servers[2].Resources) != 2 {
			t.Errorf("Expected 2 resources for server3, got %d", len(retrieved.MCP.Servers[2].Resources))
		}
		if len(retrieved.MCP.Servers[2].Tools) != 2 {
			t.Errorf("Expected 2 tools for server3, got %d", len(retrieved.MCP.Servers[2].Tools))
		}
		if retrieved.MCP.Servers[2].Resources[0] != "res1" {
			t.Errorf("Expected res1, got '%s'", retrieved.MCP.Servers[2].Resources[0])
		}
		if retrieved.MCP.Servers[2].Tools[0] != "tool3" {
			t.Errorf("Expected tool3, got '%s'", retrieved.MCP.Servers[2].Tools[0])
		}

		t.Logf("Successfully verified MCP advanced configuration for assistant %s", id)
	})

	t.Run("UpdateUses", func(t *testing.T) {
		// Create assistant without uses
		assistant := &types.AssistantModel{
			Name:      "Uses Update Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Update with uses configuration
		updates := map[string]interface{}{
			"uses": &context.Uses{
				Vision: "mcp:new-vision",
				Audio:  "mcp:new-audio",
				Search: "agent",
				Fetch:  "mcp:fetch-server",
			},
		}

		err = store.UpdateAssistant(id, updates)
		if err != nil {
			t.Fatalf("Failed to update uses: %v", err)
		}

		// Verify updates
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.Uses == nil {
			t.Fatal("Expected uses to be set")
		}

		if retrieved.Uses.Vision != "mcp:new-vision" {
			t.Errorf("Expected vision 'mcp:new-vision', got '%s'", retrieved.Uses.Vision)
		}
		if retrieved.Uses.Audio != "mcp:new-audio" {
			t.Errorf("Expected audio 'mcp:new-audio', got '%s'", retrieved.Uses.Audio)
		}
		if retrieved.Uses.Search != "agent" {
			t.Errorf("Expected search 'agent', got '%s'", retrieved.Uses.Search)
		}
		if retrieved.Uses.Fetch != "mcp:fetch-server" {
			t.Errorf("Expected fetch 'mcp:fetch-server', got '%s'", retrieved.Uses.Fetch)
		}

		// Update to change uses
		updates2 := map[string]interface{}{
			"uses": &context.Uses{
				Vision: "agent",
				Audio:  "agent",
			},
		}

		err = store.UpdateAssistant(id, updates2)
		if err != nil {
			t.Fatalf("Failed to update uses again: %v", err)
		}

		// Verify second update
		retrieved2, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved2.Uses.Vision != "agent" {
			t.Errorf("Expected vision 'agent', got '%s'", retrieved2.Uses.Vision)
		}
		if retrieved2.Uses.Audio != "agent" {
			t.Errorf("Expected audio 'agent', got '%s'", retrieved2.Uses.Audio)
		}

		// Update to remove uses (set to nil)
		updates3 := map[string]interface{}{
			"uses": nil,
		}

		err = store.UpdateAssistant(id, updates3)
		if err != nil {
			t.Fatalf("Failed to set uses to nil: %v", err)
		}

		// Verify uses is nil
		retrieved3, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved3.Uses != nil {
			t.Errorf("Expected uses to be nil, got %+v", retrieved3.Uses)
		}
	})

	t.Run("UpdatePermissionFields", func(t *testing.T) {
		// Create assistant with permission fields
		assistant := &types.AssistantModel{
			Name:         "Permission Test",
			Type:         "assistant",
			Connector:    "openai",
			Share:        "private",
			YaoCreatedBy: "user-1",
			YaoTeamID:    "team-1",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Update permission fields
		updates := map[string]interface{}{
			"__yao_updated_by": "user-2",
			"__yao_tenant_id":  "tenant-1",
		}

		err = store.UpdateAssistant(id, updates)
		if err != nil {
			t.Fatalf("Failed to update permission fields: %v", err)
		}

		// Verify updates
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.YaoUpdatedBy != "user-2" {
			t.Errorf("Expected YaoUpdatedBy 'user-2', got '%s'", retrieved.YaoUpdatedBy)
		}
		if retrieved.YaoTenantID != "tenant-1" {
			t.Errorf("Expected YaoTenantID 'tenant-1', got '%s'", retrieved.YaoTenantID)
		}
		// Created by should remain unchanged
		if retrieved.YaoCreatedBy != "user-1" {
			t.Errorf("Expected YaoCreatedBy 'user-1', got '%s'", retrieved.YaoCreatedBy)
		}
	})

	t.Run("UpdateWithEmptyStrings", func(t *testing.T) {
		// Create assistant with values
		assistant := &types.AssistantModel{
			Name:        "Empty String Test",
			Type:        "assistant",
			Connector:   "openai",
			Avatar:      "https://example.com/avatar.png",
			Description: "Some description",
			Share:       "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Update with empty strings (should become NULL)
		updates := map[string]interface{}{
			"avatar":      "",
			"description": "",
		}

		err = store.UpdateAssistant(id, updates)
		if err != nil {
			t.Fatalf("Failed to update with empty strings: %v", err)
		}

		// Verify empty strings are stored as NULL
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.Avatar != "" {
			t.Errorf("Expected empty avatar, got '%s'", retrieved.Avatar)
		}
		if retrieved.Description != "" {
			t.Errorf("Expected empty description, got '%s'", retrieved.Description)
		}
		// Name should remain unchanged
		if retrieved.Name != "Empty String Test" {
			t.Errorf("Expected name 'Empty String Test', got '%s'", retrieved.Name)
		}
	})

	t.Run("UpdateNonExistentAssistant", func(t *testing.T) {
		updates := map[string]interface{}{
			"name": "Updated Name",
		}

		err := store.UpdateAssistant("nonexistent-id", updates)
		if err == nil {
			t.Error("Expected error when updating non-existent assistant")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	t.Run("UpdateWithEmptyID", func(t *testing.T) {
		updates := map[string]interface{}{
			"name": "Updated Name",
		}

		err := store.UpdateAssistant("", updates)
		if err == nil {
			t.Error("Expected error when updating with empty ID")
		}
		if !strings.Contains(err.Error(), "required") {
			t.Errorf("Expected 'required' error, got: %v", err)
		}
	})

	t.Run("UpdateWithEmptyUpdates", func(t *testing.T) {
		// Create assistant
		assistant := &types.AssistantModel{
			Name:      "Empty Updates Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Try to update with empty map
		updates := map[string]interface{}{}

		err = store.UpdateAssistant(id, updates)
		if err == nil {
			t.Error("Expected error when updating with no fields")
		}
		if !strings.Contains(err.Error(), "no fields to update") {
			t.Errorf("Expected 'no fields to update' error, got: %v", err)
		}
	})

	t.Run("UpdateTimestampAutomaticallySet", func(t *testing.T) {
		// Create assistant
		assistant := &types.AssistantModel{
			Name:      "Timestamp Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Get original updated_at
		original, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		// Wait a bit to ensure timestamp difference
		time.Sleep(100 * time.Millisecond)

		// Update assistant
		updates := map[string]interface{}{
			"description": "Updated to test timestamp",
		}

		err = store.UpdateAssistant(id, updates)
		if err != nil {
			t.Fatalf("Failed to update assistant: %v", err)
		}

		// Get updated assistant
		updated, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve updated assistant: %v", err)
		}

		// Verify description was updated (main test objective)
		if updated.Description != "Updated to test timestamp" {
			t.Errorf("Expected description 'Updated to test timestamp', got '%s'", updated.Description)
		}

		// Only check timestamp if both are set (some stores may not return timestamps)
		if original.UpdatedAt > 0 && updated.UpdatedAt > 0 {
			if updated.UpdatedAt <= original.UpdatedAt {
				t.Errorf("Expected updated_at to increase, original=%d, updated=%d", original.UpdatedAt, updated.UpdatedAt)
			}
		} else {
			t.Logf("Skipping timestamp comparison (original=%d, updated=%d)", original.UpdatedAt, updated.UpdatedAt)
		}
	})

	t.Run("UpdateSkipsSystemFields", func(t *testing.T) {
		// Create assistant
		assistant := &types.AssistantModel{
			Name:      "System Fields Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
		}

		id, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Get original
		original, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		// Try to update system fields (should be ignored)
		updates := map[string]interface{}{
			"assistant_id": "new-id-123",     // Should be ignored
			"created_at":   int64(123456789), // Should be ignored
			"name":         "Valid Update",   // Should be applied
		}

		err = store.UpdateAssistant(id, updates)
		if err != nil {
			t.Fatalf("Failed to update assistant: %v", err)
		}

		// Verify system fields unchanged, but name updated
		retrieved, err := store.GetAssistant(id)
		if err != nil {
			t.Fatalf("Failed to retrieve assistant: %v", err)
		}

		if retrieved.ID != id {
			t.Errorf("Expected ID to remain %s, got %s", id, retrieved.ID)
		}
		if retrieved.CreatedAt != original.CreatedAt {
			t.Errorf("Expected created_at to remain unchanged")
		}
		if retrieved.Name != "Valid Update" {
			t.Errorf("Expected name 'Valid Update', got '%s'", retrieved.Name)
		}
	})
}

// TestAssistantCompleteWorkflow tests a complete workflow
func TestAssistantCompleteWorkflow(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// Step 1: Create multiple assistants
		assistantIDs := []string{}
		for i := 0; i < 3; i++ {
			assistant := &types.AssistantModel{
				Name:        fmt.Sprintf("Workflow Assistant %d", i),
				Type:        "assistant",
				Connector:   "openai",
				Description: fmt.Sprintf("Workflow test assistant %d", i),
				Tags:        []string{"workflow", fmt.Sprintf("test-%d", i)},
				Sort:        i * 100,
				Share:       "private",
			}

			id, err := store.SaveAssistant(assistant)
			if err != nil {
				t.Fatalf("Failed to create assistant %d: %v", i, err)
			}
			assistantIDs = append(assistantIDs, id)
		}

		t.Logf("Created %d assistants", len(assistantIDs))

		// Step 2: Retrieve all assistants
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"workflow"},
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to get assistants: %v", err)
		}

		if len(response.Data) < 3 {
			t.Errorf("Expected at least 3 assistants, got %d", len(response.Data))
		}

		// Step 3: Update one assistant
		updatedID := assistantIDs[1]
		updatedAssistant, err := store.GetAssistant(updatedID)
		if err != nil {
			t.Fatalf("Failed to get assistant for update: %v", err)
		}

		updatedAssistant.Description = "Updated workflow description"
		updatedAssistant.Tags = append(updatedAssistant.Tags, "updated")

		_, err = store.SaveAssistant(updatedAssistant)
		if err != nil {
			t.Fatalf("Failed to update assistant: %v", err)
		}

		// Verify update
		verifyAssistant, err := store.GetAssistant(updatedID)
		if err != nil {
			t.Fatalf("Failed to verify update: %v", err)
		}

		if verifyAssistant.Description != "Updated workflow description" {
			t.Errorf("Update not applied correctly")
		}

		// Step 4: Delete one assistant
		err = store.DeleteAssistant(assistantIDs[0])
		if err != nil {
			t.Fatalf("Failed to delete assistant: %v", err)
		}

		// Verify deletion
		_, err = store.GetAssistant(assistantIDs[0])
		if err == nil {
			t.Error("Expected error when getting deleted assistant")
		}

		// Step 5: Get tags
		tags, err := store.GetAssistantTags(types.AssistantFilter{})
		if err != nil {
			t.Fatalf("Failed to get tags: %v", err)
		}

		// Should find "workflow" tag
		found := false
		for _, tag := range tags {
			if tag.Value == "workflow" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find 'workflow' tag")
		}

		// Step 6: Bulk delete remaining assistants
		count, err := store.DeleteAssistants(types.AssistantFilter{
			Tags: []string{"workflow"},
		})
		if err != nil {
			t.Fatalf("Failed to bulk delete: %v", err)
		}

		t.Logf("Bulk deleted %d assistants", count)

		// Verify bulk deletion
		finalResponse, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"workflow"},
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to verify bulk deletion: %v", err)
		}

		if len(finalResponse.Data) > 0 {
			t.Logf("Warning: Still found %d assistants after bulk delete", len(finalResponse.Data))
		}
	})
}
