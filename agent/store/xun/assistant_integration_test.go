//go:build integration

package xun_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestSaveAssistant(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

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
		require.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.NotEmpty(t, assistant.ID)

		t.Cleanup(func() { store.DeleteAssistant(id) })
	})

	t.Run("UpdateExistingAssistant", func(t *testing.T) {
		assistant := &types.AssistantModel{
			Name:        "Update Test Assistant",
			Type:        "assistant",
			Connector:   "openai",
			Description: "Original description",
			Tags:        []string{"original"},
			Share:       "private",
		}

		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteAssistant(id) })

		assistant.Description = "Updated description"
		assistant.Tags = []string{"updated", "modified"}
		assistant.Sort = 200

		updatedID, err := store.SaveAssistant(assistant)
		require.NoError(t, err)
		assert.Equal(t, id, updatedID)

		retrieved, err := store.GetAssistant(id, types.AssistantFullFields)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", retrieved.Description)
		assert.Equal(t, 2, len(retrieved.Tags))
		assert.Equal(t, "updated", retrieved.Tags[0])
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		_, err := store.SaveAssistant(nil)
		assert.Error(t, err)

		_, err = store.SaveAssistant(&types.AssistantModel{Type: "assistant", Connector: "openai"})
		assert.Error(t, err)

		_, err = store.SaveAssistant(&types.AssistantModel{Name: "Test", Connector: "openai"})
		assert.Error(t, err)

		_, err = store.SaveAssistant(&types.AssistantModel{Name: "Test", Type: "assistant"})
		assert.Error(t, err)
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
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteAssistant(id) })

		retrieved, err := store.GetAssistant(id, types.AssistantFullFields)
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved.Prompts))
		assert.NotNil(t, retrieved.Placeholder)
		assert.Equal(t, 3, len(retrieved.Tags))
	})

	t.Run("SaveWithMCPServers", func(t *testing.T) {
		assistant := &types.AssistantModel{
			Name:      "MCP Save Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
			MCP: &types.MCPServers{
				Servers: []types.MCPServerConfig{
					{ServerID: "server1"},
					{ServerID: "server2", Tools: []string{"tool1", "tool2"}},
					{ServerID: "server3", Resources: []string{"res1", "res2"}, Tools: []string{"tool3", "tool4"}},
				},
				Options: map[string]interface{}{"timeout": 30},
			},
		}

		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteAssistant(id) })

		retrieved, err := store.GetAssistant(id, []string{})
		require.NoError(t, err)
		require.NotNil(t, retrieved.MCP)
		assert.Equal(t, 3, len(retrieved.MCP.Servers))
		assert.Equal(t, "server1", retrieved.MCP.Servers[0].ServerID)
		assert.Equal(t, 2, len(retrieved.MCP.Servers[1].Tools))
		assert.Equal(t, 2, len(retrieved.MCP.Servers[2].Resources))
	})

	t.Run("PromptPresets", func(t *testing.T) {
		assistant := &types.AssistantModel{
			Name:      "Prompt Presets Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
			PromptPresets: map[string][]types.Prompt{
				"chat": {{Role: "system", Content: "You are a friendly chatbot"}, {Role: "user", Content: "Hello!"}},
				"task": {{Role: "system", Content: "You are a task executor"}},
			},
		}

		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteAssistant(id) })

		retrieved, err := store.GetAssistant(id, types.AssistantFullFields)
		require.NoError(t, err)
		require.NotNil(t, retrieved.PromptPresets)
		assert.Equal(t, 2, len(retrieved.PromptPresets))

		chatPrompts, ok := retrieved.PromptPresets["chat"]
		require.True(t, ok)
		assert.Equal(t, 2, len(chatPrompts))
		assert.Equal(t, "system", chatPrompts[0].Role)
	})

	t.Run("SourceField", func(t *testing.T) {
		sourceCode := `function onMessage(msg) { return { status: "ok" }; }`
		assistant := &types.AssistantModel{
			Name:      "Source Field Test",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
			Source:    sourceCode,
		}

		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteAssistant(id) })

		retrieved, err := store.GetAssistant(id, types.AssistantFullFields)
		require.NoError(t, err)
		assert.Equal(t, sourceCode, retrieved.Source)
	})

	t.Run("PermissionFields", func(t *testing.T) {
		assistant := &types.AssistantModel{
			Name:         "Permission Test",
			Type:         "assistant",
			Connector:    "openai",
			Share:        "team",
			YaoCreatedBy: "user_001",
			YaoUpdatedBy: "user_002",
			YaoTeamID:    "team_001",
			YaoTenantID:  "tenant_001",
		}

		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteAssistant(id) })

		retrieved, err := store.GetAssistant(id, types.AssistantFullFields)
		require.NoError(t, err)
		assert.Equal(t, "user_001", retrieved.YaoCreatedBy)
		assert.Equal(t, "user_002", retrieved.YaoUpdatedBy)
		assert.Equal(t, "team_001", retrieved.YaoTeamID)
		assert.Equal(t, "tenant_001", retrieved.YaoTenantID)
	})

	t.Run("EmptyStringAsNull", func(t *testing.T) {
		assistant := &types.AssistantModel{
			Name:        "Empty String Test",
			Type:        "assistant",
			Connector:   "openai",
			Share:       "private",
			Description: "",
			Avatar:      "",
		}

		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteAssistant(id) })

		retrieved, err := store.GetAssistant(id, types.AssistantFullFields)
		require.NoError(t, err)
		assert.Empty(t, retrieved.Description)
		assert.Empty(t, retrieved.Avatar)
	})
}

