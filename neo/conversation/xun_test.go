package conversation

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
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation")

	err := capsule.Schema().DropTableIfExists("__unit_test_conversation")
	if err != nil {
		t.Fatal(err)
	}

	conv, err := NewXun(Setting{
		Connector: "default",
		Table:     "__unit_test_conversation",
	})

	if err != nil {
		t.Error(err)
		return
	}

	has, err := capsule.Schema().HasTable("__unit_test_conversation")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, has)

	// validate the table
	tab, err := conv.schema.GetTable(conv.setting.Table)
	if err != nil {
		t.Fatal(err)
	}

	fields := []string{"id", "sid", "cid", "rid", "role", "name", "content", "created_at", "updated_at", "expired_at"}
	for _, field := range fields {
		assert.Equal(t, true, tab.HasColumn(field))
	}

	conv, err = NewXun(Setting{
		Connector: "default",
		Table:     "__unit_test_conversation",
	})

	has, err = capsule.Schema().HasTable("__unit_test_conversation")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, has)
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

	defer sch.DropTableIfExists("__unit_test_conversation")

	sch.DropTableIfExists("__unit_test_conversation")
	conv, err := NewXun(Setting{
		Connector: "mysql",
		Table:     "__unit_test_conversation",
	})

	if err != nil {
		t.Error(err)
		return
	}

	has, err := sch.HasTable("__unit_test_conversation")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, has)

	// validate the table
	tab, err := conv.schema.GetTable(conv.setting.Table)
	if err != nil {
		t.Fatal(err)
	}

	fields := []string{"id", "sid", "cid", "rid", "role", "name", "content", "created_at", "updated_at", "expired_at"}
	for _, field := range fields {
		assert.Equal(t, true, tab.HasColumn(field))
	}

	conv, err = NewXun(Setting{
		Connector: "default",
		Table:     "__unit_test_conversation",
	})

	has, err = sch.HasTable("__unit_test_conversation")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, has)
}

func TestXunSaveAndGetHistory(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation")

	err := capsule.Schema().DropTableIfExists("__unit_test_conversation")
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
	}, cid)
	assert.Nil(t, err)

	// get the history
	data, err := conv.GetHistory("123456", cid)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(data))
}

func TestXunSaveAndGetRequest(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation")

	err := capsule.Schema().DropTableIfExists("__unit_test_conversation")
	if err != nil {
		t.Fatal(err)
	}

	conv, err := NewXun(Setting{
		Connector: "default",
		Table:     "__unit_test_conversation",
		TTL:       3600,
	})

	// save the history
	err = conv.SaveRequest("123456", "912836", "test.command", []map[string]interface{}{
		{"role": "user", "name": "user1", "content": "hello"},
		{"role": "assistant", "name": "user1", "content": "Hello there, how"},
	})
	assert.Nil(t, err)

	// get the history
	data, err := conv.GetRequest("123456", "912836")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(data))
}

func TestXunSaveAndGetHistoryWithCID(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation")

	err := capsule.Schema().DropTableIfExists("__unit_test_conversation")
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
	err = conv.SaveHistory(sid, messages, cid)
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
	err = conv.SaveHistory(sid, moreMessages, anotherCID)
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
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation")
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation_chat")

	// Drop both tables before test
	err := capsule.Schema().DropTableIfExists("__unit_test_conversation")
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
		err = conv.SaveHistory(sid, messages, chatID)
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
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation")
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
	err = conv.SaveHistory(sid, messages, cid)
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
	defer capsule.Schema().DropTableIfExists("__unit_test_conversation")
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
		err = conv.SaveHistory(sid, messages, cid)
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
