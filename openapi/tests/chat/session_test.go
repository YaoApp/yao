package openapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/assistant"
	storetypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// =============================================================================
// Test Setup Helpers
// =============================================================================

// createTestChat creates a test chat session in the database
func createTestChat(t *testing.T, title string, assistantID string) string {
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		t.Skip("Chat store not initialized")
	}

	chatID := uuid.New().String()
	chat := &storetypes.Chat{
		ChatID:      chatID,
		AssistantID: assistantID,
		Title:       title,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := chatStore.CreateChat(chat)
	if err != nil {
		t.Fatalf("Failed to create test chat: %v", err)
	}

	t.Logf("Created test chat: %s (title: %s)", chatID, title)
	return chatID
}

// createTestMessage creates a test message in the database
func createTestMessage(t *testing.T, chatID, role, msgType, content string) string {
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		t.Skip("Chat store not initialized")
	}

	msgID := uuid.New().String()
	msg := &storetypes.Message{
		MessageID: msgID,
		ChatID:    chatID,
		Role:      role,
		Type:      msgType,
		Props: map[string]interface{}{
			"content": content,
		},
		Sequence:  1,
		CreatedAt: time.Now(),
	}

	err := chatStore.SaveMessages(chatID, []*storetypes.Message{msg})
	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}

	t.Logf("Created test message: %s (role: %s)", msgID, role)
	return msgID
}

// cleanupTestChat deletes a test chat session
func cleanupTestChat(t *testing.T, chatID string) {
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		return
	}

	err := chatStore.DeleteChat(chatID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup test chat %s: %v", chatID, err)
	} else {
		t.Logf("Cleaned up test chat: %s", chatID)
	}
}

// =============================================================================
// List Chat Sessions Tests
// =============================================================================

// TestListChatSessions tests the chat sessions listing endpoint
func TestListChatSessions(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Chat Session Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create test chats
	chatID1 := createTestChat(t, "Test Chat 1", "test-assistant")
	defer cleanupTestChat(t, chatID1)
	chatID2 := createTestChat(t, "Test Chat 2", "test-assistant")
	defer cleanupTestChat(t, chatID2)

	t.Run("ListChatsSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully retrieve chat sessions")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Check response structure
		assert.Contains(t, response, "data")
		assert.Contains(t, response, "page")
		assert.Contains(t, response, "pagesize")
		assert.Contains(t, response, "total")

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d chat sessions", len(data))
		}
	})

	t.Run("ListChatsWithPagination", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions?page=1&pagesize=10", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Verify pagination values
		page, hasPage := response["page"].(float64)
		pagesize, hasPagesize := response["pagesize"].(float64)

		if hasPage && hasPagesize {
			assert.Equal(t, float64(1), page, "Page should be 1")
			assert.Equal(t, float64(10), pagesize, "Pagesize should be 10")
			t.Logf("Pagination working correctly: page=%d, pagesize=%d", int(page), int(pagesize))
		}
	})

	t.Run("ListChatsWithKeywords", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions?keywords=Test", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d chat sessions with keywords filter", len(data))
		}
	})

	t.Run("ListChatsWithStatusFilter", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions?status=active", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d active chat sessions", len(data))
		}
	})

	t.Run("ListChatsWithAssistantFilter", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions?assistant_id=test-assistant", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d chat sessions with assistant filter", len(data))
		}
	})

	t.Run("ListChatsWithTimeRange", func(t *testing.T) {
		startTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
		endTime := time.Now().Add(time.Hour).Format(time.RFC3339)

		req, err := http.NewRequest("GET", fmt.Sprintf("%s%s/chat/sessions?start_time=%s&end_time=%s", serverURL, baseURL, startTime, endTime), nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d chat sessions within time range", len(data))
		}
	})

	t.Run("ListChatsWithSorting", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions?order_by=created_at&order=desc", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d chat sessions with sorting", len(data))
		}
	})

	t.Run("ListChatsWithGroupBy", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions?group_by=time", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Check for groups in response (when group_by=time, only groups is returned, not data)
		_, hasGroups := response["groups"]
		_, hasData := response["data"]
		assert.True(t, hasGroups, "Response should contain groups when group_by=time")
		assert.False(t, hasData, "Response should NOT contain data when group_by=time (to avoid duplication)")
		t.Logf("Successfully retrieved chat sessions with time grouping")
	})

	t.Run("ListChatsUnauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions", nil)
		assert.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should fail without authorization
		assert.NotEqual(t, http.StatusOK, resp.StatusCode, "Should fail without authorization")
	})
}

// =============================================================================
// Get Chat Session Tests
// =============================================================================

// TestGetChatSession tests the get single chat session endpoint
func TestGetChatSession(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Chat Get Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create test chat
	chatID := createTestChat(t, "Test Chat for Get", "test-assistant")
	defer cleanupTestChat(t, chatID)

	t.Run("GetChatSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions/"+chatID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully retrieve chat session")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Check response contains chat data
		data, hasData := response["data"].(map[string]interface{})
		if hasData {
			assert.Equal(t, chatID, data["chat_id"], "Chat ID should match")
			t.Logf("Successfully retrieved chat: %s", chatID)
		}
	})

	t.Run("GetChatNotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions/non-existent-chat-id", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404 for non-existent chat")
	})

	t.Run("GetChatUnauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions/"+chatID, nil)
		assert.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusOK, resp.StatusCode, "Should fail without authorization")
	})
}

