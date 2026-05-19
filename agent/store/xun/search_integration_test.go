//go:build integration

package xun_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestSaveSearch(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	chat := &types.Chat{AssistantID: "test_assistant", Title: "Search Test Chat"}
	err = store.CreateChat(chat)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

	t.Run("SaveBasicSearch", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		search := &types.Search{
			RequestID: requestID,
			ChatID:    chat.ChatID,
			Query:     "What is the weather today?",
			Source:    "web",
			Duration:  150,
		}

		err := store.SaveSearch(search)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteSearches(chat.ChatID) })

		searches, err := store.GetSearches(requestID)
		require.NoError(t, err)
		require.Equal(t, 1, len(searches))

		assert.Equal(t, "What is the weather today?", searches[0].Query)
		assert.Equal(t, "web", searches[0].Source)
		assert.Equal(t, int64(150), searches[0].Duration)
	})

	t.Run("SaveSearchWithKeywords", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		search := &types.Search{
			RequestID: requestID,
			ChatID:    chat.ChatID,
			Query:     "Latest news about AI",
			Keywords:  []string{"AI", "news", "latest"},
			Source:    "web",
			Duration:  200,
		}

		err := store.SaveSearch(search)
		require.NoError(t, err)

		searches, err := store.GetSearches(requestID)
		require.NoError(t, err)
		require.Equal(t, 1, len(searches))
		assert.Equal(t, 3, len(searches[0].Keywords))
		assert.Equal(t, "AI", searches[0].Keywords[0])
	})

	t.Run("SaveSearchWithReferences", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		search := &types.Search{
			RequestID: requestID,
			ChatID:    chat.ChatID,
			Query:     "How to learn Go programming?",
			Source:    "web",
			References: []types.Reference{
				{Index: 1, Type: "web", Title: "Go Programming Tutorial", URL: "https://go.dev/tour/", Snippet: "An interactive introduction to Go"},
				{Index: 2, Type: "web", Title: "Effective Go", URL: "https://go.dev/doc/effective_go", Snippet: "Tips for writing clear, idiomatic Go code"},
			},
			XML:      "<references>...</references>",
			Prompt:   "Please cite sources using [1], [2]...",
			Duration: 300,
		}

		err := store.SaveSearch(search)
		require.NoError(t, err)

		searches, err := store.GetSearches(requestID)
		require.NoError(t, err)
		require.Equal(t, 1, len(searches))
		assert.Equal(t, 2, len(searches[0].References))
		assert.Equal(t, "Go Programming Tutorial", searches[0].References[0].Title)
		assert.Equal(t, "<references>...</references>", searches[0].XML)
		assert.Equal(t, "Please cite sources using [1], [2]...", searches[0].Prompt)
	})

	t.Run("SaveSearchWithConfig", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		search := &types.Search{
			RequestID: requestID,
			ChatID:    chat.ChatID,
			Query:     "Config test",
			Source:    "auto",
			Config: map[string]any{
				"uses": map[string]any{"search": "builtin", "web": "builtin", "keyword": "builtin"},
				"web":  map[string]any{"provider": "tavily", "max_results": 5},
			},
			Duration: 100,
		}

		err := store.SaveSearch(search)
		require.NoError(t, err)

		searches, err := store.GetSearches(requestID)
		require.NoError(t, err)
		require.Equal(t, 1, len(searches))
		require.NotNil(t, searches[0].Config)

		uses, ok := searches[0].Config["uses"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "builtin", uses["search"])
	})

	t.Run("SaveSearchWithEntitiesAndRelations", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		search := &types.Search{
			RequestID: requestID,
			ChatID:    chat.ChatID,
			Query:     "Who is the CEO of Apple?",
			Source:    "kb",
			Entities: []types.Entity{
				{Name: "Apple", Type: "Organization"},
				{Name: "Tim Cook", Type: "Person"},
			},
			Relations: []types.Relation{
				{Subject: "Tim Cook", Predicate: "CEO_of", Object: "Apple"},
			},
			Graph: []types.GraphNode{
				{ID: "node1", Type: "Organization", Label: "Apple", Score: 0.95},
				{ID: "node2", Type: "Person", Label: "Tim Cook", Score: 0.92},
			},
			Duration: 250,
		}

		err := store.SaveSearch(search)
		require.NoError(t, err)

		searches, err := store.GetSearches(requestID)
		require.NoError(t, err)
		require.Equal(t, 1, len(searches))
		assert.Equal(t, 2, len(searches[0].Entities))
		assert.Equal(t, "Apple", searches[0].Entities[0].Name)
		assert.Equal(t, 1, len(searches[0].Relations))
		assert.Equal(t, "CEO_of", searches[0].Relations[0].Predicate)
		assert.Equal(t, 2, len(searches[0].Graph))
	})

	t.Run("SaveSearchWithDSL", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		search := &types.Search{
			RequestID: requestID,
			ChatID:    chat.ChatID,
			Query:     "Find orders over $1000",
			Source:    "db",
			DSL: map[string]any{
				"wheres": []map[string]any{{"column": "amount", "op": ">", "value": 1000}},
				"orders": []map[string]any{{"column": "created_at", "option": "desc"}},
			},
			Duration: 50,
		}

		err := store.SaveSearch(search)
		require.NoError(t, err)

		searches, err := store.GetSearches(requestID)
		require.NoError(t, err)
		require.Equal(t, 1, len(searches))
		require.NotNil(t, searches[0].DSL)

		wheres, ok := searches[0].DSL["wheres"].([]any)
		require.True(t, ok)
		assert.Greater(t, len(wheres), 0)
	})

	t.Run("SaveSearchWithError", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		search := &types.Search{
			RequestID: requestID,
			ChatID:    chat.ChatID,
			Query:     "Failed search",
			Source:    "web",
			Error:     "API rate limit exceeded",
			Duration:  10,
		}

		err := store.SaveSearch(search)
		require.NoError(t, err)

		searches, err := store.GetSearches(requestID)
		require.NoError(t, err)
		require.Equal(t, 1, len(searches))
		assert.Equal(t, "API rate limit exceeded", searches[0].Error)
	})

	t.Run("SaveSearchWithoutRequestID", func(t *testing.T) {
		search := &types.Search{ChatID: chat.ChatID, Query: "Test", Source: "web"}
		err := store.SaveSearch(search)
		assert.Error(t, err)
	})

	t.Run("SaveSearchWithoutChatID", func(t *testing.T) {
		search := &types.Search{RequestID: "req_test", Query: "Test", Source: "web"}
		err := store.SaveSearch(search)
		assert.Error(t, err)
	})

	t.Run("SaveSearchWithoutSource", func(t *testing.T) {
		search := &types.Search{RequestID: "req_test", ChatID: chat.ChatID, Query: "Test"}
		err := store.SaveSearch(search)
		assert.Error(t, err)
	})

	t.Run("SaveNilSearch", func(t *testing.T) {
		err := store.SaveSearch(nil)
		assert.Error(t, err)
	})
}

