package openai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestProcessTiktoken(t *testing.T) {
	// Hash
	args := []interface{}{"gpt-3.5-turbo", "hello world"}
	res := process.New("yao.openai.Tiktoken", args...).Run()
	assert.Equal(t, 2, res)

	args = []interface{}{"gpt-3.5-turbo", "你好世界！"}
	res = process.New("yao.openai.Tiktoken", args...).Run()
	assert.Equal(t, 6, res)
}
