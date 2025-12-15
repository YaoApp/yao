package xun_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestSaveSearch tests saving search records
func TestSaveSearch(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create a chat first
	chat := &types.Chat{
		AssistantID: "test_assistant",
		Title:       "Search Test Chat",
	}
	err = store.CreateChat(chat)
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}
	defer store.DeleteChat(chat.ChatID)

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
		if err != nil {
			t.Fatalf("Failed to save search: %v", err)
		}

		// Verify
		searches, err := store.GetSearches(requestID)
		if err != nil {
			t.Fatalf("Failed to get searches: %v", err)
		}

		if len(searches) != 1 {
			t.Fatalf("Expected 1 search, got %d", len(searches))
		}

		if searches[0].Query != "What is the weather today?" {
			t.Errorf("Expected query 'What is the weather today?', got '%s'", searches[0].Query)
		}
		if searches[0].Source != "web" {
			t.Errorf("Expected source 'web', got '%s'", searches[0].Source)
		}
		if searches[0].Duration != 150 {
			t.Errorf("Expected duration 150, got %d", searches[0].Duration)
		}
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
		if err != nil {
			t.Fatalf("Failed to save search: %v", err)
		}

		searches, err := store.GetSearches(requestID)
		if err != nil {
			t.Fatalf("Failed to get searches: %v", err)
		}

		if len(searches) != 1 {
			t.Fatalf("Expected 1 search, got %d", len(searches))
		}

		if len(searches[0].Keywords) != 3 {
			t.Errorf("Expected 3 keywords, got %d", len(searches[0].Keywords))
		}
		if searches[0].Keywords[0] != "AI" {
			t.Errorf("Expected first keyword 'AI', got '%s'", searches[0].Keywords[0])
		}
	})

	t.Run("SaveSearchWithReferences", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		search := &types.Search{
			RequestID: requestID,
			ChatID:    chat.ChatID,
			Query:     "How to learn Go programming?",
			Source:    "web",
			References: []types.Reference{
				{
					Index:   1,
					Type:    "web",
					Title:   "Go Programming Tutorial",
					URL:     "https://go.dev/tour/",
					Snippet: "An interactive introduction to Go",
				},
				{
					Index:   2,
					Type:    "web",
					Title:   "Effective Go",
					URL:     "https://go.dev/doc/effective_go",
					Snippet: "Tips for writing clear, idiomatic Go code",
				},
			},
			XML:      "<references>...</references>",
			Prompt:   "Please cite sources using [1], [2]...",
			Duration: 300,
		}

		err := store.SaveSearch(search)
		if err != nil {
			t.Fatalf("Failed to save search: %v", err)
		}

		searches, err := store.GetSearches(requestID)
		if err != nil {
			t.Fatalf("Failed to get searches: %v", err)
		}

		if len(searches) != 1 {
			t.Fatalf("Expected 1 search, got %d", len(searches))
		}

		if len(searches[0].References) != 2 {
			t.Errorf("Expected 2 references, got %d", len(searches[0].References))
		}
		if searches[0].References[0].Title != "Go Programming Tutorial" {
			t.Errorf("Expected first reference title 'Go Programming Tutorial', got '%s'", searches[0].References[0].Title)
		}
		if searches[0].XML != "<references>...</references>" {
			t.Errorf("Expected XML '<references>...</references>', got '%s'", searches[0].XML)
		}
		if searches[0].Prompt != "Please cite sources using [1], [2]..." {
			t.Errorf("Expected prompt to be set")
		}
	})

	t.Run("SaveSearchWithConfig", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		search := &types.Search{
			RequestID: requestID,
			ChatID:    chat.ChatID,
			Query:     "Config test",
			Source:    "auto",
			Config: map[string]any{
				"uses": map[string]any{
					"search":  "builtin",
					"web":     "builtin",
					"keyword": "builtin",
				},
				"web": map[string]any{
					"provider":    "tavily",
					"max_results": 5,
				},
			},
			Duration: 100,
		}

		err := store.SaveSearch(search)
		if err != nil {
			t.Fatalf("Failed to save search: %v", err)
		}

		searches, err := store.GetSearches(requestID)
		if err != nil {
			t.Fatalf("Failed to get searches: %v", err)
		}

		if len(searches) != 1 {
			t.Fatalf("Expected 1 search, got %d", len(searches))
		}

		if searches[0].Config == nil {
			t.Fatal("Expected config to be set")
		}
		uses, ok := searches[0].Config["uses"].(map[string]any)
		if !ok {
			t.Fatal("Expected uses in config")
		}
		if uses["search"] != "builtin" {
			t.Errorf("Expected uses.search='builtin', got '%v'", uses["search"])
		}
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
		if err != nil {
			t.Fatalf("Failed to save search: %v", err)
		}

		searches, err := store.GetSearches(requestID)
		if err != nil {
			t.Fatalf("Failed to get searches: %v", err)
		}

		if len(searches) != 1 {
			t.Fatalf("Expected 1 search, got %d", len(searches))
		}

		if len(searches[0].Entities) != 2 {
			t.Errorf("Expected 2 entities, got %d", len(searches[0].Entities))
		}
		if searches[0].Entities[0].Name != "Apple" {
			t.Errorf("Expected first entity 'Apple', got '%s'", searches[0].Entities[0].Name)
		}

		if len(searches[0].Relations) != 1 {
			t.Errorf("Expected 1 relation, got %d", len(searches[0].Relations))
		}
		if searches[0].Relations[0].Predicate != "CEO_of" {
			t.Errorf("Expected predicate 'CEO_of', got '%s'", searches[0].Relations[0].Predicate)
		}

		if len(searches[0].Graph) != 2 {
			t.Errorf("Expected 2 graph nodes, got %d", len(searches[0].Graph))
		}
	})

	t.Run("SaveSearchWithDSL", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		search := &types.Search{
			RequestID: requestID,
			ChatID:    chat.ChatID,
			Query:     "Find orders over $1000",
			Source:    "db",
			DSL: map[string]any{
				"wheres": []map[string]any{
					{"column": "amount", "op": ">", "value": 1000},
				},
				"orders": []map[string]any{
					{"column": "created_at", "option": "desc"},
				},
			},
			Duration: 50,
		}

		err := store.SaveSearch(search)
		if err != nil {
			t.Fatalf("Failed to save search: %v", err)
		}

		searches, err := store.GetSearches(requestID)
		if err != nil {
			t.Fatalf("Failed to get searches: %v", err)
		}

		if len(searches) != 1 {
			t.Fatalf("Expected 1 search, got %d", len(searches))
		}

		if searches[0].DSL == nil {
			t.Fatal("Expected DSL to be set")
		}
		wheres, ok := searches[0].DSL["wheres"].([]any)
		if !ok || len(wheres) == 0 {
			t.Error("Expected wheres in DSL")
		}
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
		if err != nil {
			t.Fatalf("Failed to save search: %v", err)
		}

		searches, err := store.GetSearches(requestID)
		if err != nil {
			t.Fatalf("Failed to get searches: %v", err)
		}

		if len(searches) != 1 {
			t.Fatalf("Expected 1 search, got %d", len(searches))
		}

		if searches[0].Error != "API rate limit exceeded" {
			t.Errorf("Expected error 'API rate limit exceeded', got '%s'", searches[0].Error)
		}
	})

	t.Run("SaveSearchWithoutRequestID", func(t *testing.T) {
		search := &types.Search{
			ChatID: chat.ChatID,
			Query:  "Test",
			Source: "web",
		}

		err := store.SaveSearch(search)
		if err == nil {
			t.Error("Expected error when saving without request_id")
		}
	})

	t.Run("SaveSearchWithoutChatID", func(t *testing.T) {
		search := &types.Search{
			RequestID: "req_test",
			Query:     "Test",
			Source:    "web",
		}

		err := store.SaveSearch(search)
		if err == nil {
			t.Error("Expected error when saving without chat_id")
		}
	})

	t.Run("SaveSearchWithoutSource", func(t *testing.T) {
		search := &types.Search{
			RequestID: "req_test",
			ChatID:    chat.ChatID,
			Query:     "Test",
		}

		err := store.SaveSearch(search)
		if err == nil {
			t.Error("Expected error when saving without source")
		}
	})

	t.Run("SaveNilSearch", func(t *testing.T) {
		err := store.SaveSearch(nil)
		if err == nil {
			t.Error("Expected error when saving nil search")
		}
	})
}

