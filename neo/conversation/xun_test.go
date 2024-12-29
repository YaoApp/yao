package conversation

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

	conv, err := NewXun(Setting{
		Connector: "default",
		Table:     "__unit_test_conversation",
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

	// validate the history table
	tab, err := conv.schema.GetTable(conv.getHistoryTable())
	if err != nil {
		t.Fatal(err)
	}

	fields := []string{"id", "sid", "cid", "uid", "role", "name", "content", "context", "created_at", "updated_at", "expired_at"}
	for _, field := range fields {
		assert.Equal(t, true, tab.HasColumn(field))
	}

	// validate the chat table
	tab, err = conv.schema.GetTable(conv.getChatTable())
	if err != nil {
		t.Fatal(err)
	}

	chatFields := []string{"id", "chat_id", "title", "sid", "created_at", "updated_at"}
	for _, field := range chatFields {
		assert.Equal(t, true, tab.HasColumn(field))
	}

	// validate the assistant table
	tab, err = conv.schema.GetTable(conv.getAssistantTable())
	if err != nil {
		t.Fatal(err)
	}

	assistantFields := []string{"id", "assistant_id", "type", "name", "avatar", "connector", "description", "options", "prompts", "flows", "files", "functions", "tags", "readonly", "permissions", "automated", "mentionable", "created_at", "updated_at"}
	for _, field := range assistantFields {
		assert.Equal(t, true, tab.HasColumn(field))
	}
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

	conv, err := NewXun(Setting{
		Connector: "mysql",
		Table:     "__unit_test_conversation",
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

	// validate the history table
	tab, err := conv.schema.GetTable(conv.getHistoryTable())
	if err != nil {
		t.Fatal(err)
	}

	fields := []string{"id", "sid", "cid", "uid", "role", "name", "content", "context", "created_at", "updated_at", "expired_at"}
	for _, field := range fields {
		assert.Equal(t, true, tab.HasColumn(field))
	}

	// validate the chat table
	tab, err = conv.schema.GetTable(conv.getChatTable())
	if err != nil {
		t.Fatal(err)
	}

	chatFields := []string{"id", "chat_id", "title", "sid", "created_at", "updated_at"}
	for _, field := range chatFields {
		assert.Equal(t, true, tab.HasColumn(field))
	}

	// validate the assistant table
	tab, err = conv.schema.GetTable(conv.getAssistantTable())
	if err != nil {
		t.Fatal(err)
	}

	assistantFields := []string{"id", "assistant_id", "type", "name", "avatar", "connector", "description", "options", "prompts", "flows", "files", "functions", "tags", "readonly", "permissions", "automated", "mentionable", "created_at", "updated_at"}
	for _, field := range assistantFields {
		assert.Equal(t, true, tab.HasColumn(field))
	}
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

	conv, err := NewXun(Setting{
		Connector: "default",
		Table:     "__unit_test_conversation",
		TTL:       3600,
	})

	// save the history
	cid := "123456"
	err = conv.SaveHistory("123456", []map[string]interface{}{
		{"role": "user", "name": "user1", "content": "hello"},
		{"role": "assistant", "name": "user1", "content": "Hello there, how"},
	}, cid, nil)
	assert.Nil(t, err)

	// get the history
	data, err := conv.GetHistory("123456", cid)
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

	conv, err := NewXun(Setting{
		Connector: "default",
		Table:     "__unit_test_conversation",
		TTL:       3600,
	})

	// save the history with specific cid
	sid := "123456"
	cid := "789012"
	messages := []map[string]interface{}{
		{"role": "user", "name": "user1", "content": "hello"},
		{"role": "assistant", "name": "assistant1", "content": "Hi! How can I help you?"},
	}
	err = conv.SaveHistory(sid, messages, cid, nil)
	assert.Nil(t, err)

	// get the history for specific cid
	data, err := conv.GetHistory(sid, cid)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(data))

	// save another message with different cid
	anotherCID := "345678"
	moreMessages := []map[string]interface{}{
		{"role": "user", "name": "user1", "content": "another message"},
	}
	err = conv.SaveHistory(sid, moreMessages, anotherCID, nil)
	assert.Nil(t, err)

	// get history for the first cid - should still be 2 messages
	data, err = conv.GetHistory(sid, cid)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(data))

	// get history for the second cid - should be 1 message
	data, err = conv.GetHistory(sid, anotherCID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(data))

	// get all history for the sid without specifying cid
	allData, err := conv.GetHistory(sid, cid)
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

	conv, err := NewXun(Setting{
		Connector: "default",
		Table:     "__unit_test_conversation",
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
		// First create the chat with a title
		err = conv.newQueryChat().Insert(map[string]interface{}{
			"chat_id":    chatID,
			"title":      fmt.Sprintf("Test Chat %d", i),
			"sid":        sid,
			"created_at": time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}

		// Then save the history
		err = conv.SaveHistory(sid, messages, chatID, nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Test getting chats with default filter
	filter := ChatFilter{
		PageSize: 10,
		Order:    "desc",
	}
	groups, err := conv.GetChats(sid, filter)
	if err != nil {
		t.Fatal(err)
	}

	assert.Greater(t, len(groups.Groups), 0)

	// Test with keywords
	filter.Keywords = "test"
	groups, err = conv.GetChats(sid, filter)
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

	conv, err := NewXun(Setting{
		Connector: "default",
		Table:     "__unit_test_conversation",
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
	err = conv.SaveHistory(sid, messages, cid, nil)
	assert.Nil(t, err)

	// Verify chat exists
	chat, err := conv.GetChat(sid, cid)
	assert.Nil(t, err)
	assert.NotNil(t, chat)

	// Delete the chat
	err = conv.DeleteChat(sid, cid)
	assert.Nil(t, err)

	// Verify chat is deleted
	chat, err = conv.GetChat(sid, cid)
	assert.Nil(t, err)
	assert.Equal(t, (*ChatInfo)(nil), chat)
}

func TestXunDeleteAllChats(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_history")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")

	conv, err := NewXun(Setting{
		Connector: "default",
		Table:     "__unit_test_conversation",
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
		err = conv.SaveHistory(sid, messages, cid, nil)
		assert.Nil(t, err)
	}

	// Verify chats exist
	response, err := conv.GetChats(sid, ChatFilter{})
	assert.Nil(t, err)
	assert.Greater(t, response.Total, int64(0))

	// Delete all chats
	err = conv.DeleteAllChats(sid)
	assert.Nil(t, err)

	// Verify all chats are deleted
	response, err = conv.GetChats(sid, ChatFilter{})
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

	conv, err := NewXun(Setting{
		Connector: "default",
		Table:     "__unit_test_conversation",
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
		"tags":        tagsJSON,
		"options":     optionsJSON,
		"mentionable": true,
		"automated":   true,
	}

	// Test SaveAssistant (Create) with string JSON
	v, err := conv.SaveAssistant(assistant)
	assert.Nil(t, err)
	assistantID := v.(string)
	assert.NotEmpty(t, assistantID)

	// Test case 2: JSON fields as native types
	assistant2 := map[string]interface{}{
		"name":        "Test Assistant 2",
		"type":        "assistant",
		"avatar":      "https://example.com/avatar2.png",
		"connector":   "openai",
		"description": "Test Description 2",
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
	v, err = conv.SaveAssistant(assistant2)
	assert.Nil(t, err)
	assistant2ID := v.(string)
	assert.NotEmpty(t, assistant2ID)

	// Test case 3: Test with nil JSON fields
	assistant3 := map[string]interface{}{
		"name":        "Test Assistant 3",
		"type":        "assistant",
		"connector":   "openai",
		"description": "Test Description 3",
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
	v, err = conv.SaveAssistant(assistant3)
	assert.Nil(t, err)
	assistant3ID := v.(string)
	assert.NotEmpty(t, assistant3ID)

	// Test GetAssistants to verify JSON fields are properly stored
	resp, err := conv.GetAssistants(AssistantFilter{})
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
	_, err = conv.SaveAssistant(assistant2)
	assert.Nil(t, err)

	// Verify update
	resp, err = conv.GetAssistants(AssistantFilter{})
	assert.Nil(t, err)
	for _, item := range resp.Data {
		if item["assistant_id"].(string) == assistant2ID {
			// Now we expect parsed JSON values
			assert.Equal(t, []interface{}{"tag1", "tag2", "tag3"}, item["tags"])
			assert.Equal(t, map[string]interface{}{"model": "gpt-4"}, item["options"])
			break
		}
	}

	// Test DeleteAssistant
	err = conv.DeleteAssistant(assistantID)
	assert.Nil(t, err)
	err = conv.DeleteAssistant(assistant2ID)
	assert.Nil(t, err)
	err = conv.DeleteAssistant(assistant3ID)
	assert.Nil(t, err)

	resp, err = conv.GetAssistants(AssistantFilter{})
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

	conv, err := NewXun(Setting{
		Connector: "default",
		Table:     "__unit_test_conversation",
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
			"mentionable": mentionable,
			"automated":   automated,
		}
		_, err = conv.SaveAssistant(assistant)
		assert.Nil(t, err)
	}

	// Test first page
	resp, err := conv.GetAssistants(AssistantFilter{
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 10, len(resp.Data))
	assert.Equal(t, int64(25), resp.Total)
	assert.Equal(t, 3, resp.PageCnt)
	assert.Equal(t, 2, resp.Next)
	assert.Equal(t, 0, resp.Prev)

	// Test second page
	resp, err = conv.GetAssistants(AssistantFilter{
		Page:     2,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 10, len(resp.Data))
	assert.Equal(t, 3, resp.Next)
	assert.Equal(t, 1, resp.Prev)

	// Test last page
	resp, err = conv.GetAssistants(AssistantFilter{
		Page:     3,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 5, len(resp.Data))
	assert.Equal(t, 0, resp.Next)
	assert.Equal(t, 2, resp.Prev)

	// Test filtering with tags
	resp, err = conv.GetAssistants(AssistantFilter{
		Tags:     []string{"tag0"},
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 5, len(resp.Data))

	// Test filtering with keywords
	resp, err = conv.GetAssistants(AssistantFilter{
		Keywords: "Assistant 1",
		Page:     1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering with connector
	resp, err = conv.GetAssistants(AssistantFilter{
		Connector: "connector0",
		Page:      1,
		PageSize:  10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering with mentionable
	mentionableTrue := true
	resp, err = conv.GetAssistants(AssistantFilter{
		Mentionable: &mentionableTrue,
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering with automated
	automatedTrue := true
	resp, err = conv.GetAssistants(AssistantFilter{
		Automated: &automatedTrue,
		Page:      1,
		PageSize:  10,
	})
	assert.Nil(t, err)
	assert.Greater(t, len(resp.Data), 0)

	// Test filtering by assistant_id
	// First get an assistant_id from previous results
	firstAssistantID := resp.Data[0]["assistant_id"].(string)

	// Test exact match with assistant_id
	resp, err = conv.GetAssistants(AssistantFilter{
		AssistantID: firstAssistantID,
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(resp.Data))
	assert.Equal(t, firstAssistantID, resp.Data[0]["assistant_id"])

	// Test assistant_id with other filters
	resp, err = conv.GetAssistants(AssistantFilter{
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
	resp, err = conv.GetAssistants(AssistantFilter{
		AssistantID: "non-existent-id",
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resp.Data))

	// Test combined filters
	resp, err = conv.GetAssistants(AssistantFilter{
		Tags:        []string{"tag0"},
		Keywords:    "Assistant",
		Connector:   "connector0",
		Mentionable: &mentionableTrue,
		Automated:   &automatedTrue,
		Page:        1,
		PageSize:    10,
	})
	assert.Nil(t, err)

	// Test filtering with select fields
	resp, err = conv.GetAssistants(AssistantFilter{
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
	resp, err = conv.GetAssistants(AssistantFilter{
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
}
