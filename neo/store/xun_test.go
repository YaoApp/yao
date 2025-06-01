package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestNewXunDefault(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_assistant")

	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	if err != nil {
		t.Fatal(err)
	}

	err = capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")
	if err != nil {
		t.Fatal(err)
	}

	err = capsule.Schema().DropTableIfExists("__unit_test_conversation_assistant")
	if err != nil {
		t.Fatal(err)
	}

	// Add a small delay to ensure table is created
	time.Sleep(100 * time.Millisecond)

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})

	if err != nil {
		t.Error(err)
		return
	}

	// Check history table
	has, err := capsule.Schema().HasTable("__unit_test_conversation_history")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, has)

	// Check chat table
	has, err = capsule.Schema().HasTable("__unit_test_conversation_chat")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, has)

	// Check assistant table
	has, err = capsule.Schema().HasTable("__unit_test_conversation_assistant")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, has)

	// Validate table structure by attempting operations
	// Test history operations
	messages := []map[string]interface{}{
		{"role": "user", "content": "test message"},
	}
	err = store.SaveHistory("test_user", messages, "test_chat", nil)
	assert.Nil(t, err)

	history, err := store.GetHistory("test_user", "test_chat")
	assert.Nil(t, err)
	assert.NotEmpty(t, history)

	// Test chat operations
	err = store.UpdateChatTitle("test_user", "test_chat", "Test Chat")
	assert.Nil(t, err)

	chat, err := store.GetChat("test_user", "test_chat")
	assert.Nil(t, err)
	assert.NotNil(t, chat)

	// Test assistant operations
	assistant := map[string]interface{}{
		"name":        "Test Assistant",
		"type":        "assistant",
		"connector":   "test",
		"description": "Test Description",
		"tags":        []string{"test"},
		"mentionable": true,
		"automated":   true,
	}

	id, err := store.SaveAssistant(assistant)
	assert.Nil(t, err)
	assert.NotNil(t, id)

	// Clean up test data
	err = store.DeleteChat("test_user", "test_chat")
	assert.Nil(t, err)

	err = store.DeleteAssistant(id.(string))
	assert.Nil(t, err)
}

func TestNewXunConnector(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("mysql")
	if err != nil {
		t.Fatal(err)
	}

	sch, err := conn.Schema()
	if err != nil {
		t.Fatal(err)
	}

	defer sch.DropTableIfExists("__unit_test_conversation_history")
	defer sch.DropTableIfExists("__unit_test_conversation_chat")
	defer sch.DropTableIfExists("__unit_test_conversation_assistant")
	defer sch.DropTableIfExists("__unit_test_conversation_knowledge")
	defer sch.DropTableIfExists("__unit_test_conversation_attachment")

	sch.DropTableIfExists("__unit_test_conversation_history")
	sch.DropTableIfExists("__unit_test_conversation_chat")
	sch.DropTableIfExists("__unit_test_conversation_assistant")
	sch.DropTableIfExists("__unit_test_conversation_knowledge")
	sch.DropTableIfExists("__unit_test_conversation_attachment")

	// Add a small delay to ensure table is created
	time.Sleep(100 * time.Millisecond)

	store, err := NewXun(Setting{
		Connector: "mysql",
		Prefix:    "__unit_test_conversation_",
	})

	if err != nil {
		t.Error(err)
		return
	}

	// Check history table
	has, err := sch.HasTable("__unit_test_conversation_history")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, has)

	// Check chat table
	has, err = sch.HasTable("__unit_test_conversation_chat")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, has)

	// Check assistant table
	has, err = sch.HasTable("__unit_test_conversation_assistant")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, has)

	// Test basic operations
	messages := []map[string]interface{}{
		{"role": "user", "content": "test message"},
	}
	err = store.SaveHistory("test_user", messages, "test_chat", nil)
	assert.Nil(t, err)

	history, err := store.GetHistory("test_user", "test_chat")
	assert.Nil(t, err)
	assert.NotEmpty(t, history)

	err = store.DeleteChat("test_user", "test_chat")
	assert.Nil(t, err)
}

func TestXunSaveAndGetHistory(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")

	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	if err != nil {
		t.Fatal(err)
	}

	err = capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")
	if err != nil {
		t.Fatal(err)
	}

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
		TTL:       3600,
	})

	// save the history
	cid := "123456"
	err = store.SaveHistory("123456", []map[string]interface{}{
		{"role": "user", "name": "user1", "content": "hello"},
		{"role": "assistant", "name": "user1", "content": "Hello there, how"},
	}, cid, nil)
	assert.Nil(t, err)

	// get the history
	data, err := store.GetHistory("123456", cid)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(data))
}

func TestXunSaveAndGetHistoryWithCID(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")

	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	if err != nil {
		t.Fatal(err)
	}

	err = capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")
	if err != nil {
		t.Fatal(err)
	}

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
		TTL:       3600,
	})

	// save the history with specific cid
	sid := "123456"
	cid := "789012"
	assistantID := "test-assistant-1"
	messages := []map[string]interface{}{
		{"role": "user", "name": "user1", "content": "hello"},
		{"role": "assistant", "name": "assistant1", "content": "Hi! How can I help you?"},
	}
	context := map[string]interface{}{
		"assistant_id": assistantID,
	}
	err = store.SaveHistory(sid, messages, cid, context)
	assert.Nil(t, err)

	// get the history for specific cid
	data, err := store.GetHistory(sid, cid)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(data))

	// Verify assistant_id is saved in chat
	chat, err := store.GetChat(sid, cid)
	assert.Nil(t, err)
	assert.Equal(t, assistantID, chat.Chat["assistant_id"])

	// save another message with different cid and assistant
	anotherCID := "345678"
	anotherAssistantID := "test-assistant-2"
	moreMessages := []map[string]interface{}{
		{"role": "user", "name": "user1", "content": "another message"},
		{"role": "assistant", "name": "assistant2", "content": "Hello!"},
	}
	anotherContext := map[string]interface{}{
		"assistant_id": anotherAssistantID,
	}
	err = store.SaveHistory(sid, moreMessages, anotherCID, anotherContext)
	assert.Nil(t, err)

	// Verify second chat's assistant_id
	chat2, err := store.GetChat(sid, anotherCID)
	assert.Nil(t, err)
	assert.Equal(t, anotherAssistantID, chat2.Chat["assistant_id"])

	// get history for the first cid - should still be 2 messages
	data, err = store.GetHistory(sid, cid)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(data))

	// get history for the second cid - should be 2 messages
	data, err = store.GetHistory(sid, anotherCID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(data))

	// get all history for the sid without specifying cid
	allData, err := store.GetHistory(sid, cid)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(allData))
}