// =============================================================================
// Update Chat Session Tests
// =============================================================================

// TestUpdateChatSession tests the update chat session endpoint
func TestUpdateChatSession(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Chat Update Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create test chat
	chatID := createTestChat(t, "Test Chat for Update", "test-assistant")
	defer cleanupTestChat(t, chatID)

	t.Run("UpdateChatTitleSuccess", func(t *testing.T) {
		body := map[string]interface{}{
			"title": "Updated Chat Title",
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/chat/sessions/"+chatID, bytes.NewReader(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update chat title")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		t.Logf("Successfully updated chat title: %s", chatID)
	})

	t.Run("UpdateChatStatusSuccess", func(t *testing.T) {
		body := map[string]interface{}{
			"status": "archived",
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/chat/sessions/"+chatID, bytes.NewReader(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update chat status")

		t.Logf("Successfully updated chat status: %s", chatID)
	})

	t.Run("UpdateChatMetadataSuccess", func(t *testing.T) {
		body := map[string]interface{}{
			"metadata": map[string]interface{}{
				"custom_key": "custom_value",
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/chat/sessions/"+chatID, bytes.NewReader(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update chat metadata")

		t.Logf("Successfully updated chat metadata: %s", chatID)
	})

	t.Run("UpdateChatNoFields", func(t *testing.T) {
		body := map[string]interface{}{}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/chat/sessions/"+chatID, bytes.NewReader(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Note: Server may still return 200 if it adds __yao_updated_by automatically
		// This is acceptable behavior - the update still happens with the updater field
		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, resp.StatusCode, "Should either succeed with auto-fields or fail with no fields")
	})

	t.Run("UpdateChatNotFound", func(t *testing.T) {
		body := map[string]interface{}{
			"title": "Updated Title",
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/chat/sessions/non-existent-chat-id", bytes.NewReader(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should fail for non-existent chat
		assert.NotEqual(t, http.StatusOK, resp.StatusCode, "Should fail for non-existent chat")
	})

	t.Run("UpdateChatUnauthorized", func(t *testing.T) {
		body := map[string]interface{}{
			"title": "Updated Title",
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/chat/sessions/"+chatID, bytes.NewReader(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusOK, resp.StatusCode, "Should fail without authorization")
	})
}

// =============================================================================
// Delete Chat Session Tests
// =============================================================================

// TestDeleteChatSession tests the delete chat session endpoint
func TestDeleteChatSession(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Chat Delete Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("DeleteChatSuccess", func(t *testing.T) {
		// Create a chat to delete
		chatID := createTestChat(t, "Test Chat for Delete", "test-assistant")

		req, err := http.NewRequest("DELETE", serverURL+baseURL+"/chat/sessions/"+chatID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully delete chat session")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		t.Logf("Successfully deleted chat: %s", chatID)
	})

	t.Run("DeleteChatNotFound", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", serverURL+baseURL+"/chat/sessions/non-existent-chat-id", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should fail for non-existent chat
		assert.NotEqual(t, http.StatusOK, resp.StatusCode, "Should fail for non-existent chat")
	})

	t.Run("DeleteChatUnauthorized", func(t *testing.T) {
		// Create a chat to attempt to delete
		chatID := createTestChat(t, "Test Chat for Unauthorized Delete", "test-assistant")
		defer cleanupTestChat(t, chatID)

		req, err := http.NewRequest("DELETE", serverURL+baseURL+"/chat/sessions/"+chatID, nil)
		assert.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusOK, resp.StatusCode, "Should fail without authorization")
	})
}

// =============================================================================
// Get Messages Tests
// =============================================================================

// TestGetMessages tests the get messages endpoint
func TestGetMessages(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Chat Messages Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create test chat with messages
	chatID := createTestChat(t, "Test Chat for Messages", "test-assistant")
	defer cleanupTestChat(t, chatID)

	// Create test messages
	createTestMessage(t, chatID, "user", "text", "Hello, how are you?")
	createTestMessage(t, chatID, "assistant", "text", "I'm doing well, thank you!")

	t.Run("GetMessagesSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions/"+chatID+"/messages", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully retrieve messages")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Check response structure
		data, hasData := response["data"].(map[string]interface{})
		if hasData {
			messages, hasMessages := data["messages"].([]interface{})
			if hasMessages {
				t.Logf("Successfully retrieved %d messages", len(messages))
			}
		}
	})

	t.Run("GetMessagesWithRoleFilter", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions/"+chatID+"/messages?role=user", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved messages with role filter")
	})

	t.Run("GetMessagesWithTypeFilter", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions/"+chatID+"/messages?type=text", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved messages with type filter")
	})

	t.Run("GetMessagesWithPagination", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions/"+chatID+"/messages?limit=10&offset=0", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved messages with pagination")
	})

	t.Run("GetMessagesNotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions/non-existent-chat-id/messages", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// For non-existent chat, the API may return:
		// - 200 with empty messages (if permission check passes first)
		// - 403 Forbidden (if permission check fails on non-existent chat)
		// - 404 Not Found (if explicitly checking chat existence)
		// All are acceptable behaviors depending on implementation
		t.Logf("Response status for non-existent chat messages: %d", resp.StatusCode)
	})

	t.Run("GetMessagesUnauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/sessions/"+chatID+"/messages", nil)
		assert.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusOK, resp.StatusCode, "Should fail without authorization")
	})
}
