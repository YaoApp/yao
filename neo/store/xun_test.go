package store

import (
	"fmt"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
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

	sch.DropTableIfExists("__unit_test_conversation_history")
	sch.DropTableIfExists("__unit_test_conversation_chat")
	sch.DropTableIfExists("__unit_test_conversation_assistant")

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
	messages := []map[string]interface{}{
		{"role": "user", "name": "user1", "content": "hello"},
		{"role": "assistant", "name": "assistant1", "content": "Hi! How can I help you?"},
	}
	err = store.SaveHistory(sid, messages, cid, nil)
	assert.Nil(t, err)

	// get the history for specific cid
	data, err := store.GetHistory(sid, cid)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(data))

	// save another message with different cid
	anotherCID := "345678"
	moreMessages := []map[string]interface{}{
		{"role": "user", "name": "user1", "content": "another message"},
	}
	err = store.SaveHistory(sid, moreMessages, anotherCID, nil)
	assert.Nil(t, err)

	// get history for the first cid - should still be 2 messages
	data, err = store.GetHistory(sid, cid)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(data))

	// get history for the second cid - should be 1 message
	data, err = store.GetHistory(sid, anotherCID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(data))

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

	// Drop both tables before test
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
	})
	if err != nil {
		t.Fatal(err)
	}

	// Save some test chats
	sid := "test_user"
	messages := []map[string]interface{}{
		{"role": "user", "content": "test message"},
	}

	// Create chats with different dates
	for i := 0; i < 5; i++ {
		chatID := fmt.Sprintf("chat_%d", i)
		title := fmt.Sprintf("Test Chat %d", i)

		// Save history first to create the chat
		err = store.SaveHistory(sid, messages, chatID, nil)
		assert.Nil(t, err)

		// Update the chat title
		err = store.UpdateChatTitle(sid, chatID, title)
		assert.Nil(t, err)
	}

	// Test getting chats with default filter
	filter := ChatFilter{
		PageSize: 10,
		Order:    "desc",
	}
	groups, err := store.GetChats(sid, filter)
	if err != nil {
		t.Fatal(err)
	}

	assert.Greater(t, len(groups.Groups), 0)

	// Test with keywords
	filter.Keywords = "test"
	groups, err = store.GetChats(sid, filter)
	if err != nil {
		t.Fatal(err)
	}

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
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
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

	// Test creating a new assistant with different JSON field formats
	// Test case 1: JSON fields as strings
	tagsJSON := `["tag1", "tag2", "tag3"]`
	optionsJSON := `{"model": "gpt-4"}`
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
		"flows":       []string{"flow1", "flow2"},
		"files":       []string{"file1", "file2"},
		"functions":   []map[string]interface{}{{"name": "func1"}, {"name": "func2"}},
		"permissions": map[string]interface{}{"read": true, "write": true},
		"mentionable": true,
		"automated":   true,
	}

	// Test SaveAssistant (Create) with native types
	v, err = store.SaveAssistant(assistant2)
	assert.Nil(t, err)
	assistant2ID := v.(string)
	assert.NotEmpty(t, assistant2ID)

	// Test GetAssistant for the second assistant
	assistant2Data, err := store.GetAssistant(assistant2ID)
	assert.Nil(t, err)
	assert.NotNil(t, assistant2Data)
	assert.Equal(t, "Test Assistant 2", assistant2Data["name"])
	assert.Equal(t, []interface{}{"tag1", "tag2", "tag3"}, assistant2Data["tags"])
	assert.Equal(t, map[string]interface{}{"model": "gpt-4"}, assistant2Data["options"])
	assert.Equal(t, []interface{}{"prompt1", "prompt2"}, assistant2Data["prompts"])
	assert.Equal(t, []interface{}{"flow1", "flow2"}, assistant2Data["flows"])
	assert.Equal(t, []interface{}{"file1", "file2"}, assistant2Data["files"])
	assert.Equal(t, []interface{}{
		map[string]interface{}{"name": "func1"},
		map[string]interface{}{"name": "func2"},
	}, assistant2Data["functions"])
	assert.Equal(t, map[string]interface{}{"read": true, "write": true}, assistant2Data["permissions"])
	assert.Equal(t, int64(1), assistant2Data["mentionable"])
	assert.Equal(t, int64(1), assistant2Data["automated"])

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
		"flows":       nil,
		"files":       nil,
		"functions":   nil,
		"permissions": nil,
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
	assert.Nil(t, assistant3Data["flows"])
	assert.Nil(t, assistant3Data["files"])
	assert.Nil(t, assistant3Data["functions"])
	assert.Nil(t, assistant3Data["permissions"])
	assert.Equal(t, int64(1), assistant3Data["mentionable"])
	assert.Equal(t, int64(1), assistant3Data["automated"])

	// Test GetAssistant with non-existent ID
	nonExistentData, err := store.GetAssistant("non-existent-id")
	assert.Error(t, err)
	assert.Nil(t, nonExistentData)
	assert.Contains(t, err.Error(), "not found")

	// Test GetAssistants to verify JSON fields are properly stored
	resp, err := store.GetAssistants(AssistantFilter{})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(resp.Data))

	// Verify first assistant (string JSON)
	found := false
	for _, item := range resp.Data {
		if item["assistant_id"].(string) == assistantID {
			found = true
			// Now we expect parsed JSON values instead of JSON strings
			assert.Equal(t, []interface{}{"tag1", "tag2", "tag3"}, item["tags"])
			assert.Equal(t, map[string]interface{}{"model": "gpt-4"}, item["options"])
			break
		}
	}
	assert.True(t, found)

	// Verify second assistant (native types converted to JSON)
	found = false
	for _, item := range resp.Data {
		if item["assistant_id"].(string) == assistant2ID {
			found = true
			// Now we expect parsed JSON values directly
			assert.Equal(t, []interface{}{"tag1", "tag2", "tag3"}, item["tags"])
			assert.Equal(t, map[string]interface{}{"model": "gpt-4"}, item["options"])

			// Verify other JSON fields
			assert.Equal(t, []interface{}{"prompt1", "prompt2"}, item["prompts"])
			assert.Equal(t, []interface{}{"flow1", "flow2"}, item["flows"])
			assert.Equal(t, []interface{}{"file1", "file2"}, item["files"])
			assert.Equal(t,
				[]interface{}{
					map[string]interface{}{"name": "func1"},
					map[string]interface{}{"name": "func2"},
				},
				item["functions"])
			assert.Equal(t,
				map[string]interface{}{
					"read":  true,
					"write": true,
				},
				item["permissions"])
			break
		}
	}
	assert.True(t, found)

	// Verify third assistant (nil fields)
	found = false
	for _, item := range resp.Data {
		if item["assistant_id"].(string) == assistant3ID {
			found = true
			assert.Nil(t, item["tags"])
			assert.Nil(t, item["options"])
			assert.Nil(t, item["prompts"])
			assert.Nil(t, item["flows"])
			assert.Nil(t, item["files"])
			assert.Nil(t, item["functions"])
			assert.Nil(t, item["permissions"])
			break
		}
	}
	assert.True(t, found)

	// Test updating with mixed JSON formats
	assistant2["assistant_id"] = assistant2ID
	_, err = store.SaveAssistant(assistant2)
	assert.Nil(t, err)

	// Verify update
	resp, err = store.GetAssistants(AssistantFilter{})
	assert.Nil(t, err)
	for _, item := range resp.Data {
		if item["assistant_id"].(string) == assistant2ID {
			// Now we expect parsed JSON values
			assert.Equal(t, []interface{}{"tag1", "tag2", "tag3"}, item["tags"])
			assert.Equal(t, map[string]interface{}{"model": "gpt-4"}, item["options"])
			break
		}
	}

	// Test non-existent assistant_id
	resp, err = store.GetAssistants(AssistantFilter{
		AssistantID: "non-existent-id",
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))

	// Test filtering with select fields
	resp, err = store.GetAssistants(AssistantFilter{
		Select:   []string{"name", "description", "tags"},
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	// Verify only selected fields are returned
	for _, item := range resp.Data {
		// These fields should exist
		assert.Contains(t, item, "name")
		assert.Contains(t, item, "description")
		assert.Contains(t, item, "tags")
		// These fields should not exist
		assert.NotContains(t, item, "options")
		assert.NotContains(t, item, "prompts")
		assert.NotContains(t, item, "flows")
		assert.NotContains(t, item, "files")
		assert.NotContains(t, item, "functions")
		assert.NotContains(t, item, "permissions")
	}

	// Test filtering with select fields and other filters combined
	resp, err = store.GetAssistants(AssistantFilter{
		Tags:     []string{"tag1"},
		Keywords: "Assistant",
		Select:   []string{"name", "tags"},
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	// Verify only selected fields are returned
	for _, item := range resp.Data {
		// These fields should exist
		assert.Contains(t, item, "name")
		assert.Contains(t, item, "tags")
		// These fields should not exist
		assert.NotContains(t, item, "description")
		assert.NotContains(t, item, "options")
		assert.NotContains(t, item, "prompts")
		assert.NotContains(t, item, "flows")
		assert.NotContains(t, item, "files")
		assert.NotContains(t, item, "functions")
		assert.NotContains(t, item, "permissions")
	}

	// Test filtering with automated
	automatedTrue := true
	resp, err = store.GetAssistants(AssistantFilter{
		Automated: &automatedTrue,
		Page:      1,
		PageSize:  10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering with mentionable
	mentionableTrue := true
	resp, err = store.GetAssistants(AssistantFilter{
		Mentionable: &mentionableTrue,
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test combined filters
	resp, err = store.GetAssistants(AssistantFilter{
		Tags:        []string{"tag1"},
		Keywords:    "Assistant",
		Connector:   "openai",
		Mentionable: &mentionableTrue,
		Automated:   &automatedTrue,
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)

	// Test filtering with built_in
	builtInTrue := true
	resp, err = store.GetAssistants(AssistantFilter{
		BuiltIn:  &builtInTrue,
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	for _, assistant := range resp.Data {
		assert.Equal(t, int64(1), assistant["built_in"], "All assistants should be built-in")
	}

	builtInFalse := false
	resp, err = store.GetAssistants(AssistantFilter{
		BuiltIn:  &builtInFalse,
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	for _, assistant := range resp.Data {
		assert.Equal(t, int64(0), assistant["built_in"], "All assistants should not be built-in")
	}

	// Now test the delete operations
	// First create some test data for delete operations
	for i := 0; i < 5; i++ {
		assistant := map[string]interface{}{
			"name":        fmt.Sprintf("Delete Test Assistant %d", i),
			"type":        "assistant",
			"connector":   "openai",
			"description": fmt.Sprintf("Delete Test Description %d", i),
			"tags":        []string{"delete-tag1", "delete-tag2"},
			"built_in":    i%2 == 0,
			"mentionable": true,
			"automated":   true,
		}
		_, err = store.SaveAssistant(assistant)
		assert.Nil(t, err)
	}

	// Test delete by connector
	count, err := store.DeleteAssistants(AssistantFilter{
		Connector: "openai",
	})
	assert.Nil(t, err)
	assert.Greater(t, count, int64(0))

	// Verify deletion
	resp, err = store.GetAssistants(AssistantFilter{
		Connector: "openai",
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))

	// Create more test data for built_in test
	for i := 0; i < 5; i++ {
		assistant := map[string]interface{}{
			"name":        fmt.Sprintf("Built-in Test Assistant %d", i),
			"type":        "assistant",
			"connector":   "openai",
			"description": fmt.Sprintf("Built-in Test Description %d", i),
			"tags":        []string{"builtin-tag1", "builtin-tag2"},
			"built_in":    true,
			"mentionable": true,
			"automated":   true,
		}
		_, err = store.SaveAssistant(assistant)
		assert.Nil(t, err)
	}

	// Test delete by built_in status
	builtInTrue = true
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

	// Create more test data for tags test
	for i := 0; i < 5; i++ {
		assistant := map[string]interface{}{
			"name":        fmt.Sprintf("Tags Test Assistant %d", i),
			"type":        "assistant",
			"connector":   "openai",
			"description": fmt.Sprintf("Tags Test Description %d", i),
			"tags":        []string{"tag1", "tag2"},
			"built_in":    false,
			"mentionable": true,
			"automated":   true,
		}
		_, err = store.SaveAssistant(assistant)
		assert.Nil(t, err)
	}

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

	// Create more test data for keywords test
	for i := 0; i < 5; i++ {
		assistant := map[string]interface{}{
			"name":        fmt.Sprintf("Keywords Test Assistant %d", i),
			"type":        "assistant",
			"connector":   "openai",
			"description": fmt.Sprintf("Keywords Test Description %d", i),
			"tags":        []string{"keyword-tag1", "keyword-tag2"},
			"built_in":    false,
			"mentionable": true,
			"automated":   true,
		}
		_, err = store.SaveAssistant(assistant)
		assert.Nil(t, err)
	}

	// Test delete by keywords
	count, err = store.DeleteAssistants(AssistantFilter{
		Keywords: "Keywords Test",
	})
	assert.Nil(t, err)
	assert.Greater(t, count, int64(0))

	// Verify all assistants are deleted
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

	// Create multiple assistants for pagination testing
	mentionable := true
	automated := true
	for i := 0; i < 25; i++ {
		tagsJSON, err := jsoniter.MarshalToString([]string{fmt.Sprintf("tag%d", i%5)})
		if err != nil {
			t.Fatal(err)
		}

		// Alternate mentionable and automated flags
		if i%2 == 0 {
			mentionable = !mentionable
		}
		if i%3 == 0 {
			automated = !automated
		}

		assistant := map[string]interface{}{
			"name":        fmt.Sprintf("Assistant %d", i),
			"type":        "assistant",
			"connector":   fmt.Sprintf("connector%d", i%3),
			"description": fmt.Sprintf("Description %d", i),
			"tags":        tagsJSON,
			"sort":        9999 - i,
			"updated_at":  time.Now().Add(time.Duration(-i) * time.Hour),
			"built_in":    i%2 == 0,
			"mentionable": mentionable,
			"automated":   automated,
		}
		_, err = store.SaveAssistant(assistant)
		assert.Nil(t, err)
	}

	// Test first page
	resp, err := store.GetAssistants(AssistantFilter{
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 10, len(resp.Data))
	assert.Equal(t, int64(25), resp.Total)
	assert.Equal(t, 3, resp.PageCnt)
	assert.Equal(t, 2, resp.Next)
	assert.Equal(t, 0, resp.Prev)

	// Verify sorting order (sort ASC, updated_at DESC)
	for i := 1; i < len(resp.Data); i++ {
		curr := resp.Data[i]["sort"].(int64)
		prev := resp.Data[i-1]["sort"].(int64)
		assert.True(t, curr >= prev, "Results should be sorted by sort ASC")

		// When sort values are equal, check updated_at if both values exist
		if curr == prev {
			currTime, currOk := resp.Data[i]["updated_at"].(time.Time)
			prevTime, prevOk := resp.Data[i-1]["updated_at"].(time.Time)

			// Only compare times if both values exist
			if currOk && prevOk {
				assert.True(t, currTime.Before(prevTime) || currTime.Equal(prevTime),
					"Results with same sort should be ordered by updated_at DESC")
			}
		}
	}

	// Test second page
	resp, err = store.GetAssistants(AssistantFilter{
		Page:     2,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 10, len(resp.Data))
	assert.Equal(t, 3, resp.Next)
	assert.Equal(t, 1, resp.Prev)

	// Test last page
	resp, err = store.GetAssistants(AssistantFilter{
		Page:     3,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 5, len(resp.Data))
	assert.Equal(t, 0, resp.Next)
	assert.Equal(t, 2, resp.Prev)

	// Test filtering with tags
	resp, err = store.GetAssistants(AssistantFilter{
		Tags:     []string{"tag0"},
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 5, len(resp.Data))

	// Test filtering with keywords
	resp, err = store.GetAssistants(AssistantFilter{
		Keywords: "Assistant 1",
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering with connector
	resp, err = store.GetAssistants(AssistantFilter{
		Connector: "connector0",
		Page:      1,
		PageSize:  10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering with mentionable
	mentionableTrue := true
	resp, err = store.GetAssistants(AssistantFilter{
		Mentionable: &mentionableTrue,
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering with automated
	automatedTrue := true
	resp, err = store.GetAssistants(AssistantFilter{
		Automated: &automatedTrue,
		Page:      1,
		PageSize:  10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering with built_in
	builtInTrue := true
	resp, err = store.GetAssistants(AssistantFilter{
		BuiltIn:  &builtInTrue,
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	for _, assistant := range resp.Data {
		assert.Equal(t, int64(1), assistant["built_in"], "All assistants should be built-in")
	}

	builtInFalse := false
	resp, err = store.GetAssistants(AssistantFilter{
		BuiltIn:  &builtInFalse,
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	for _, assistant := range resp.Data {
		assert.Equal(t, int64(0), assistant["built_in"], "All assistants should not be built-in")
	}

	// Test assistant_id with other filters
	// First get an assistant_id from previous results
	firstAssistantID := resp.Data[0]["assistant_id"].(string)

	// Test exact match with assistant_id
	resp, err = store.GetAssistants(AssistantFilter{
		AssistantID: firstAssistantID,
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(resp.Data))
	assert.Equal(t, firstAssistantID, resp.Data[0]["assistant_id"])

	// Test assistant_id with other filters
	resp, err = store.GetAssistants(AssistantFilter{
		AssistantID: firstAssistantID,
		Select:      []string{"name", "assistant_id", "description"},
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(resp.Data))
	assert.Equal(t, firstAssistantID, resp.Data[0]["assistant_id"])
	// Verify only selected fields are returned
	assert.Contains(t, resp.Data[0], "name")
	assert.Contains(t, resp.Data[0], "assistant_id")
	assert.Contains(t, resp.Data[0], "description")
	assert.NotContains(t, resp.Data[0], "tags")
	assert.NotContains(t, resp.Data[0], "options")

	// Test non-existent assistant_id
	resp, err = store.GetAssistants(AssistantFilter{
		AssistantID: "non-existent-id",
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))

	// Test filtering with select fields
	resp, err = store.GetAssistants(AssistantFilter{
		Select:   []string{"name", "description", "tags"},
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 10, len(resp.Data))
	// Verify only selected fields are returned
	for _, item := range resp.Data {
		// These fields should exist
		assert.Contains(t, item, "name")
		assert.Contains(t, item, "description")
		assert.Contains(t, item, "tags")
		// These fields should not exist
		assert.NotContains(t, item, "options")
		assert.NotContains(t, item, "prompts")
		assert.NotContains(t, item, "flows")
		assert.NotContains(t, item, "files")
		assert.NotContains(t, item, "functions")
		assert.NotContains(t, item, "permissions")
	}

	// Test filtering with select fields and other filters combined
	resp, err = store.GetAssistants(AssistantFilter{
		Tags:     []string{"tag0"},
		Keywords: "Assistant",
		Select:   []string{"name", "tags"},
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	// Verify only selected fields are returned
	for _, item := range resp.Data {
		// These fields should exist
		assert.Contains(t, item, "name")
		assert.Contains(t, item, "tags")
		// These fields should not exist
		assert.NotContains(t, item, "description")
		assert.NotContains(t, item, "options")
		assert.NotContains(t, item, "prompts")
		assert.NotContains(t, item, "flows")
		assert.NotContains(t, item, "files")
		assert.NotContains(t, item, "functions")
		assert.NotContains(t, item, "permissions")
	}

	// Test combined filters
	resp, err = store.GetAssistants(AssistantFilter{
		Tags:        []string{"tag0"},
		Keywords:    "Assistant",
		Connector:   "connector0",
		Mentionable: &mentionableTrue,
		Automated:   &automatedTrue,
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)

	// Now test the delete operations
	// Test delete by connector
	count, err := store.DeleteAssistants(AssistantFilter{
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
	builtInTrue = true
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
		Keywords: "Assistant",
	})
	assert.Nil(t, err)
	assert.Greater(t, count, int64(0))

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
		if !expectedTags[tag] {
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
