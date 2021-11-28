package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
)

func TestRequestGetHtml(t *testing.T) {
	resp := RequestGet("https://api.github.com/zen", nil, nil)
	assert.Equal(t, 200, resp.Status)
	assert.Equal(t, nil, resp.Data)
}

func TestRequestGetJSON(t *testing.T) {
	resp := RequestGet("https://api.github.com/users/yaoapp", nil, nil)
	assert.Equal(t, 200, resp.Status)
	data := any.Of(resp.Data).MapStr().Dot()
	assert.Equal(t, "https://github.com/YaoApp", data.Get("html_url"))
}

func TestRequestPost(t *testing.T) {
	resp := RequestPost("https://api.github.com/zen", map[string]interface{}{"foo": "bar"}, nil)
	assert.Equal(t, 404, resp.Status)
}

func TestRequestProcessGet(t *testing.T) {
	args := []interface{}{"https://api.github.com/zen"}
	res := gou.NewProcess("xiang.helper.Get", args...).Run()
	resp, ok := res.(Response)
	assert.True(t, ok)
	assert.Equal(t, 200, resp.Status)
	assert.Equal(t, nil, resp.Data)
}

func TestRequestProcessPost(t *testing.T) {
	args := []interface{}{"https://api.github.com/zen", map[string]interface{}{"foo": "bar"}, nil}
	res := gou.NewProcess("xiang.helper.Post", args...).Run()
	resp, ok := res.(Response)
	assert.True(t, ok)
	assert.Equal(t, 404, resp.Status)
}

func TestRequestProcessSend(t *testing.T) {
	args := []interface{}{"GET", "https://api.github.com/zen"}
	res := gou.NewProcess("xiang.helper.Send", args...).Run()
	resp, ok := res.(Response)
	assert.True(t, ok)
	assert.Equal(t, 200, resp.Status)
	assert.Equal(t, nil, resp.Data)
}