func TestDeleteAssistant(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("DeleteExisting", func(t *testing.T) {
		assistant := &types.AssistantModel{Name: "Delete Test", Type: "assistant", Connector: "openai", Share: "private"}
		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)

		err = store.DeleteAssistant(id)
		require.NoError(t, err)

		_, err = store.GetAssistant(id, []string{})
		assert.Error(t, err)
	})

	t.Run("DeleteNonExistent", func(t *testing.T) {
		err := store.DeleteAssistant("nonexistent_id")
		assert.Error(t, err)
	})
}

func TestGetAssistant(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	assistant := &types.AssistantModel{
		Name:        "Get Test Assistant",
		Type:        "assistant",
		Connector:   "openai",
		Share:       "private",
		Description: "Description for get test",
		Tags:        []string{"get", "test"},
		Mentionable: true,
	}
	id, err := store.SaveAssistant(assistant)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteAssistant(id) })

	t.Run("GetWithDefaultFields", func(t *testing.T) {
		retrieved, err := store.GetAssistant(id, []string{})
		require.NoError(t, err)
		assert.Equal(t, id, retrieved.ID)
		assert.Equal(t, "Get Test Assistant", retrieved.Name)
	})

	t.Run("GetWithSelectFields", func(t *testing.T) {
		retrieved, err := store.GetAssistant(id, []string{"assistant_id", "name", "type"})
		require.NoError(t, err)
		assert.Equal(t, id, retrieved.ID)
		assert.Equal(t, "Get Test Assistant", retrieved.Name)
		assert.Equal(t, "assistant", retrieved.Type)
	})

	t.Run("GetWithAllFields", func(t *testing.T) {
		retrieved, err := store.GetAssistant(id, types.AssistantFullFields)
		require.NoError(t, err)
		assert.Equal(t, "Description for get test", retrieved.Description)
		assert.Equal(t, 2, len(retrieved.Tags))
		assert.True(t, retrieved.Mentionable)
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := store.GetAssistant("nonexistent_id", []string{})
		assert.Error(t, err)
	})
}

