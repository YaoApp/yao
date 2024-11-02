package utils_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/yao/utils"
)

func TestProcessParseQuery(t *testing.T) {
	utils.Init()
	args := []interface{}{"a=1&b=2&c=3&c=4"}
	result, err := process.New("utils.url.ParseQuery", args...).Exec()
	if err != nil {
		t.Errorf("ProcessParseQuery error: %s", err)
	}

	assert.Equal(t, result.(url.Values).Get("a"), "1")
	assert.Equal(t, result.(url.Values).Get("b"), "2")
	assert.Equal(t, result.(url.Values)["c"], []string{"3", "4"})
}

func TestProcessParseURL(t *testing.T) {
	utils.Init()
	args := []interface{}{"http://www.google.com:8080/search?q=dotnet"}
	result, err := process.New("utils.url.ParseURL", args...).Exec()
	if err != nil {
		t.Errorf("ProcessParseURL error: %s", err)
	}

	assert.Equal(t, result.(map[string]interface{})["scheme"], "http")
	assert.Equal(t, result.(map[string]interface{})["host"], "www.google.com:8080")
	assert.Equal(t, result.(map[string]interface{})["domain"], "www.google.com")
	assert.Equal(t, result.(map[string]interface{})["path"], "/search")
	assert.Equal(t, result.(map[string]interface{})["port"], "8080")
	assert.Equal(t, result.(map[string]interface{})["query"].(url.Values).Get("q"), "dotnet")
	assert.Equal(t, result.(map[string]interface{})["url"], "http://www.google.com:8080/search?q=dotnet")
}

func TestProcessQueryParam(t *testing.T) {
	utils.Init()
	args := []interface{}{
		map[string]interface{}{
			"where.name.eq": "yao",
		},
	}
	result, err := process.New("utils.url.QueryParam", args...).Exec()
	if err != nil {
		t.Errorf("ProcessQueryParam error: %s", err)
	}

	assert.Len(t, result.(types.QueryParam).Wheres, 1)
}