func TestXunGetChats(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_assistant")

	// Drop tables before test
	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	if err != nil {
		t.Fatal(err)
	}
	err = capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")
	if err != nil {
		t.Fatal(err)
	}
	err = capsule.Schema().DropTableIfExists("__unit_test_conversation_assistant")
	if err != nil {
		t.Fatal(err)
	}

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create test assistants first
	assistant1 := map[string]interface{}{
		"assistant_id": "test-assistant-1",
		"name":         "Test Assistant 1",
		"avatar":       "avatar1.png",
		"type":         "assistant",
		"connector":    "test",
	}
	assistant2 := map[string]interface{}{
		"assistant_id": "test-assistant-2",
		"name":         "Test Assistant 2",
		"avatar":       "avatar2.png",
		"type":         "assistant",
		"connector":    "test",
	}
	_, err = store.SaveAssistant(assistant1)
	assert.Nil(t, err)
	_, err = store.SaveAssistant(assistant2)
	assert.Nil(t, err)

	// Save some test chats
	sid := "test_user"
	messages := []map[string]interface{}{
		{"role": "user", "content": "test message"},
	}

	// Create chats with different dates and assistants
	for i := 0; i < 5; i++ {
		chatID := fmt.Sprintf("chat_%d", i)
		title := fmt.Sprintf("Test Chat %d", i)
		var context map[string]interface{}

		// Alternate between having assistant and no assistant
		if i%2 == 0 {
			context = map[string]interface{}{
				"assistant_id": "test-assistant-1",
			}
		} else if i%3 == 0 {
			context = map[string]interface{}{
				"assistant_id": "test-assistant-2",
			}
		}

		// Save history first to create the chat
		err = store.SaveHistory(sid, messages, chatID, context)
		assert.Nil(t, err)

		// Update the chat title
		err = store.UpdateChatTitle(sid, chatID, title)
		assert.Nil(t, err)

		// Verify chat was created with correct assistant info
		chat, err := store.GetChat(sid, chatID)
		assert.Nil(t, err)
		assert.NotNil(t, chat)
		assert.Equal(t, chatID, chat.Chat["chat_id"])
		assert.Equal(t, title, chat.Chat["title"])

		if i%2 == 0 {
			assert.Equal(t, "test-assistant-1", chat.Chat["assistant_id"])
			assert.Equal(t, "Test Assistant 1", chat.Chat["assistant_name"])
			assert.Equal(t, "avatar1.png", chat.Chat["assistant_avatar"])
		} else if i%3 == 0 {
			assert.Equal(t, "test-assistant-2", chat.Chat["assistant_id"])
			assert.Equal(t, "Test Assistant 2", chat.Chat["assistant_name"])
			assert.Equal(t, "avatar2.png", chat.Chat["assistant_avatar"])
		} else {
			assert.Nil(t, chat.Chat["assistant_id"])
			assert.Nil(t, chat.Chat["assistant_name"])
			assert.Nil(t, chat.Chat["assistant_avatar"])
		}
	}

	// Test GetChats
	filter := ChatFilter{
		PageSize: 10,
		Order:    "desc",
	}
	groups, err := store.GetChats(sid, filter)
	assert.Nil(t, err)
	assert.NotNil(t, groups)
	assert.Greater(t, len(groups.Groups), 0)

	// Verify assistant information in chat list
	for _, group := range groups.Groups {
		for _, chat := range group.Chats {
			if assistantID, ok := chat["assistant_id"].(string); ok && assistantID != "" {
				if assistantID == "test-assistant-1" {
					assert.Equal(t, "Test Assistant 1", chat["assistant_name"])
					assert.Equal(t, "avatar1.png", chat["assistant_avatar"])
				} else if assistantID == "test-assistant-2" {
					assert.Equal(t, "Test Assistant 2", chat["assistant_name"])
					assert.Equal(t, "avatar2.png", chat["assistant_avatar"])
				}
			} else {
				assert.Nil(t, chat["assistant_name"])
				assert.Nil(t, chat["assistant_avatar"])
			}
		}
	}

	// Test with keywords
	filter.Keywords = "test"
	groups, err = store.GetChats(sid, filter)
	assert.Nil(t, err)
	assert.Greater(t, len(groups.Groups), 0)
}

func TestXunDeleteChat(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create a test chat
	sid := "test_user"
	cid := "test_chat"
	messages := []map[string]interface{}{
		{"role": "user", "content": "test message"},
	}

	// Save the chat and history
	err = store.SaveHistory(sid, messages, cid, nil)
	assert.Nil(t, err)

	// Verify chat exists
	chat, err := store.GetChat(sid, cid)
	assert.Nil(t, err)
	assert.NotNil(t, chat)

	// Delete the chat
	err = store.DeleteChat(sid, cid)
	assert.Nil(t, err)

	// Verify chat is deleted
	chat, err = store.GetChat(sid, cid)
	assert.Nil(t, err)
	assert.Equal(t, (*ChatInfo)(nil), chat)
}

func TestXunDeleteAllChats(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create multiple test chats
	sid := "test_user"
	messages := []map[string]interface{}{
		{"role": "user", "content": "test message"},
	}

	// Save multiple chats
	for i := 0; i < 3; i++ {
		cid := fmt.Sprintf("test_chat_%d", i)
		err = store.SaveHistory(sid, messages, cid, nil)
		assert.Nil(t, err)
	}

	// Verify chats exist
	response, err := store.GetChats(sid, ChatFilter{})
	assert.Nil(t, err)
	assert.Greater(t, response.Total, int64(0))

	// Delete all chats
	err = store.DeleteAllChats(sid)
	assert.Nil(t, err)

	// Verify all chats are deleted
	response, err = store.GetChats(sid, ChatFilter{})
	assert.Nil(t, err)
	assert.Equal(t, int64(0), response.Total)
}

