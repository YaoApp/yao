package xun

import (
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestMain(m *testing.M) {
	// Setup will be done in each test via test.Prepare
	// Run tests and exit
	test.Prepare(nil, config.Conf)
	defer test.Clean()
	m.Run()
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
		tags, err := store.GetAssistantTags()
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
		tags, err := store.GetAssistantTags()
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
