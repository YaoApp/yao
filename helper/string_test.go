package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestProcessStrConcat(t *testing.T) {
	res := process.New("xiang.helper.StrConcat", "FOO", 20, "BAR").Run().(string)
	assert.Equal(t, "FOO20BAR", res)
}
