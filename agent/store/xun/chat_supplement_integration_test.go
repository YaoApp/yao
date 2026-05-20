//go:build integration

package xun_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	goumodel "github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestListChatsByUserAndTeam(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	chat1 := &types.Chat{AssistantID: "test_assistant", Title: "User1 Team1 Chat"}
	chat2 := &types.Chat{AssistantID: "test_assistant", Title: "User1 Team2 Chat"}
	chat3 := &types.Chat{AssistantID: "test_assistant", Title: "User2 Team1 Chat"}
	chat4 := &types.Chat{AssistantID: "test_assistant", Title: "User2 Team2 Chat"}

	for _, chat := range []*types.Chat{chat1, chat2, chat3, chat4} {
		err := store.CreateChat(chat)
		require.NoError(t, err)
	}
	defer func() {
		store.DeleteChat(chat1.ChatID)
		store.DeleteChat(chat2.ChatID)
		store.DeleteChat(chat3.ChatID)
		store.DeleteChat(chat4.ChatID)
	}()

	updatePermissionFields := func(chatID, userID, teamID string) error {
		m := goumodel.Select("__yao.agent.chat")
		if m == nil {
			return fmt.Errorf("model __yao.agent.chat not found")
		}
		_, err := m.UpdateWhere(
			goumodel.QueryParam{Wheres: []goumodel.QueryWhere{{Column: "chat_id", Value: chatID}}},
			map[string]interface{}{
				"__yao_created_by": userID,
				"__yao_team_id":    teamID,
			},
		)
		return err
	}

	require.NoError(t, updatePermissionFields(chat1.ChatID, "user1", "team1"))
	require.NoError(t, updatePermissionFields(chat2.ChatID, "user1", "team2"))
	require.NoError(t, updatePermissionFields(chat3.ChatID, "user2", "team1"))
	require.NoError(t, updatePermissionFields(chat4.ChatID, "user2", "team2"))

	t.Run("FilterByUserID", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			UserID: "user1", Page: 1, PageSize: 20,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(result.Data))
		for _, chat := range result.Data {
			assert.Contains(t, []string{"User1 Team1 Chat", "User1 Team2 Chat"}, chat.Title)
		}
	})

	t.Run("FilterByTeamID", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			TeamID: "team1", Page: 1, PageSize: 20,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(result.Data))
		for _, chat := range result.Data {
			assert.Contains(t, []string{"User1 Team1 Chat", "User2 Team1 Chat"}, chat.Title)
		}
	})

	t.Run("FilterByUserIDAndTeamID", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			UserID: "user1", TeamID: "team1", Page: 1, PageSize: 20,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(result.Data))
		if len(result.Data) > 0 {
			assert.Equal(t, "User1 Team1 Chat", result.Data[0].Title)
		}
	})

	t.Run("FilterByUserIDWithOtherFilters", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			UserID: "user1", Status: "active", Page: 1, PageSize: 20,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(result.Data))
	})

	t.Run("FilterByTeamIDWithQueryFilter", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			TeamID: "team2", Page: 1, PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.Where("title", "like", "%User1%")
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(result.Data))
		if len(result.Data) > 0 {
			assert.Equal(t, "User1 Team2 Chat", result.Data[0].Title)
		}
	})

	t.Run("FilterByNonExistentUser", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			UserID: "nonexistent_user", Page: 1, PageSize: 20,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, len(result.Data))
	})

	t.Run("FilterByNonExistentTeam", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			TeamID: "nonexistent_team", Page: 1, PageSize: 20,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, len(result.Data))
	})

	t.Run("QueryFilterForOrCondition", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			Page: 1, PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.Where(func(sub query.Query) {
					sub.Where("__yao_created_by", "user1").
						OrWhere("__yao_team_id", "team2")
				})
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(result.Data))
	})
}