func TestGetSearches(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	chat := &types.Chat{AssistantID: "test_assistant"}
	err = store.CreateChat(chat)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

	t.Run("GetMultipleSearches", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

		for i := 1; i <= 3; i++ {
			search := &types.Search{
				RequestID: requestID,
				ChatID:    chat.ChatID,
				Query:     fmt.Sprintf("Query %d", i),
				Source:    "web",
				Duration:  int64(i * 100),
			}
			err := store.SaveSearch(search)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond)
		}
		t.Cleanup(func() { store.DeleteSearches(chat.ChatID) })

		searches, err := store.GetSearches(requestID)
		require.NoError(t, err)
		assert.Equal(t, 3, len(searches))

		for i := 0; i < len(searches)-1; i++ {
			assert.False(t, searches[i].CreatedAt.After(searches[i+1].CreatedAt))
		}
	})

	t.Run("GetSearchesForNonExistentRequest", func(t *testing.T) {
		searches, err := store.GetSearches("nonexistent_request")
		require.NoError(t, err)
		assert.Equal(t, 0, len(searches))
	})

	t.Run("GetSearchesWithEmptyRequestID", func(t *testing.T) {
		_, err := store.GetSearches("")
		assert.Error(t, err)
	})
}

func TestGetReference(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	chat := &types.Chat{AssistantID: "test_assistant"}
	err = store.CreateChat(chat)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	search := &types.Search{
		RequestID: requestID,
		ChatID:    chat.ChatID,
		Query:     "Test query",
		Source:    "web",
		References: []types.Reference{
			{Index: 1, Type: "web", Title: "Reference 1", URL: "https://example.com/1"},
			{Index: 2, Type: "web", Title: "Reference 2", URL: "https://example.com/2"},
			{Index: 3, Type: "kb", Title: "Reference 3", Content: "KB content"},
		},
		Duration: 100,
	}
	err = store.SaveSearch(search)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteSearches(chat.ChatID) })

	t.Run("GetExistingReference", func(t *testing.T) {
		ref, err := store.GetReference(requestID, 1)
		require.NoError(t, err)
		assert.Equal(t, "Reference 1", ref.Title)
		assert.Equal(t, "https://example.com/1", ref.URL)
	})

	t.Run("GetReferenceByIndex", func(t *testing.T) {
		ref, err := store.GetReference(requestID, 3)
		require.NoError(t, err)
		assert.Equal(t, "kb", ref.Type)
		assert.Equal(t, "KB content", ref.Content)
	})

	t.Run("GetNonExistentReference", func(t *testing.T) {
		_, err := store.GetReference(requestID, 999)
		assert.Error(t, err)
	})

	t.Run("GetReferenceWithInvalidIndex", func(t *testing.T) {
		_, err := store.GetReference(requestID, 0)
		assert.Error(t, err)

		_, err = store.GetReference(requestID, -1)
		assert.Error(t, err)
	})

	t.Run("GetReferenceWithEmptyRequestID", func(t *testing.T) {
		_, err := store.GetReference("", 1)
		assert.Error(t, err)
	})
}

