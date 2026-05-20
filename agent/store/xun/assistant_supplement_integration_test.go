//go:build integration

package xun_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestAssistantPermissionFields(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

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
		require.NoError(t, err)

		retrieved, err := store.GetAssistant(id, nil)
		require.NoError(t, err)

		assert.Equal(t, "user-123", retrieved.YaoCreatedBy)
		assert.Equal(t, "user-123", retrieved.YaoUpdatedBy)
		assert.Equal(t, "team-456", retrieved.YaoTeamID)
		assert.Equal(t, "tenant-789", retrieved.YaoTenantID)
	})

	t.Run("UpdatePermissionFields", func(t *testing.T) {
		assistant := &types.AssistantModel{
			Name:         "Update Permission Test",
			Type:         "assistant",
			Connector:    "openai",
			Share:        "private",
			YaoCreatedBy: "user-original",
			YaoTeamID:    "team-original",
		}

		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)

		assistant.ID = id
		assistant.YaoUpdatedBy = "user-updater"
		assistant.YaoTenantID = "tenant-new"

		_, err = store.SaveAssistant(assistant)
		require.NoError(t, err)

		retrieved, err := store.GetAssistant(id, nil)
		require.NoError(t, err)

		assert.Equal(t, "user-original", retrieved.YaoCreatedBy)
		assert.Equal(t, "user-updater", retrieved.YaoUpdatedBy)
		assert.Equal(t, "tenant-new", retrieved.YaoTenantID)
	})

	t.Run("EmptyPermissionFields", func(t *testing.T) {
		assistant := &types.AssistantModel{
			Name:      "No Permission Fields",
			Type:      "assistant",
			Connector: "openai",
			Share:     "private",
		}

		id, err := store.SaveAssistant(assistant)
		require.NoError(t, err)

		retrieved, err := store.GetAssistant(id, nil)
		require.NoError(t, err)

		assert.Empty(t, retrieved.YaoCreatedBy)
		assert.Empty(t, retrieved.YaoUpdatedBy)
		assert.Empty(t, retrieved.YaoTeamID)
		assert.Empty(t, retrieved.YaoTenantID)
	})
}

func TestGetAssistantWithLocale(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("GetAssistantWithLocaleTranslation", func(t *testing.T) {
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
		require.NoError(t, err)

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
		defer delete(i18n.Locales, id)

		retrievedEN, err := store.GetAssistant(id, types.AssistantFullFields, "en")
		require.NoError(t, err)
		assert.Equal(t, "Test Assistant", retrievedEN.Name)
		assert.Equal(t, "This is a test assistant", retrievedEN.Description)
		require.NotNil(t, retrievedEN.Placeholder)
		assert.Equal(t, "Chat with me", retrievedEN.Placeholder.Title)
		assert.Equal(t, "Start a conversation", retrievedEN.Placeholder.Description)
		require.Len(t, retrievedEN.Placeholder.Prompts, 2)
		assert.Equal(t, "How can I help you?", retrievedEN.Placeholder.Prompts[0])

		retrievedZH, err := store.GetAssistant(id, types.AssistantFullFields, "zh-cn")
		require.NoError(t, err)
		assert.Equal(t, "测试助手", retrievedZH.Name)
		assert.Equal(t, "这是一个测试助手", retrievedZH.Description)
		require.NotNil(t, retrievedZH.Placeholder)
		assert.Equal(t, "与我聊天", retrievedZH.Placeholder.Title)

		retrievedNoLocale, err := store.GetAssistant(id, types.AssistantFullFields)
		require.NoError(t, err)
		assert.Equal(t, "{{name}}", retrievedNoLocale.Name)
		assert.Equal(t, "{{description}}", retrievedNoLocale.Description)
	})
}

func TestGetAssistantsWithQueryFilter(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

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
		a := asst
		id, err := store.SaveAssistant(&a)
		require.NoError(t, err)
		createdIDs = append(createdIDs, id)
	}
	defer func() {
		for _, id := range createdIDs {
			_ = store.DeleteAssistant(id)
		}
	}()

	t.Run("FilterByPublic", func(t *testing.T) {
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"query-filter-test"},
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.Where("public", true)
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(response.Data))
		if len(response.Data) > 0 {
			assert.Equal(t, "Public Assistant", response.Data[0].Name)
		}
	})

	t.Run("FilterByTeamAndShare", func(t *testing.T) {
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"query-filter-test"},
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.Where("__yao_team_id", "team-1").
					Where("share", "team")
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(response.Data))
		if len(response.Data) > 0 {
			assert.Equal(t, "Team Shared Assistant", response.Data[0].Name)
		}
	})

	t.Run("FilterByOwner", func(t *testing.T) {
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"query-filter-test"},
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.Where("__yao_created_by", "user-1")
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(response.Data))
		for _, asst := range response.Data {
			assert.Equal(t, "user-1", asst.YaoCreatedBy)
		}
	})

	t.Run("ComplexQueryFilter", func(t *testing.T) {
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"query-filter-test"},
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.Where(func(qb query.Query) {
					qb.Where("public", true)
				}).OrWhere(func(qb query.Query) {
					qb.Where("__yao_team_id", "team-1").Where(func(qb query.Query) {
						qb.Where("__yao_created_by", "user-1").
							OrWhere("share", "team")
					})
				})
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(response.Data))

		names := make(map[string]bool)
		for _, asst := range response.Data {
			names[asst.Name] = true
		}
		assert.True(t, names["Public Assistant"])
		assert.True(t, names["Team Shared Assistant"])
		assert.True(t, names["Private Assistant Owner"])
		assert.False(t, names["Private Assistant Other"])
	})

	t.Run("QueryFilterWithNullCheck", func(t *testing.T) {
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:     []string{"query-filter-test"},
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.WhereNull("__yao_team_id")
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 0, len(response.Data))
	})

	t.Run("QueryFilterCombinedWithOtherFilters", func(t *testing.T) {
		response, err := store.GetAssistants(types.AssistantFilter{
			Tags:      []string{"query-filter-test"},
			Connector: "openai",
			Page:      1,
			PageSize:  20,
			QueryFilter: func(qb query.Query) {
				qb.Where("public", true)
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(response.Data))
		if len(response.Data) > 0 {
			assert.Equal(t, "openai", response.Data[0].Connector)
			assert.True(t, response.Data[0].Public)
		}
	})
}
