package aigc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestCall(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)

	aigc, err := Select("translate")
	if err != nil {
		t.Fatal(err)
	}

	content, ex := aigc.Call("你好哇", "", nil)
	if ex != nil {
		t.Fatal(ex.Message)
	}
	assert.Contains(t, content, "Hello")
}

func TestCallWithProcess(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)

	aigc, err := Select("draw")
	if err != nil {
		t.Fatal(err)
	}

	args, ex := aigc.Call("帮我画一只小白兔，要有白色的耳朵. 画布高度 256，宽度 256", "", nil)
	if ex != nil {
		t.Fatal(ex.Message)
	}

	data, ok := args.(map[string]interface{})
	if !ok {
		t.Fatal("args is not map[string]interface{}")
	}

	assert.Equal(t, float64(256), data["height"])
	assert.Equal(t, float64(256), data["width"])
}

func prepare(t *testing.T) {
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}