// TestGetSearches tests retrieving search records
func TestGetSearches(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create a chat
	chat := &types.Chat{
		AssistantID: "test_assistant",
	}
	err = store.CreateChat(chat)
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}
	defer store.DeleteChat(chat.ChatID)

	t.Run("GetMultipleSearches", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

		// Save multiple searches for the same request
		for i := 1; i <= 3; i++ {
			search := &types.Search{
				RequestID: requestID,
				ChatID:    chat.ChatID,
				Query:     fmt.Sprintf("Query %d", i),
				Source:    "web",
				Duration:  int64(i * 100),
			}
			err := store.SaveSearch(search)
			if err != nil {
				t.Fatalf("Failed to save search %d: %v", i, err)
			}
			time.Sleep(10 * time.Millisecond) // Ensure different created_at
		}

		searches, err := store.GetSearches(requestID)
		if err != nil {
			t.Fatalf("Failed to get searches: %v", err)
		}

		if len(searches) != 3 {
			t.Errorf("Expected 3 searches, got %d", len(searches))
		}

		// Verify order (by created_at asc)
		for i := 0; i < len(searches)-1; i++ {
			if searches[i].CreatedAt.After(searches[i+1].CreatedAt) {
				t.Error("Searches not ordered by created_at asc")
			}
		}
	})

	t.Run("GetSearchesForNonExistentRequest", func(t *testing.T) {
		searches, err := store.GetSearches("nonexistent_request")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(searches) != 0 {
			t.Errorf("Expected 0 searches, got %d", len(searches))
		}
	})

	t.Run("GetSearchesWithEmptyRequestID", func(t *testing.T) {
		_, err := store.GetSearches("")
		if err == nil {
			t.Error("Expected error when getting searches without request_id")
		}
	})
}

