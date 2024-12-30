package neo

import (
	"fmt"
	"testing"

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

	// Create an assistant with string JSON fields
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

	// Test processAssistantCreate with string JSON
	p, err := process.Of("neo.assistant.create", assistant)
	if err != nil {
		t.Fatal(err)
	}

	output, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assistantID := output
	assert.NotNil(t, assistantID)

	// Test processAssistantFind
	p, err = process.Of("neo.assistant.find", assistantID)
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	foundAssistant := output.(map[string]interface{})
	assert.Equal(t, assistantID, foundAssistant["assistant_id"])
	assert.Equal(t, "Test Assistant", foundAssistant["name"])
	assert.Equal(t, []interface{}{"tag1", "tag2", "tag3"}, foundAssistant["tags"])
	assert.Equal(t, map[string]interface{}{"model": "gpt-4"}, foundAssistant["options"])

	// Test processAssistantFind with non-existent ID
	p, err = process.Of("neo.assistant.find", "non-existent-id")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Assistant not found")

	// Test with native type JSON fields
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

	// Test processAssistantCreate with native types
	p, err = process.Of("neo.assistant.create", assistant2)
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assistant2ID := output
	assert.NotNil(t, assistant2ID)

	// Test with nil JSON fields
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

	// Test processAssistantCreate with nil fields
	p, err = process.Of("neo.assistant.create", assistant3)
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assistant3ID := output
	assert.NotNil(t, assistant3ID)

	// Test processAssistantSearch to verify all assistants
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
	assert.Equal(t, int64(3), total)

	items := searchRes.Get("data")
	if items == nil {
		items = []map[string]interface{}{}
	}
	assert.Equal(t, 3, len(items.([]map[string]interface{})))

	// Verify each assistant's JSON fields
	for _, item := range items.([]map[string]interface{}) {
		switch item["assistant_id"].(string) {
		case assistantID:
			assert.Equal(t, []interface{}{"tag1", "tag2", "tag3"}, item["tags"])
			assert.Equal(t, map[string]interface{}{"model": "gpt-4"}, item["options"])
		case assistant2ID:
			assert.Equal(t, []interface{}{"tag1", "tag2", "tag3"}, item["tags"])
			assert.Equal(t, map[string]interface{}{"model": "gpt-4"}, item["options"])
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
		case assistant3ID:
			assert.Nil(t, item["tags"])
			assert.Nil(t, item["options"])
			assert.Nil(t, item["prompts"])
			assert.Nil(t, item["flows"])
			assert.Nil(t, item["files"])
			assert.Nil(t, item["functions"])
			assert.Nil(t, item["permissions"])
		}
	}

	// Test updating with mixed JSON formats
	assistant2["assistant_id"] = assistant2ID
	assistant2["tags"] = `["tag4", "tag5"]`
	assistant2["options"] = map[string]interface{}{"model": "gpt-3.5"}
	p, err = process.Of("neo.assistant.save", assistant2)
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	savedID := output
	assert.NotNil(t, savedID)

	// Double check with a new search
	p, err = process.Of("neo.assistant.search")
	if err != nil {
		t.Fatal(err)
	}

	output, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	searchRes = any.Of(output).Map()
	items = searchRes.Get("data")
	found := false
	for _, item := range items.([]map[string]interface{}) {
		if item["assistant_id"].(string) == assistant2ID {
			found = true
			assert.Equal(t, []interface{}{"tag4", "tag5"}, item["tags"])
			assert.Equal(t, map[string]interface{}{"model": "gpt-3.5"}, item["options"])
			break
		}
	}
	assert.True(t, found)

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

	// Delete remaining assistants
	p, err = process.Of("neo.assistant.delete", assistant2ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Exec()
	assert.Nil(t, err)

	p, err = process.Of("neo.assistant.delete", assistant3ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Exec()
	assert.Nil(t, err)

	// Verify all assistants are deleted
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
}

func TestProcessAssistantSearchPagination(t *testing.T) {
	prepare(t)
	defer test.Clean()

	// Create multiple assistants for pagination testing
	for i := 0; i < 25; i++ {
		assistant := map[string]interface{}{
			"name":        fmt.Sprintf("Assistant %d", i),
			"type":        "assistant",
			"connector":   fmt.Sprintf("connector%d", i%3),
			"description": fmt.Sprintf("Description %d", i),
			"tags":        []string{fmt.Sprintf("tag%d", i%5)},
			"mentionable": i%2 == 0,
			"automated":   i%3 == 0,
		}

		p, err := process.Of("neo.assistant.create", assistant)
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
	p, err := process.Of("neo.assistant.create", map[string]interface{}{})
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

	// Test invalid assistant ID for find
	p, err = process.Of("neo.assistant.find", "non-existent-id")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Assistant not found")

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
