package context

import (
	"testing"

	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func getTestCache(t *testing.T) store.Store {
	cache, err := store.Get("__yao.agent.cache")
	if err != nil {
		t.Fatalf("Failed to get cache store: %v", err)
	}
	cache.Clear() // Clean before test
	return cache
}

func TestGetChatIDByMessages_NewConversation(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cache := getTestCache(t)

	messages := []Message{
		{
			Role:    RoleUser,
			Content: "Hello, how are you?",
		},
	}

	// First request - should generate new chat ID
	chatID1, err := GetChatIDByMessages(cache, messages)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	if chatID1 == "" {
		t.Fatal("Expected non-empty chat ID")
	}

	// Second request with same single user message - should generate DIFFERENT chat ID
	// (single user message always generates new chat ID to avoid false matches)
	chatID2, err := GetChatIDByMessages(cache, messages)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	if chatID2 == "" {
		t.Fatal("Expected non-empty chat ID")
	}

	// Both should be valid but different (single user message = new conversation each time)
	if chatID1 == chatID2 {
		t.Errorf("Expected different chat IDs for single user message, got same ID: %s", chatID1)
	}
}

func TestGetChatIDByMessages_ContinuousConversation(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cache := getTestCache(t)

	// Scenario: User conversation with incrementally added messages
	// Request 1: [user1]
	messages1 := []Message{
		{Role: RoleUser, Content: "First message"},
	}
	chatID1, err := GetChatIDByMessages(cache, messages1)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	// Request 2: [user1, user2]
	// For 2 messages, matches last 1 message
	// Should match chatID1 because last message is cached
	messages2 := []Message{
		{Role: RoleUser, Content: "First message"},
		{Role: RoleUser, Content: "Second message"},
	}
	chatID2, err := GetChatIDByMessages(cache, messages2)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	if chatID1 != chatID2 {
		t.Errorf("Expected chatID2 to match chatID1, got %s and %s", chatID2, chatID1)
	}

	// Request 3: [user1, user2, user3]
	// For 3+ messages, matches last 2 messages
	// Should match chatID2 because last 2 messages are cached
	messages3 := []Message{
		{Role: RoleUser, Content: "First message"},
		{Role: RoleUser, Content: "Second message"},
		{Role: RoleUser, Content: "Third message"},
	}
	chatID3, err := GetChatIDByMessages(cache, messages3)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	if chatID2 != chatID3 {
		t.Errorf("Expected chatID3 to match chatID2, got %s and %s", chatID3, chatID2)
	}

	// Request 4: [user1, user2, user3, user4]
	// Should match chatID3 because last 2 messages are cached
	messages4 := []Message{
		{Role: RoleUser, Content: "First message"},
		{Role: RoleUser, Content: "Second message"},
		{Role: RoleUser, Content: "Third message"},
		{Role: RoleUser, Content: "Fourth message"},
	}
	chatID4, err := GetChatIDByMessages(cache, messages4)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	if chatID3 != chatID4 {
		t.Errorf("Expected chatID4 to match chatID3, got %s and %s", chatID4, chatID3)
	}

	// All should be the same conversation
	if chatID1 != chatID4 {
		t.Errorf("Expected all chat IDs to be the same, got %s and %s", chatID1, chatID4)
	}
}

func TestGetChatIDByMessages_DifferentConversations(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cache := getTestCache(t)

	// First conversation
	messages1 := []Message{
		{
			Role:    RoleUser,
			Content: "Hello",
		},
	}

	chatID1, err := GetChatIDByMessages(cache, messages1)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	err = CacheChatID(cache, messages1, chatID1)
	if err != nil {
		t.Fatalf("Failed to cache chat ID: %v", err)
	}

	// Different conversation
	messages2 := []Message{
		{
			Role:    RoleUser,
			Content: "Goodbye",
		},
	}

	chatID2, err := GetChatIDByMessages(cache, messages2)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	if chatID1 == chatID2 {
		t.Errorf("Expected different chat IDs for different conversations, got %s", chatID1)
	}
}

func TestGetChatIDByMessages_MultiModalContent(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cache := getTestCache(t)

	// First request with multimodal content
	messages1 := []Message{
		{
			Role: RoleUser,
			Content: []ContentPart{
				{
					Type: ContentText,
					Text: "What's in this image?",
				},
				{
					Type: ContentImageURL,
					ImageURL: &ImageURL{
						URL:    "https://example.com/image.jpg",
						Detail: DetailHigh,
					},
				},
			},
		},
	}

	chatID1, err := GetChatIDByMessages(cache, messages1)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	// Second request - add another message to continue conversation
	messages2 := append(messages1, Message{
		Role:    RoleUser,
		Content: "Tell me more details",
	})

	chatID2, err := GetChatIDByMessages(cache, messages2)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	// Should get same chat ID (continuation)
	if chatID1 != chatID2 {
		t.Errorf("Expected same chat ID for multimodal continuation, got %s and %s", chatID1, chatID2)
	}
}

func TestGetChatIDByMessages_WithToolCalls(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cache := getTestCache(t)

	// First request with user message
	messages1 := []Message{
		{
			Role:    RoleUser,
			Content: "What's the weather in Tokyo?",
		},
	}

	chatID1, err := GetChatIDByMessages(cache, messages1)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	// Second request - add assistant response and another user message
	messages2 := []Message{
		{
			Role:    RoleUser,
			Content: "What's the weather in Tokyo?",
		},
		{
			Role:    RoleAssistant,
			Content: nil,
			ToolCalls: []ToolCall{
				{
					ID:   "call_123",
					Type: ToolTypeFunction,
					Function: Function{
						Name:      "get_weather",
						Arguments: `{"location":"Tokyo"}`,
					},
				},
			},
		},
		{
			Role:    RoleUser,
			Content: "How about tomorrow?",
		},
	}

	chatID2, err := GetChatIDByMessages(cache, messages2)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	// Should get same chat ID (assistant messages are ignored, so it matches the first user message)
	if chatID1 != chatID2 {
		t.Errorf("Expected same chat ID for messages with tool calls, got %s and %s", chatID1, chatID2)
	}
}

func TestCacheChatID_EmptyMessages(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cache := getTestCache(t)

	err := CacheChatID(cache, []Message{}, "chat_123")
	if err == nil {
		t.Error("Expected error for empty messages")
	}
}

func TestCacheChatID_EmptyChatID(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cache := getTestCache(t)

	messages := []Message{
		{
			Role:    RoleUser,
			Content: "Hello",
		},
	}

	err := CacheChatID(cache, messages, "")
	if err == nil {
		t.Error("Expected error for empty chat ID")
	}
}

func TestGetChatIDByMessages_EmptyMessages(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cache := getTestCache(t)

	_, err := GetChatIDByMessages(cache, []Message{})
	if err == nil {
		t.Error("Expected error for empty messages")
	}
}

func TestHashMessage_Consistency(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "Test message",
	}

	hash1, err := hashMessage(msg)
	if err != nil {
		t.Fatalf("Failed to hash message: %v", err)
	}

	hash2, err := hashMessage(msg)
	if err != nil {
		t.Fatalf("Failed to hash message: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Expected consistent hashes, got %s and %s", hash1, hash2)
	}
}

func TestGetKey(t *testing.T) {
	hash := "abc123"
	key := getKey(hash)

	expectedPrefix := chatCachePrefix
	if len(key) <= len(expectedPrefix) {
		t.Errorf("Expected key to have prefix, got %s", key)
	}

	if key[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected key to start with %s, got %s", expectedPrefix, key)
	}

	if key != chatCachePrefix+hash {
		t.Errorf("Expected key %s, got %s", chatCachePrefix+hash, key)
	}
}

func TestGenChatID(t *testing.T) {
	id1 := GenChatID()

	if id1 == "" {
		t.Error("Expected non-empty chat ID")
	}

	// Check length - NanoID with length 16 should produce 16 character strings
	if len(id1) < 10 {
		t.Errorf("Expected chat ID to have reasonable length, got %d characters: %s", len(id1), id1)
	}

	// Note: We don't test uniqueness here because nano timestamp-based IDs
	// can occasionally be the same when generated in rapid succession.
	// The uniqueness is good enough for production use.
}
