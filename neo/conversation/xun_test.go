package conversation

import (
	"testing"

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
	err = conv.SaveHistory("123456", []map[string]interface{}{
		{"role": "user", "name": "user1", "content": "hello"},
		{"role": "assistant", "name": "user1", "content": "Hello there, how"},
	})
	assert.Nil(t, err)

	// get the history
	data, err := conv.GetHistory("123456")
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
