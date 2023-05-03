package driver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/command/query"
	"github.com/yaoapp/yao/test"
)

func TestMemorySetGetDel(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem := prepare(t)
	err := mem.Set("table.delete", Command{
		ID:          "table.delete",
		Name:        "Generate test data for the table",
		Description: "Generate test data for the table",
		Stack:       "Table.*",
		Path:        "*",
		Args: []map[string]interface{}{
			{
				"name":        "data",
				"type":        "Array",
				"description": "The data sets to generate",
				"required":    true,
				"default":     []interface{}{},
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	cmd, has := mem.Get("table.delete")
	if !has {
		t.Fatal("table.delete not found")
	}

	assert.Equal(t, "table.delete", cmd.ID)
	mem.Del("table.delete")

	_, has = mem.Get("table.delete")
	assert.False(t, has)
}

func TestMemoryMatch(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem := prepare(t)
	id, err := mem.Match(query.Param{Stack: "Table.Page.pet"}, "Generate table test data")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "table.data", id)

	id, err = mem.Match(query.Param{Stack: "Form.Page.pet", Path: "/Form/pet"}, "Generate table test data")
	assert.ErrorContains(t, err, "no related command found")
}

func prepare(t *testing.T) *Memory {
	mem, err := NewMemory("gpt-3_5-turbo", nil)
	if err != nil {
		t.Fatal(err)
	}

	mem.Set("table.data", Command{
		ID:          "table.data",
		Name:        "Generate test data for the table",
		Description: "Generate test data for the table",
		Stack:       "Table.*",
		Path:        "Table.*",
		Args: []map[string]interface{}{
			{
				"name":        "data",
				"type":        "Array",
				"description": "The data sets to generate",
				"required":    true,
				"default":     []interface{}{},
			},
		},
	})

	return mem
}