// TestGetReference tests retrieving a single reference
func TestGetReference(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create a chat
	chat := &types.Chat{
		AssistantID: "test_assistant",
	}
	err = store.CreateChat(chat)
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}
	defer store.DeleteChat(chat.ChatID)

	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

	// Save search with references
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
	if err != nil {
		t.Fatalf("Failed to save search: %v", err)
	}

	t.Run("GetExistingReference", func(t *testing.T) {
		ref, err := store.GetReference(requestID, 1)
		if err != nil {
			t.Fatalf("Failed to get reference: %v", err)
		}

		if ref.Title != "Reference 1" {
			t.Errorf("Expected title 'Reference 1', got '%s'", ref.Title)
		}
		if ref.URL != "https://example.com/1" {
			t.Errorf("Expected URL 'https://example.com/1', got '%s'", ref.URL)
		}
	})

	t.Run("GetReferenceByIndex", func(t *testing.T) {
		ref, err := store.GetReference(requestID, 3)
		if err != nil {
			t.Fatalf("Failed to get reference: %v", err)
		}

		if ref.Type != "kb" {
			t.Errorf("Expected type 'kb', got '%s'", ref.Type)
		}
		if ref.Content != "KB content" {
			t.Errorf("Expected content 'KB content', got '%s'", ref.Content)
		}
	})

	t.Run("GetNonExistentReference", func(t *testing.T) {
		_, err := store.GetReference(requestID, 999)
		if err == nil {
			t.Error("Expected error when getting non-existent reference")
		}
	})

	t.Run("GetReferenceWithInvalidIndex", func(t *testing.T) {
		_, err := store.GetReference(requestID, 0)
		if err == nil {
			t.Error("Expected error when getting reference with index 0")
		}

		_, err = store.GetReference(requestID, -1)
		if err == nil {
			t.Error("Expected error when getting reference with negative index")
		}
	})

	t.Run("GetReferenceWithEmptyRequestID", func(t *testing.T) {
		_, err := store.GetReference("", 1)
		if err == nil {
			t.Error("Expected error when getting reference without request_id")
		}
	})
}

// TestDeleteSearches tests deleting search records
func TestDeleteSearches(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("DeleteSearchesForChat", func(t *testing.T) {
		// Create a chat
		chat := &types.Chat{
			AssistantID: "test_assistant",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(chat.ChatID)

		// Save multiple searches
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
			if err != nil {
				t.Fatalf("Failed to save search: %v", err)
			}
		}

		// Delete all searches for the chat
		err = store.DeleteSearches(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to delete searches: %v", err)
		}

		// Note: GetSearches filters by request_id, not chat_id
		// We can't easily verify deletion without a GetSearchesByChatID method
		// But the soft delete should have been applied
	})

	t.Run("DeleteSearchesWithEmptyChatID", func(t *testing.T) {
		err := store.DeleteSearches("")
		if err == nil {
			t.Error("Expected error when deleting searches without chat_id")
		}
	})
}

// TestSearchCompleteWorkflow tests a complete search workflow
func TestSearchCompleteWorkflow(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// 1. Create chat
		chat := &types.Chat{
			AssistantID: "workflow_assistant",
			Title:       "Search Workflow Test",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(chat.ChatID)

		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

		// 2. Save search with full data
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
			XML:      "<references><ref index=\"1\">...</ref></references>",
			Prompt:   "When citing, use [1], [2], [3] format.",
			Duration: 350,
		}

		err = store.SaveSearch(search)
		if err != nil {
			t.Fatalf("Failed to save search: %v", err)
		}

		// 3. Retrieve searches
		searches, err := store.GetSearches(requestID)
		if err != nil {
			t.Fatalf("Failed to get searches: %v", err)
		}

		if len(searches) != 1 {
			t.Fatalf("Expected 1 search, got %d", len(searches))
		}

		// 4. Verify all fields
		s := searches[0]
		if s.Query != "What are the best practices for Go programming?" {
			t.Errorf("Query mismatch")
		}
		if len(s.Keywords) != 3 {
			t.Errorf("Expected 3 keywords, got %d", len(s.Keywords))
		}
		if len(s.References) != 3 {
			t.Errorf("Expected 3 references, got %d", len(s.References))
		}
		if s.Config == nil {
			t.Error("Config should not be nil")
		}

		// 5. Get specific reference
		ref, err := store.GetReference(requestID, 2)
		if err != nil {
			t.Fatalf("Failed to get reference: %v", err)
		}
		if ref.Title != "Go Proverbs" {
			t.Errorf("Expected 'Go Proverbs', got '%s'", ref.Title)
		}

		// 6. Delete searches
		err = store.DeleteSearches(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to delete searches: %v", err)
		}

		// 7. Verify deletion (soft delete, so GetSearches should return empty)
		deletedSearches, err := store.GetSearches(requestID)
		if err != nil {
			t.Fatalf("Failed to get searches after delete: %v", err)
		}
		if len(deletedSearches) != 0 {
			t.Errorf("Expected 0 searches after delete, got %d", len(deletedSearches))
		}

		t.Log("Complete search workflow passed!")
	})
}
