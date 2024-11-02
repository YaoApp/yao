package utils_test

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestProcessTreeFlatten(t *testing.T) {
	testPrepare()
	bytes := []byte(`[
		{
		  "id": 1,
		  "parent": null,
		  "children": [{ "children": [], "id": 5, "parent": 1 }]
		},
		{ "id": 2, "parent": null, "children": [] },
		{ "id": 3, "parent": null, "children": [] }
	  ]`)

	var data interface{}
	err := jsoniter.Unmarshal(bytes, &data)
	if err != nil {
		t.Fatal(err)
	}

	rows := process.New("utils.tree.Flatten", data, map[string]interface{}{"primary": "id", "children": "children", "parent": "parent"}).Run().([]interface{})
	assert.Equal(t, 4, len(rows))
	assert.Equal(t, float64(1), rows[1].(map[string]interface{})["parent"])
}
