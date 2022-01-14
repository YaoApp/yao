package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestProcessMapDel(t *testing.T) {
	args := []interface{}{
		map[string]interface{}{"foo": "Value1", "bar": "Value2"},
		"bar",
	}
	new := gou.NewProcess("xiang.helper.MapDel", args...).Run().(map[string]interface{})
	_, has := new["bar"]
	assert.False(t, has)
	assert.Equal(t, "Value1", new["foo"])
}

func TestProcessGetSet(t *testing.T) {
	args := []interface{}{
		map[string]interface{}{"foo": "Value1"},
		"bar",
		"Value2",
	}
	new := gou.NewProcess("xiang.helper.MapSet", args...).Run().(map[string]interface{})
	assert.Equal(t, "Value1", new["foo"])
	assert.Equal(t, "Value2", new["bar"])

	bar := gou.NewProcess("xiang.helper.MapGet", new, "bar").Run().(string)
	assert.Equal(t, "Value2", bar)
}

func TestProcessMapKeys(t *testing.T) {
	args := []interface{}{
		map[string]interface{}{"foo": "Value1", "bar": "Value2"},
	}
	keys := gou.NewProcess("xiang.helper.MapKeys", args...).Run().([]string)
	assert.Contains(t, keys, "foo")
	assert.Contains(t, keys, "bar")
}

func TestProcessMapValues(t *testing.T) {
	args := []interface{}{
		map[string]interface{}{"foo": "Value1", "bar": "Value2"},
	}
	values := gou.NewProcess("xiang.helper.MapValues", args...).Run().([]interface{})
	assert.Contains(t, values, "Value1")
	assert.Contains(t, values, "Value2")
}

func TestProcessMapMultiDel(t *testing.T) {
	args := []interface{}{
		map[string]interface{}{"foo": "Value1", "bar": "Value2"},
		"foo",
		"bar",
	}
	new := gou.NewProcess("xiang.helper.MapMultiDel", args...).Run().(map[string]interface{})
	assert.Nil(t, new["foo"])
	assert.Nil(t, new["bar"])
}

func TestProcessMapToArray(t *testing.T) {

	arr := gou.NewProcess("xiang.helper.MapToArray", map[string]interface{}{
		"foo": "Value1",
		"bar": "Value2",
	}).Run().([]map[string]interface{})

	assert.Len(t, arr, 2)

	assert.True(t, arr[0]["key"] == "foo" || arr[0]["key"] == "bar")
	assert.True(t, arr[0]["value"] == "Value1" || arr[0]["value"] == "Value2")
}