func TestXunAssistantCRUD(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_assistant")

	// Drop assistant table before test
	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_assistant")
	if err != nil {
		t.Fatal(err)
	}

	// Add a small delay to ensure table is created
	time.Sleep(100 * time.Millisecond)

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Clean up any existing data
	_, err = store.DeleteAssistants(AssistantFilter{})
	assert.Nil(t, err)

	// Test case 1: JSON fields as strings
	tagsJSON := `["tag1", "tag2", "tag3"]`
	optionsJSON := `{"model": "gpt-4"}`
	placeholderJSON := `{"title": "Test Title", "description": "Test Description", "prompts": ["prompt1", "prompt2"]}`
	assistant := map[string]interface{}{
		"name":        "Test Assistant",
		"type":        "assistant",
		"avatar":      "https://example.com/avatar.png",
		"connector":   "openai",
		"description": "Test Description",
		"path":        "/assistants/test",
		"sort":        100,
		"built_in":    true,
		"tags":        tagsJSON,
		"options":     optionsJSON,
		"placeholder": placeholderJSON,
		"mentionable": true,
		"automated":   true,
	}

	// Test SaveAssistant (Create) with string JSON
	v, err := store.SaveAssistant(assistant)
	assert.Nil(t, err)
	assistantID := v.(string)
	assert.NotEmpty(t, assistantID)

	// Test GetAssistant for the first assistant
	assistantData, err := store.GetAssistant(assistantID)
	assert.Nil(t, err)
	assert.NotNil(t, assistantData)
	assert.Equal(t, "Test Assistant", assistantData["name"])
	assert.Equal(t, "assistant", assistantData["type"])
	assert.Equal(t, "https://example.com/avatar.png", assistantData["avatar"])
	assert.Equal(t, "openai", assistantData["connector"])
	assert.Equal(t, "Test Description", assistantData["description"])
	assert.Equal(t, "/assistants/test", assistantData["path"])
	assert.Equal(t, int64(100), assistantData["sort"])
	assert.Equal(t, int64(1), assistantData["built_in"])
	assert.Equal(t, []interface{}{"tag1", "tag2", "tag3"}, assistantData["tags"])
	assert.Equal(t, map[string]interface{}{"model": "gpt-4"}, assistantData["options"])
	assert.Equal(t, map[string]interface{}{
		"title":       "Test Title",
		"description": "Test Description",
		"prompts":     []interface{}{"prompt1", "prompt2"},
	}, assistantData["placeholder"])
	assert.Equal(t, int64(1), assistantData["mentionable"])
	assert.Equal(t, int64(1), assistantData["automated"])

	// Test case 2: JSON fields as native types
	assistant2 := map[string]interface{}{
		"name":        "Test Assistant 2",
		"type":        "assistant",
		"avatar":      "https://example.com/avatar2.png",
		"connector":   "openai",
		"description": "Test Description 2",
		"path":        "/assistants/test2",
		"sort":        200,
		"built_in":    false,
		"tags":        []string{"tag1", "tag2", "tag3"},
		"options":     map[string]interface{}{"model": "gpt-4"},
		"prompts":     []string{"prompt1", "prompt2"},
		"workflow":    []string{"flow1", "flow2"},
		"knowledge":   []string{"file1", "file2"},
		"tools":       []map[string]interface{}{{"name": "tool1"}, {"name": "tool2"}},
		"permissions": map[string]interface{}{"read": true, "write": true},
		"placeholder": map[string]interface{}{
			"title":       "Test Title 2",
			"description": "Test Description 2",
			"prompts":     []string{"prompt3", "prompt4"},
		},
		"mentionable": true,
		"automated":   true,
	}

	// Test SaveAssistant (Create) with native types
	v, err = store.SaveAssistant(assistant2)
	assert.Nil(t, err)
	assistant2ID := v.(string)
	assert.NotEmpty(t, assistant2ID)

	// Test case 3: Test with nil JSON fields
	assistant3 := map[string]interface{}{
		"name":        "Test Assistant 3",
		"type":        "assistant",
		"connector":   "openai",
		"description": "Test Description 3",
		"path":        nil,
		"sort":        9999,
		"built_in":    false,
		"tags":        nil,
		"options":     nil,
		"prompts":     nil,
		"workflow":    nil,
		"knowledge":   nil,
		"tools":       nil,
		"permissions": nil,
		"placeholder": nil,
		"mentionable": true,
		"automated":   true,
	}

	// Test SaveAssistant (Create) with nil fields
	v, err = store.SaveAssistant(assistant3)
	assert.Nil(t, err)
	assistant3ID := v.(string)
	assert.NotEmpty(t, assistant3ID)

	// Test GetAssistant for the third assistant
	assistant3Data, err := store.GetAssistant(assistant3ID)
	assert.Nil(t, err)
	assert.NotNil(t, assistant3Data)
	assert.Equal(t, "Test Assistant 3", assistant3Data["name"])
	assert.Nil(t, assistant3Data["tags"])
	assert.Nil(t, assistant3Data["options"])
	assert.Nil(t, assistant3Data["prompts"])
	assert.Nil(t, assistant3Data["workflow"])
	assert.Nil(t, assistant3Data["knowledge"])
	assert.Nil(t, assistant3Data["tools"])
	assert.Nil(t, assistant3Data["permissions"])
	assert.Nil(t, assistant3Data["placeholder"])
	assert.Equal(t, int64(1), assistant3Data["mentionable"])
	assert.Equal(t, int64(1), assistant3Data["automated"])

	// Test GetAssistant with non-existent ID
	nonExistentData, err := store.GetAssistant("non-existent-id")
	assert.Error(t, err)
	assert.Nil(t, nonExistentData)
	assert.Contains(t, err.Error(), "is empty")

	// Test GetAssistants to verify JSON fields are properly stored
	resp, err := store.GetAssistants(AssistantFilter{})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(resp.Data))

	// Clean up all test data
	_, err = store.DeleteAssistants(AssistantFilter{})
	assert.Nil(t, err)

	// Verify cleanup
	resp, err = store.GetAssistants(AssistantFilter{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))
}