func TestDeleteSearches(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("DeleteSearchesForChat", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		for i := 1; i <= 3; i++ {
			requestID := fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), i)
			search := &types.Search{
				RequestID: requestID,
				ChatID:    chat.ChatID,
				Query:     fmt.Sprintf("Query %d", i),
				Source:    "web",
				Duration:  100,
			}
			err := store.SaveSearch(search)
			require.NoError(t, err)
		}

		err = store.DeleteSearches(chat.ChatID)
		require.NoError(t, err)
	})

	t.Run("DeleteSearchesWithEmptyChatID", func(t *testing.T) {
		err := store.DeleteSearches("")
		assert.Error(t, err)
	})
}

func TestSearchCompleteWorkflow(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("CompleteWorkflow", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "workflow_assistant", Title: "Search Workflow Test"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

		search := &types.Search{
			RequestID: requestID,
			ChatID:    chat.ChatID,
			Query:     "What are the best practices for Go programming?",
			Config: map[string]any{
				"uses": map[string]any{"search": "builtin", "web": "builtin"},
				"web":  map[string]any{"provider": "tavily", "max_results": 5},
			},
			Keywords: []string{"Go", "programming", "best practices"},
			Source:   "auto",
			References: []types.Reference{
				{Index: 1, Type: "web", Title: "Effective Go", URL: "https://go.dev/doc/effective_go"},
				{Index: 2, Type: "web", Title: "Go Proverbs", URL: "https://go-proverbs.github.io/"},
				{Index: 3, Type: "kb", Title: "Internal Go Guide", Content: "Our team's Go coding standards..."},
			},
			XML:      `<references><ref index="1">...</ref></references>`,
			Prompt:   "When citing, use [1], [2], [3] format.",
			Duration: 350,
		}

		err = store.SaveSearch(search)
		require.NoError(t, err)

		searches, err := store.GetSearches(requestID)
		require.NoError(t, err)
		require.Equal(t, 1, len(searches))

		s := searches[0]
		assert.Equal(t, "What are the best practices for Go programming?", s.Query)
		assert.Equal(t, 3, len(s.Keywords))
		assert.Equal(t, 3, len(s.References))
		assert.NotNil(t, s.Config)

		ref, err := store.GetReference(requestID, 2)
		require.NoError(t, err)
		assert.Equal(t, "Go Proverbs", ref.Title)

		err = store.DeleteSearches(chat.ChatID)
		require.NoError(t, err)

		deletedSearches, err := store.GetSearches(requestID)
		require.NoError(t, err)
		assert.Equal(t, 0, len(deletedSearches))
	})
}