func TestGetAssistants(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	assistants := []*types.AssistantModel{
		{Name: "Filter A1", Type: "assistant", Connector: "openai", Share: "private", Tags: []string{"alpha", "chat"}, Mentionable: true, Automated: false},
		{Name: "Filter A2", Type: "assistant", Connector: "anthropic", Share: "private", Tags: []string{"beta", "task"}, Mentionable: false, Automated: true},
		{Name: "Filter B1", Type: "tool", Connector: "openai", Share: "team", Tags: []string{"alpha"}, Mentionable: true, Automated: true},
		{Name: "Filter B2", Type: "assistant", Connector: "openai", Share: "private", Tags: []string{"gamma"}, Mentionable: false, Automated: false},
		{Name: "Filter C1", Type: "assistant", Connector: "openai", Share: "private", Tags: []string{"alpha", "task"}, Mentionable: true, Automated: true},
	}

	var ids []string
	for _, a := range assistants {
		id, err := store.SaveAssistant(a)
		require.NoError(t, err)
		ids = append(ids, id)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			store.DeleteAssistant(id)
		}
	})

	t.Run("FilterByTag", func(t *testing.T) {
		result, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"alpha"},
			Page:     1,
			PageSize: 20,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Data), 3)
	})

	t.Run("FilterByKeyword", func(t *testing.T) {
		result, err := store.GetAssistants(types.AssistantFilter{
			Keywords: "Filter A",
			Page:     1,
			PageSize: 20,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Data), 2)
	})

	t.Run("FilterByType", func(t *testing.T) {
		result, err := store.GetAssistants(types.AssistantFilter{
			Type:     "tool",
			Page:     1,
			PageSize: 20,
		})
		require.NoError(t, err)

		for _, a := range result.Data {
			assert.Equal(t, "tool", a.Type)
		}
	})

	t.Run("FilterByConnector", func(t *testing.T) {
		result, err := store.GetAssistants(types.AssistantFilter{
			Connector: "anthropic",
			Page:      1,
			PageSize:  20,
		})
		require.NoError(t, err)

		for _, a := range result.Data {
			assert.Equal(t, "anthropic", a.Connector)
		}
	})

	t.Run("FilterByMentionable", func(t *testing.T) {
		mentionable := true
		result, err := store.GetAssistants(types.AssistantFilter{
			Mentionable: &mentionable,
			Page:        1,
			PageSize:    20,
		})
		require.NoError(t, err)

		for _, a := range result.Data {
			assert.True(t, a.Mentionable)
		}
	})

	t.Run("FilterByAutomated", func(t *testing.T) {
		automated := true
		result, err := store.GetAssistants(types.AssistantFilter{
			Automated: &automated,
			Page:      1,
			PageSize:  20,
		})
		require.NoError(t, err)

		for _, a := range result.Data {
			assert.True(t, a.Automated)
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		result1, err := store.GetAssistants(types.AssistantFilter{Page: 1, PageSize: 2})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(result1.Data), 2)
		assert.Equal(t, 1, result1.Page)
		assert.Equal(t, 2, result1.PageSize)

		if result1.Total > 2 {
			result2, err := store.GetAssistants(types.AssistantFilter{Page: 2, PageSize: 2})
			require.NoError(t, err)
			assert.Equal(t, 2, result2.Page)
		}
	})
}

func TestDeleteAssistants(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("DeleteByFilter", func(t *testing.T) {
		a1 := &types.AssistantModel{Name: "Batch Del 1", Type: "assistant", Connector: "openai", Share: "private", Tags: []string{"batch_delete_test"}}
		a2 := &types.AssistantModel{Name: "Batch Del 2", Type: "assistant", Connector: "openai", Share: "private", Tags: []string{"batch_delete_test"}}

		id1, err := store.SaveAssistant(a1)
		require.NoError(t, err)
		id2, err := store.SaveAssistant(a2)
		require.NoError(t, err)

		count, err := store.DeleteAssistants(types.AssistantFilter{
			AssistantIDs: []string{id1, id2},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)

		_, err = store.GetAssistant(id1, []string{})
		assert.Error(t, err)
		_, err = store.GetAssistant(id2, []string{})
		assert.Error(t, err)
	})
}

func TestGetAssistantTags(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	a1 := &types.AssistantModel{Name: "Tag Test 1", Type: "assistant", Connector: "openai", Share: "private", Tags: []string{"unique_tag_a", "shared_tag"}}
	a2 := &types.AssistantModel{Name: "Tag Test 2", Type: "assistant", Connector: "openai", Share: "private", Tags: []string{"unique_tag_b", "shared_tag"}}

	id1, err := store.SaveAssistant(a1)
	require.NoError(t, err)
	id2, err := store.SaveAssistant(a2)
	require.NoError(t, err)
	t.Cleanup(func() {
		store.DeleteAssistant(id1)
		store.DeleteAssistant(id2)
	})

	t.Run("GetAllTags", func(t *testing.T) {
		tags, err := store.GetAssistantTags(types.AssistantFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tags), 3)

		tagValues := make(map[string]bool)
		for _, tag := range tags {
			tagValues[tag.Value] = true
			assert.NotEmpty(t, tag.Label)
		}
		assert.True(t, tagValues["unique_tag_a"])
		assert.True(t, tagValues["unique_tag_b"])
		assert.True(t, tagValues["shared_tag"])
	})

	t.Run("GetTagsFilteredByType", func(t *testing.T) {
		tags, err := store.GetAssistantTags(types.AssistantFilter{Type: "assistant"})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tags), 1)
	})
}