func TestXunAssistantPagination(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_assistant")

	// Drop assistant table before test
	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_assistant")
	if err != nil {
		t.Fatal(err)
	}

	// Add a small delay to ensure table is created
	time.Sleep(100 * time.Millisecond)

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create test data for filtering tests
	testAssistants := []map[string]interface{}{}
	for i := 0; i < 25; i++ {
		assistant := map[string]interface{}{
			"name":        fmt.Sprintf("Filter Test Assistant %d", i),
			"type":        "assistant",
			"connector":   fmt.Sprintf("connector%d", i%3),
			"description": fmt.Sprintf("Filter Test Description %d", i),
			"tags":        []string{fmt.Sprintf("tag%d", i%5)},
			"built_in":    i%2 == 0,
			"mentionable": i%2 == 0,
			"automated":   i%3 == 0,
			"sort":        9999 - i,
		}
		id, err := store.SaveAssistant(assistant)
		assert.Nil(t, err)
		assistant["assistant_id"] = id
		testAssistants = append(testAssistants, assistant)
	}

	// Get first assistant ID for later use
	firstAssistantID := testAssistants[0]["assistant_id"].(string)

	// Test filtering with assistantIDs
	assistantIDs := []string{firstAssistantID}
	if len(testAssistants) > 1 {
		assistantIDs = append(assistantIDs, testAssistants[1]["assistant_id"].(string))
	}

	// Test multiple assistant_ids
	resp, err := store.GetAssistants(AssistantFilter{
		AssistantIDs: assistantIDs,
		Page:         1,
		PageSize:     10,
	})
	assert.Nil(t, err)
	assert.Equal(t, len(assistantIDs), len(resp.Data))
	for _, assistant := range resp.Data {
		found := false
		for _, id := range assistantIDs {
			if assistant["assistant_id"] == id {
				found = true
				break
			}
		}
		assert.True(t, found, "Assistant ID should be in the requested list")
	}

	// Test assistantIDs with other filters
	resp, err = store.GetAssistants(AssistantFilter{
		AssistantIDs: assistantIDs,
		Select:       []string{"name", "assistant_id", "description"},
		Page:         1,
		PageSize:     10,
	})
	assert.Nil(t, err)
	assert.Equal(t, len(assistantIDs), len(resp.Data))
	// Verify only selected fields are returned
	for _, item := range resp.Data {
		assert.Contains(t, item, "name")
		assert.Contains(t, item, "assistant_id")
		assert.Contains(t, item, "description")
		assert.NotContains(t, item, "tags")
		assert.NotContains(t, item, "options")
	}

	// Test filtering with select fields
	resp, err = store.GetAssistants(AssistantFilter{
		Select:   []string{"name", "description", "tags"},
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 10, len(resp.Data))

	// Test filtering with select fields and other filters combined
	resp, err = store.GetAssistants(AssistantFilter{
		Tags:     []string{"tag0"},
		Keywords: "Filter Test",
		Select:   []string{"name", "tags"},
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test combined filters
	mentionableTrue := true
	automatedTrue := true
	resp, err = store.GetAssistants(AssistantFilter{
		Tags:        []string{"tag0"},
		Keywords:    "Filter Test",
		Connector:   "connector0",
		Mentionable: &mentionableTrue,
		Automated:   &automatedTrue,
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)

	// Now test the delete operations
	// Test delete by connector
	var count int64
	count, err = store.DeleteAssistants(AssistantFilter{
		Connector: "connector0",
	})
	assert.Nil(t, err)
	assert.Greater(t, count, int64(0))

	// Verify deletion
	resp, err = store.GetAssistants(AssistantFilter{
		Connector: "connector0",
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))

	// Test delete by built_in status
	builtInTrue := true
	count, err = store.DeleteAssistants(AssistantFilter{
		BuiltIn: &builtInTrue,
	})
	assert.Nil(t, err)
	assert.Greater(t, count, int64(0))

	// Verify deletion
	resp, err = store.GetAssistants(AssistantFilter{
		BuiltIn: &builtInTrue,
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))

	// Test delete by tags
	count, err = store.DeleteAssistants(AssistantFilter{
		Tags: []string{"tag1"},
	})
	assert.Nil(t, err)
	assert.Greater(t, count, int64(0))

	// Verify deletion
	resp, err = store.GetAssistants(AssistantFilter{
		Tags: []string{"tag1"},
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))

	// Test delete by keywords
	count, err = store.DeleteAssistants(AssistantFilter{
		Keywords: "Filter Test",
	})
	assert.Nil(t, err)
	assert.Greater(t, count, int64(0))

	// Verify all assistants are deleted
	resp, err = store.GetAssistants(AssistantFilter{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))

	// Test delete by assistantIDs
	// First create some test assistants
	testIDs := []string{}
	for i := 0; i < 3; i++ {
		assistant := map[string]interface{}{
			"name":        fmt.Sprintf("AssistantIDs Test Assistant %d", i),
			"type":        "assistant",
			"connector":   "test",
			"description": fmt.Sprintf("AssistantIDs Test Description %d", i),
			"tags":        []string{"test-tag"},
			"built_in":    false,
			"mentionable": true,
			"automated":   true,
		}
		id, err := store.SaveAssistant(assistant)
		assert.Nil(t, err)
		testIDs = append(testIDs, id.(string))
	}

	// Delete by assistantIDs
	count, err = store.DeleteAssistants(AssistantFilter{
		AssistantIDs: testIDs,
	})
	assert.Nil(t, err)
	assert.Equal(t, int64(len(testIDs)), count)

	// Verify deletion
	resp, err = store.GetAssistants(AssistantFilter{
		AssistantIDs: testIDs,
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))

	// Verify all assistants are deleted
	resp, err = store.GetAssistants(AssistantFilter{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))
}

func TestGetAssistantTags(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_assistant")

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create test assistants with tags
	assistants := []map[string]interface{}{
		{
			"assistant_id": "test-assistant-1",
			"type":         "assistant",
			"connector":    "test",
			"tags":         []string{"tag1", "tag2"},
			"name":         "Test Assistant 1",
		},
		{
			"assistant_id": "test-assistant-2",
			"type":         "assistant",
			"connector":    "test",
			"tags":         []string{"tag2", "tag3"},
			"name":         "Test Assistant 2",
		},
		{
			"assistant_id": "test-assistant-3",
			"type":         "assistant",
			"connector":    "test",
			"tags":         []string{"tag1", "tag3", "tag4"},
			"name":         "Test Assistant 3",
		},
	}

	// Save test assistants
	for _, assistant := range assistants {
		_, err := store.SaveAssistant(assistant)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Get tags
	tags, err := store.GetAssistantTags()
	if err != nil {
		t.Fatal(err)
	}

	// Verify results
	expectedTags := map[string]bool{
		"tag1": true,
		"tag2": true,
		"tag3": true,
		"tag4": true,
	}

	if len(tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(tags))
	}

	for _, tag := range tags {
		value := tag.Value
		if !expectedTags[value] {
			t.Errorf("Unexpected tag found: %s", tag)
		}
	}

	// Cleanup
	for _, assistant := range assistants {
		err := store.DeleteAssistant(assistant["assistant_id"].(string))
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestXunSaveAndGetHistoryWithSilent(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")

	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	if err != nil {
		t.Fatal(err)
	}

	err = capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")
	if err != nil {
		t.Fatal(err)
	}

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
		TTL:       3600,
	})

	// save the history with silent messages
	sid := "123456"
	cid := "silent_test"

	// First save regular messages
	messages := []map[string]interface{}{
		{"role": "user", "name": "user1", "content": "hello"},
		{"role": "assistant", "name": "assistant1", "content": "Hi! How can I help you?"},
	}
	context := map[string]interface{}{
		"assistant_id": "test-assistant-1",
	}
	err = store.SaveHistory(sid, messages, cid, context)
	assert.Nil(t, err)

	// Then save silent messages
	silentMessages := []map[string]interface{}{
		{"role": "user", "name": "user1", "content": "silent message"},
		{"role": "assistant", "name": "assistant1", "content": "This is a silent response"},
	}
	silentContext := map[string]interface{}{
		"assistant_id": "test-assistant-1",
		"silent":       true,
	}
	err = store.SaveHistory(sid, silentMessages, cid, silentContext)
	assert.Nil(t, err)

	// Get history without filter (should only return non-silent messages)
	data, err := store.GetHistory(sid, cid)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(data))
	for _, msg := range data {
		// Check if silent is false, handling different types
		isSilent := false
		switch v := msg["silent"].(type) {
		case bool:
			isSilent = v
		case int:
			isSilent = v != 0
		case int64:
			isSilent = v != 0
		case float64:
			isSilent = v != 0
		}
		assert.False(t, isSilent, "message should not be silent")
	}

	// Get history with silent=true filter (should return all messages)
	silentTrue := true
	filter := ChatFilter{
		Silent: &silentTrue,
	}
	allData, err := store.GetHistoryWithFilter(sid, cid, filter)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 4, len(allData))

	// Count silent messages
	silentCount := 0
	for _, msg := range allData {
		// Check if silent is true, handling different types
		isSilent := false
		switch v := msg["silent"].(type) {
		case bool:
			isSilent = v
		case int:
			isSilent = v != 0
		case int64:
			isSilent = v != 0
		case float64:
			isSilent = v != 0
		}
		if isSilent {
			silentCount++
		}
	}
	assert.Equal(t, 2, silentCount)

	// Get chat with filter (should include silent messages)
	chat, err := store.GetChatWithFilter(sid, cid, filter)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(chat.History))

	// Get chat without filter (should exclude silent messages)
	chatNoSilent, err := store.GetChat(sid, cid)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(chatNoSilent.History))
}

