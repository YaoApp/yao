package neo

import (
	"fmt"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func prepare(t *testing.T) {
	test.Prepare(t, config.Conf)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// Clean up the test data before each test
	p, err := process.Of("neo.assistant.search", map[string]interface{}{
		"page":     1,
		"pagesize": 1000, // Use a large page size to get all records
	})
	if err != nil {
		t.Fatal(err)
	}
	output, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	res := any.Of(output).Map()
	items := res.Get("data")
	if items != nil {
		for _, item := range items.([]map[string]interface{}) {
			assistantID := item["assistant_id"].(string)
			p, err = process.Of("neo.assistant.delete", assistantID)
			if err != nil {
				t.Fatal(err)
			}
			_, err = p.Exec()
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// Verify cleanup
	p, err = process.Of("neo.assistant.search")
	if err != nil {
		t.Fatal(err)
	}
	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	res = any.Of(output).Map()
	total := res.Get("total")
	if total != nil && any.Of(total).CInt() > 0 {
		t.Fatalf("Failed to clean up test data, %d records remaining", any.Of(total).CInt())
	}

	check(t)
}

func TestProcessAssistantCRUD(t *testing.T) {
	prepare(t)
	defer test.Clean()

	// Create an assistant
	tagsJSON, err := jsoniter.MarshalToString([]string{"tag1", "tag2", "tag3"})
	if err != nil {
		t.Fatal(err)
	}

	optionsJSON, err := jsoniter.MarshalToString(map[string]interface{}{
		"model": "gpt-4",
	})
	if err != nil {
		t.Fatal(err)
	}

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

	// Test processAssistantAdd
	p, err := process.Of("neo.assistant.add", assistant)
	if err != nil {
		t.Fatal(err)
	}

	output, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	res := any.Of(output).Map()
	assert.Equal(t, "Test Assistant", res.Get("name"))
	assert.NotEmpty(t, res.Get("assistant_id"))
	assistantID := res.Get("assistant_id").(string)

	// Test processAssistantSearch - no filter
	p, err = process.Of("neo.assistant.search")
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	searchRes := any.Of(output).Map()
	total := searchRes.Get("total")
	if total == nil {
		total = int64(0)
	}
	assert.Equal(t, int64(1), total)

	items := searchRes.Get("data")
	if items == nil {
		items = []map[string]interface{}{}
	}
	assert.Equal(t, 1, len(items.([]map[string]interface{})))

	// Test processAssistantSearch - with filter
	p, err = process.Of("neo.assistant.search", map[string]interface{}{
		"tags":     []string{"tag1"},
		"page":     1,
		"pagesize": 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	searchRes = any.Of(output).Map()
	total = searchRes.Get("total")
	if total == nil {
		total = int64(0)
	}
	assert.Equal(t, int64(1), total)

	items = searchRes.Get("data")
	if items == nil {
		items = []map[string]interface{}{}
	}
	assert.Equal(t, 1, len(items.([]map[string]interface{})))

	// Test processAssistantSave (Update)
	assistant["assistant_id"] = assistantID
	assistant["name"] = "Updated Assistant"
	p, err = process.Of("neo.assistant.save", assistant)
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	res = any.Of(output).Map()
	assert.Equal(t, "Updated Assistant", res.Get("name"))

	// Test processAssistantDelete
	p, err = process.Of("neo.assistant.delete", assistantID)
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	deleteRes := any.Of(output).Map()
	assert.Equal(t, "ok", deleteRes.Get("message"))

	// Verify deletion with search
	p, err = process.Of("neo.assistant.search")
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	searchRes = any.Of(output).Map()
	total = searchRes.Get("total")
	if total == nil {
		total = int64(0)
	}
	assert.Equal(t, int64(0), total)

	items = searchRes.Get("data")
	if items == nil {
		items = []map[string]interface{}{}
	}
	assert.Equal(t, 0, len(items.([]map[string]interface{})))
}

func TestProcessAssistantSearchPagination(t *testing.T) {
	prepare(t)
	defer test.Clean()

	// Create multiple assistants for pagination testing
	for i := 0; i < 25; i++ {
		tagsJSON, err := jsoniter.MarshalToString([]string{fmt.Sprintf("tag%d", i%5)})
		if err != nil {
			t.Fatal(err)
		}

		assistant := map[string]interface{}{
			"name":        fmt.Sprintf("Assistant %d", i),
			"type":        "assistant",
			"connector":   fmt.Sprintf("connector%d", i%3),
			"description": fmt.Sprintf("Description %d", i),
			"tags":        tagsJSON,
			"mentionable": i%2 == 0,
			"automated":   i%3 == 0,
		}

		p, err := process.Of("neo.assistant.add", assistant)
		if err != nil {
			t.Fatal(err)
		}

		_, err = p.Exec()
		if err != nil {
			t.Fatal(err)
		}
	}

	// Test first page
	p, err := process.Of("neo.assistant.search", map[string]interface{}{
		"page":     1,
		"pagesize": 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	output, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	res := any.Of(output).Map()
	total := res.Get("total")
	if total == nil {
		total = int64(0)
	}
	assert.Equal(t, int64(25), total)

	items := res.Get("data")
	if items == nil {
		items = []map[string]interface{}{}
	}
	assert.Equal(t, 10, len(items.([]map[string]interface{})))

	pageCnt := res.Get("pagecnt")
	if pageCnt == nil {
		pageCnt = 1
	}
	assert.Equal(t, 3, pageCnt)

	// Test second page
	p, err = process.Of("neo.assistant.search", map[string]interface{}{
		"page":     2,
		"pagesize": 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	res = any.Of(output).Map()
	items = res.Get("data")
	if items == nil {
		items = []map[string]interface{}{}
	}
	assert.Equal(t, 10, len(items.([]map[string]interface{})))

	// Test last page
	p, err = process.Of("neo.assistant.search", map[string]interface{}{
		"page":     3,
		"pagesize": 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	res = any.Of(output).Map()
	items = res.Get("data")
	if items == nil {
		items = []map[string]interface{}{}
	}
	assert.Equal(t, 5, len(items.([]map[string]interface{})))

	// Test filtering with tags
	p, err = process.Of("neo.assistant.search", map[string]interface{}{
		"tags":     []string{"tag0"},
		"page":     1,
		"pagesize": 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	res = any.Of(output).Map()
	items = res.Get("data")
	if items == nil {
		items = []map[string]interface{}{}
	}
	assert.Equal(t, 5, len(items.([]map[string]interface{})))
}

func TestProcessAssistantValidation(t *testing.T) {
	prepare(t)
	defer test.Clean()

	// Test missing required fields
	p, err := process.Of("neo.assistant.add", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NotNil(t, err)

	// Test invalid assistant ID for delete
	p, err = process.Of("neo.assistant.delete", "non-existent-id")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NotNil(t, err)

	// Test invalid page number
	p, err = process.Of("neo.assistant.search", map[string]interface{}{
		"page":     -1,
		"pagesize": 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	output, err := p.Exec()
	assert.Nil(t, err)

	res := any.Of(output).Map()
	total := res.Get("total")
	if total == nil {
		total = int64(0)
	}
	assert.Equal(t, int64(0), total)
}