func TestUpdateAssistant(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("UpdateFields", func(t *testing.T) {
		assistant := &types.AssistantModel{
			Name:        "Update Fields Test",
			Type:        "assistant",
			Connector:   "openai",
			Share:       "private",
			Description: "Original",
		}
		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteAssistant(id) })

		err = store.UpdateAssistant(id, map[string]interface{}{
			"description": "Updated via UpdateAssistant",
			"mentionable": true,
			"sort":        50,
		})
		require.NoError(t, err)

		retrieved, err := store.GetAssistant(id, types.AssistantFullFields)
		require.NoError(t, err)
		assert.Equal(t, "Updated via UpdateAssistant", retrieved.Description)
		assert.True(t, retrieved.Mentionable)
		assert.Equal(t, 50, retrieved.Sort)
	})

	t.Run("UpdateNonExistent", func(t *testing.T) {
		err := store.UpdateAssistant("nonexistent_id", map[string]interface{}{"name": "test"})
		assert.Error(t, err)
	})

	t.Run("UpdateWithEmptyID", func(t *testing.T) {
		err := store.UpdateAssistant("", map[string]interface{}{"name": "test"})
		assert.Error(t, err)
	})

	t.Run("UpdateWithEmptyFields", func(t *testing.T) {
		assistant := &types.AssistantModel{Name: "Empty Update", Type: "assistant", Connector: "openai", Share: "private"}
		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteAssistant(id) })

		err = store.UpdateAssistant(id, map[string]interface{}{})
		assert.Error(t, err)
	})

	t.Run("UpdateJSONFields", func(t *testing.T) {
		assistant := &types.AssistantModel{Name: "JSON Update Test", Type: "assistant", Connector: "openai", Share: "private"}
		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteAssistant(id) })

		err = store.UpdateAssistant(id, map[string]interface{}{
			"tags":    []string{"new_tag1", "new_tag2"},
			"options": map[string]interface{}{"max_tokens": 4000},
		})
		require.NoError(t, err)

		retrieved, err := store.GetAssistant(id, types.AssistantFullFields)
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved.Tags))
		assert.Equal(t, "new_tag1", retrieved.Tags[0])
		require.NotNil(t, retrieved.Options)
	})
}

func TestAssistantCompleteWorkflow(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("CRUDWorkflow", func(t *testing.T) {
		assistant := &types.AssistantModel{
			Name:        "CRUD Workflow",
			Type:        "assistant",
			Connector:   "openai",
			Share:       "private",
			Description: "Testing full CRUD",
			Tags:        []string{"workflow", "crud"},
			Mentionable: true,
		}

		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)

		retrieved, err := store.GetAssistant(id, types.AssistantFullFields)
		require.NoError(t, err)
		assert.Equal(t, "CRUD Workflow", retrieved.Name)
		assert.Equal(t, "Testing full CRUD", retrieved.Description)

		assistant.Description = "Updated CRUD"
		assistant.Tags = []string{"workflow", "crud", "updated"}
		_, err = store.SaveAssistant(assistant)
		require.NoError(t, err)

		updated, err := store.GetAssistant(id, types.AssistantFullFields)
		require.NoError(t, err)
		assert.Equal(t, "Updated CRUD", updated.Description)
		assert.Equal(t, 3, len(updated.Tags))

		result, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"workflow"},
			Page:     1,
			PageSize: 20,
		})
		require.NoError(t, err)

		found := false
		for _, a := range result.Data {
			if a.ID == id {
				found = true
				break
			}
		}
		assert.True(t, found)

		err = store.DeleteAssistant(id)
		require.NoError(t, err)

		_, err = store.GetAssistant(id, []string{})
		assert.Error(t, err)
	})
}