func TestXunGetChatsWithSilent(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_assistant")

	// Drop tables before test
	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	if err != nil {
		t.Fatal(err)
	}
	err = capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")
	if err != nil {
		t.Fatal(err)
	}
	err = capsule.Schema().DropTableIfExists("__unit_test_conversation_assistant")
	if err != nil {
		t.Fatal(err)
	}

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create test assistant
	assistant := map[string]interface{}{
		"assistant_id": "test-assistant-1",
		"name":         "Test Assistant 1",
		"avatar":       "avatar1.png",
		"type":         "assistant",
		"connector":    "test",
	}
	_, err = store.SaveAssistant(assistant)
	assert.Nil(t, err)

	// Save some test chats
	sid := "test_user"
	messages := []map[string]interface{}{
		{"role": "user", "content": "test message"},
	}

	// Create regular chats
	for i := 0; i < 3; i++ {
		chatID := fmt.Sprintf("regular_chat_%d", i)
		title := fmt.Sprintf("Regular Chat %d", i)
		context := map[string]interface{}{
			"assistant_id": "test-assistant-1",
			"silent":       false,
		}

		// Save history to create the chat
		err = store.SaveHistory(sid, messages, chatID, context)
		assert.Nil(t, err)

		// Update the chat title
		err = store.UpdateChatTitle(sid, chatID, title)
		assert.Nil(t, err)
	}

	// Create silent chats
	for i := 0; i < 2; i++ {
		chatID := fmt.Sprintf("silent_chat_%d", i)
		title := fmt.Sprintf("Silent Chat %d", i)
		context := map[string]interface{}{
			"assistant_id": "test-assistant-1",
			"silent":       true,
		}

		// Save history to create the chat
		err = store.SaveHistory(sid, messages, chatID, context)
		assert.Nil(t, err)

		// Update the chat title
		err = store.UpdateChatTitle(sid, chatID, title)
		assert.Nil(t, err)
	}

	// Test GetChats with default filter (should exclude silent chats)
	defaultFilter := ChatFilter{
		PageSize: 10,
		Order:    "desc",
	}
	defaultGroups, err := store.GetChats(sid, defaultFilter)
	assert.Nil(t, err)
	assert.NotNil(t, defaultGroups)

	// Count total chats in all groups
	totalDefaultChats := 0
	for _, group := range defaultGroups.Groups {
		totalDefaultChats += len(group.Chats)
	}
	assert.Equal(t, 3, totalDefaultChats, "Default filter should only return non-silent chats")

	// Test GetChats with silent=true filter (should include all chats)
	silentTrue := true
	silentFilter := ChatFilter{
		PageSize: 10,
		Order:    "desc",
		Silent:   &silentTrue,
	}
	silentGroups, err := store.GetChats(sid, silentFilter)
	assert.Nil(t, err)
	assert.NotNil(t, silentGroups)

	// Count total chats in all groups
	totalSilentChats := 0
	for _, group := range silentGroups.Groups {
		totalSilentChats += len(group.Chats)
	}
	assert.Equal(t, 5, totalSilentChats, "Silent filter should return all chats")

	// Test GetChats with silent=false filter (should only include non-silent chats)
	silentFalse := false
	nonSilentFilter := ChatFilter{
		PageSize: 10,
		Order:    "desc",
		Silent:   &silentFalse,
	}
	nonSilentGroups, err := store.GetChats(sid, nonSilentFilter)
	assert.Nil(t, err)
	assert.NotNil(t, nonSilentGroups)

	// Count total chats in all groups
	totalNonSilentChats := 0
	for _, group := range nonSilentGroups.Groups {
		totalNonSilentChats += len(group.Chats)
	}
	assert.Equal(t, 3, totalNonSilentChats, "Non-silent filter should only return non-silent chats")
}

func TestXunAttachmentCRUD(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_attachment")

	// Drop attachment table before test
	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_attachment")
	if err != nil {
		t.Fatal(err)
	}

	// Add a small delay to ensure table is created
	time.Sleep(100 * time.Millisecond)

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Clean up any existing data
	_, err = store.DeleteAttachments(AttachmentFilter{})
	assert.Nil(t, err)

	// Test SaveAttachment (Create)
	attachment := map[string]interface{}{
		"file_id":      "test-file-123",
		"uid":          "user-123",
		"manager":      "local",
		"content_type": "image/jpeg",
		"name":         "test-image.jpg",
		"guest":        false,
		"public":       true,
		"gzip":         false,
		"bytes":        102400,
		"scope":        []string{"user", "admin"},
		"status":       "uploaded",
		"progress":     "100%",
		"error":        nil,
	}

	v, err := store.SaveAttachment(attachment)
	assert.Nil(t, err)
	fileID := v.(string)
	assert.Equal(t, "test-file-123", fileID)

	// Test GetAttachment
	attachmentData, err := store.GetAttachment(fileID)
	assert.Nil(t, err)
	assert.NotNil(t, attachmentData)
	assert.Equal(t, "test-file-123", attachmentData["file_id"])
	assert.Equal(t, "user-123", attachmentData["uid"])
	assert.Equal(t, "local", attachmentData["manager"])
	assert.Equal(t, "image/jpeg", attachmentData["content_type"])
	assert.Equal(t, "test-image.jpg", attachmentData["name"])
	assert.Equal(t, int64(1), attachmentData["public"])
	assert.Equal(t, []interface{}{"user", "admin"}, attachmentData["scope"])
	assert.Equal(t, "uploaded", attachmentData["status"])
	assert.Equal(t, "100%", attachmentData["progress"])
	assert.Nil(t, attachmentData["error"])

	// Test SaveAttachment (Update)
	attachment["name"] = "updated-image.jpg"
	attachment["bytes"] = 204800
	attachment["status"] = "indexing"
	attachment["progress"] = "Processing..."
	attachment["error"] = "Connection timeout"
	v, err = store.SaveAttachment(attachment)
	assert.Nil(t, err)
	assert.Equal(t, "test-file-123", v.(string))

	// Verify update
	attachmentData, err = store.GetAttachment(fileID)
	assert.Nil(t, err)
	assert.Equal(t, "updated-image.jpg", attachmentData["name"])
	assert.Equal(t, int64(204800), attachmentData["bytes"])
	assert.Equal(t, "indexing", attachmentData["status"])
	assert.Equal(t, "Processing...", attachmentData["progress"])
	assert.Equal(t, "Connection timeout", attachmentData["error"])

	// Test GetAttachments with filters
	resp, err := store.GetAttachments(AttachmentFilter{
		UID:      "user-123",
		Manager:  "local",
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(resp.Data))
	assert.Equal(t, "test-file-123", resp.Data[0]["file_id"])

	// Test with non-existent file
	nonExistentData, err := store.GetAttachment("non-existent-file")
	assert.Error(t, err)
	assert.Nil(t, nonExistentData)
	assert.Contains(t, err.Error(), "is empty")

	// Test DeleteAttachment
	err = store.DeleteAttachment(fileID)
	assert.Nil(t, err)

	// Verify deletion
	_, err = store.GetAttachment(fileID)
	assert.Error(t, err)
}

func TestXunKnowledgeCRUD(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_knowledge")

	// Drop knowledge table before test
	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_knowledge")
	if err != nil {
		t.Fatal(err)
	}

	// Add a small delay to ensure table is created
	time.Sleep(100 * time.Millisecond)

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Clean up any existing data
	_, err = store.DeleteKnowledges(KnowledgeFilter{})
	assert.Nil(t, err)

	// Test SaveKnowledge (Create)
	knowledge := map[string]interface{}{
		"collection_id": "test-collection-123",
		"name":          "Test Knowledge Collection",
		"description":   "A test knowledge collection for unit tests",
		"uid":           "user-123",
		"public":        true,
		"readonly":      false,
		"system":        false,
		"sort":          100,
		"cover":         "cover-image.jpg",
		"scope":         []string{"user", "admin"},
		"option":        map[string]interface{}{"embedding": "openai", "chunk_size": 1000},
	}

	v, err := store.SaveKnowledge(knowledge)
	assert.Nil(t, err)
	collectionID := v.(string)
	assert.Equal(t, "test-collection-123", collectionID)

	// Test GetKnowledge
	knowledgeData, err := store.GetKnowledge(collectionID)
	assert.Nil(t, err)
	assert.NotNil(t, knowledgeData)
	assert.Equal(t, "test-collection-123", knowledgeData["collection_id"])
	assert.Equal(t, "Test Knowledge Collection", knowledgeData["name"])
	assert.Equal(t, "A test knowledge collection for unit tests", knowledgeData["description"])
	assert.Equal(t, "user-123", knowledgeData["uid"])
	assert.Equal(t, int64(1), knowledgeData["public"])
	assert.Equal(t, int64(100), knowledgeData["sort"])
	assert.Equal(t, []interface{}{"user", "admin"}, knowledgeData["scope"])
	assert.Equal(t, map[string]interface{}{"embedding": "openai", "chunk_size": float64(1000)}, knowledgeData["option"])

	// Test SaveKnowledge (Update)
	knowledge["name"] = "Updated Knowledge Collection"
	knowledge["description"] = "Updated description"
	knowledge["sort"] = 200
	v, err = store.SaveKnowledge(knowledge)
	assert.Nil(t, err)
	assert.Equal(t, "test-collection-123", v.(string))

	// Verify update
	knowledgeData, err = store.GetKnowledge(collectionID)
	assert.Nil(t, err)
	assert.Equal(t, "Updated Knowledge Collection", knowledgeData["name"])
	assert.Equal(t, "Updated description", knowledgeData["description"])
	assert.Equal(t, int64(200), knowledgeData["sort"])

	// Test knowledge without sort field (should get default value 9999)
	knowledgeWithoutSort := map[string]interface{}{
		"collection_id": "test-collection-456",
		"name":          "Test Knowledge Without Sort",
		"description":   "Test knowledge without explicit sort value",
		"uid":           "user-123",
	}
	v2, err := store.SaveKnowledge(knowledgeWithoutSort)
	assert.Nil(t, err)

	// Verify default sort value
	knowledgeData2, err := store.GetKnowledge(v2.(string))
	assert.Nil(t, err)
	assert.Equal(t, int64(9999), knowledgeData2["sort"])

	// Test GetKnowledges with filters
	resp, err := store.GetKnowledges(KnowledgeFilter{
		UID:      "user-123",
		Keywords: "Updated",
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(resp.Data))
	assert.Equal(t, "test-collection-123", resp.Data[0]["collection_id"])

	// Test with non-existent collection
	nonExistentData, err := store.GetKnowledge("non-existent-collection")
	assert.Error(t, err)
	assert.Nil(t, nonExistentData)
	assert.Contains(t, err.Error(), "is empty")

	// Test DeleteKnowledge
	err = store.DeleteKnowledge(collectionID)
	assert.Nil(t, err)
	err = store.DeleteKnowledge(v2.(string))
	assert.Nil(t, err)

	// Verify deletion
	_, err = store.GetKnowledge(collectionID)
	assert.Error(t, err)
}

func TestXunKnowledgeFiltering(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_knowledge")

	// Drop knowledge table before test
	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_knowledge")
	if err != nil {
		t.Fatal(err)
	}

	// Add a small delay to ensure table is created
	time.Sleep(100 * time.Millisecond)

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create test data for filtering tests
	testKnowledges := []map[string]interface{}{}
	for i := 0; i < 15; i++ {
		knowledge := map[string]interface{}{
			"collection_id": fmt.Sprintf("test-collection-%d", i),
			"name":          fmt.Sprintf("Collection %d", i),
			"description":   fmt.Sprintf("Description for collection %d", i),
			"uid":           fmt.Sprintf("user-%d", i%3),
			"public":        i%2 == 0,
			"readonly":      i%3 == 0,
			"system":        i%4 == 0,
			"sort":          100 + i*10, // Different sort values for testing ordering
			"cover":         fmt.Sprintf("cover%d.jpg", i),
		}
		id, err := store.SaveKnowledge(knowledge)
		assert.Nil(t, err)
		knowledge["collection_id"] = id
		testKnowledges = append(testKnowledges, knowledge)
	}

	// Test sorting functionality - should return results ordered by sort ASC then created_at DESC
	respAll, err := store.GetKnowledges(KnowledgeFilter{
		Page:     1,
		PageSize: 15,
	})
	assert.Nil(t, err)
	assert.Equal(t, 15, len(respAll.Data))

	// Verify sort order - first item should have the smallest sort value
	firstSort := respAll.Data[0]["sort"].(int64)
	lastSort := respAll.Data[len(respAll.Data)-1]["sort"].(int64)
	assert.LessOrEqual(t, firstSort, lastSort, "Results should be ordered by sort ASC")

	// More specific sort order verification
	for i := 1; i < len(respAll.Data); i++ {
		prevSort := respAll.Data[i-1]["sort"].(int64)
		currSort := respAll.Data[i]["sort"].(int64)
		assert.LessOrEqual(t, prevSort, currSort, "Sort order should be ascending")
	}

	// Test filtering by UID
	resp, err := store.GetKnowledges(KnowledgeFilter{
		UID:      "user-0",
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering by public status
	publicTrue := true
	resp, err = store.GetKnowledges(KnowledgeFilter{
		Public:   &publicTrue,
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering by readonly status
	readonlyTrue := true
	resp, err = store.GetKnowledges(KnowledgeFilter{
		Readonly: &readonlyTrue,
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering by system status
	systemTrue := true
	resp, err = store.GetKnowledges(KnowledgeFilter{
		System:   &systemTrue,
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering by keywords
	resp, err = store.GetKnowledges(KnowledgeFilter{
		Keywords: "Collection 1",
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test DeleteKnowledges with filter
	count, err := store.DeleteKnowledges(KnowledgeFilter{
		UID: "user-0",
	})
	assert.Nil(t, err)
	assert.Greater(t, count, int64(0))

	// Verify deletion
	resp, err = store.GetKnowledges(KnowledgeFilter{
		UID: "user-0",
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))

	// Clean up all test data
	_, err = store.DeleteKnowledges(KnowledgeFilter{})
	assert.Nil(t, err)
}

func TestXunAttachmentFiltering(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_attachment")

	// Drop attachment table before test
	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_attachment")
	if err != nil {
		t.Fatal(err)
	}

	// Add a small delay to ensure table is created
	time.Sleep(100 * time.Millisecond)

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create test data for filtering tests
	testAttachments := []map[string]interface{}{}
	for i := 0; i < 15; i++ {
		attachment := map[string]interface{}{
			"file_id":       fmt.Sprintf("test-file-%d", i),
			"uid":           fmt.Sprintf("user-%d", i%3),
			"manager":       fmt.Sprintf("manager%d", i%2),
			"content_type":  fmt.Sprintf("type/%d", i%4),
			"name":          fmt.Sprintf("file%d.txt", i),
			"guest":         i%2 == 0,
			"public":        i%3 == 0,
			"gzip":          i%4 == 0,
			"bytes":         1024 * (i + 1),
			"collection_id": fmt.Sprintf("collection-%d", i%5),
		}
		id, err := store.SaveAttachment(attachment)
		assert.Nil(t, err)
		attachment["file_id"] = id
		testAttachments = append(testAttachments, attachment)
	}

	// Test filtering by UID
	resp, err := store.GetAttachments(AttachmentFilter{
		UID:      "user-0",
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering by manager
	resp, err = store.GetAttachments(AttachmentFilter{
		Manager:  "manager0",
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering by content_type
	resp, err = store.GetAttachments(AttachmentFilter{
		ContentType: "type/0",
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering by guest status
	guestTrue := true
	resp, err = store.GetAttachments(AttachmentFilter{
		Guest:    &guestTrue,
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering by public status
	publicTrue := true
	resp, err = store.GetAttachments(AttachmentFilter{
		Public:   &publicTrue,
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering by keywords
	resp, err = store.GetAttachments(AttachmentFilter{
		Keywords: "file1",
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test DeleteAttachments with filter
	count, err := store.DeleteAttachments(AttachmentFilter{
		Manager: "manager0",
	})
	assert.Nil(t, err)
	assert.Greater(t, count, int64(0))

	// Verify deletion
	resp, err = store.GetAttachments(AttachmentFilter{
		Manager: "manager0",
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))

	// Clean up all test data
	_, err = store.DeleteAttachments(AttachmentFilter{})
	assert.Nil(t, err)
}

func TestXunAttachmentStatusFields(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_attachment")

	// Drop attachment table before test
	err := capsule.Schema().DropTableIfExists("__unit_test_conversation_attachment")
	if err != nil {
		t.Fatal(err)
	}

	// Add a small delay to ensure table is created
	time.Sleep(100 * time.Millisecond)

	store, err := NewXun(Setting{
		Connector: "default",
		Prefix:    "__unit_test_conversation_",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Clean up any existing data
	_, err = store.DeleteAttachments(AttachmentFilter{})
	assert.Nil(t, err)

	// Test all possible enum status values
	statusValues := []string{"uploading", "uploaded", "indexing", "indexed", "upload_failed", "index_failed"}

	for i, status := range statusValues {
		// Create attachment with specific status
		attachment := map[string]interface{}{
			"file_id":      fmt.Sprintf("test-file-%s-%d", status, i),
			"uid":          "user-123",
			"manager":      "local",
			"content_type": "image/jpeg",
			"name":         fmt.Sprintf("test-%s.jpg", status),
			"guest":        false,
			"public":       true,
			"gzip":         false,
			"bytes":        102400,
			"status":       status,
			"progress":     fmt.Sprintf("%s in progress", status),
			"error":        nil,
		}

		// Set error message for failed statuses
		if status == "upload_failed" || status == "index_failed" {
			attachment["error"] = fmt.Sprintf("%s error occurred", status)
		}

		v, err := store.SaveAttachment(attachment)
		assert.Nil(t, err)
		fileID := v.(string)

		// Verify the attachment was saved with correct status
		attachmentData, err := store.GetAttachment(fileID)
		assert.Nil(t, err)
		assert.Equal(t, status, attachmentData["status"])
		assert.Equal(t, fmt.Sprintf("%s in progress", status), attachmentData["progress"])

		if status == "upload_failed" || status == "index_failed" {
			assert.Equal(t, fmt.Sprintf("%s error occurred", status), attachmentData["error"])
		} else {
			assert.Nil(t, attachmentData["error"])
		}
	}

	// Test default status value (should be "uploading")
	attachmentWithoutStatus := map[string]interface{}{
		"file_id":      "test-file-default",
		"uid":          "user-123",
		"manager":      "local",
		"content_type": "image/jpeg",
		"name":         "test-default.jpg",
		"guest":        false,
		"public":       true,
		"gzip":         false,
		"bytes":        102400,
		// status not specified - should use default
	}

	v, err := store.SaveAttachment(attachmentWithoutStatus)
	assert.Nil(t, err)
	fileID := v.(string)

	// Verify default status
	attachmentData, err := store.GetAttachment(fileID)
	assert.Nil(t, err)
	assert.Equal(t, "uploading", attachmentData["status"]) // Should be default value
	assert.Nil(t, attachmentData["progress"])              // Should be null
	assert.Nil(t, attachmentData["error"])                 // Should be null

	// Test updating status workflow: uploading -> uploaded -> indexing -> indexed
	workflowAttachment := map[string]interface{}{
		"file_id":      "test-file-workflow",
		"uid":          "user-123",
		"manager":      "local",
		"content_type": "text/plain",
		"name":         "workflow-test.txt",
		"status":       "uploading",
		"progress":     "Starting upload...",
	}

	v, err = store.SaveAttachment(workflowAttachment)
	assert.Nil(t, err)
	workflowFileID := v.(string)

	// Update to uploaded
	workflowAttachment["status"] = "uploaded"
	workflowAttachment["progress"] = "Upload completed, starting indexing..."
	_, err = store.SaveAttachment(workflowAttachment)
	assert.Nil(t, err)

	attachmentData, err = store.GetAttachment(workflowFileID)
	assert.Nil(t, err)
	assert.Equal(t, "uploaded", attachmentData["status"])
	assert.Equal(t, "Upload completed, starting indexing...", attachmentData["progress"])

	// Update to indexing
	workflowAttachment["status"] = "indexing"
	workflowAttachment["progress"] = "Indexing in progress..."
	_, err = store.SaveAttachment(workflowAttachment)
	assert.Nil(t, err)

	attachmentData, err = store.GetAttachment(workflowFileID)
	assert.Nil(t, err)
	assert.Equal(t, "indexing", attachmentData["status"])
	assert.Equal(t, "Indexing in progress...", attachmentData["progress"])

	// Update to indexed (final state)
	workflowAttachment["status"] = "indexed"
	workflowAttachment["progress"] = "Indexing completed"
	_, err = store.SaveAttachment(workflowAttachment)
	assert.Nil(t, err)

	attachmentData, err = store.GetAttachment(workflowFileID)
	assert.Nil(t, err)
	assert.Equal(t, "indexed", attachmentData["status"])
	assert.Equal(t, "Indexing completed", attachmentData["progress"])

	// Clean up test data
	_, err = store.DeleteAttachments(AttachmentFilter{})
	assert.Nil(t, err)
}
