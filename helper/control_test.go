package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func TestProcessThrow(t *testing.T) {
	e := exception.New("Someting error", 500)
	assert.PanicsWithValue(t, *e, func() {
		process.New("xiang.helper.Throw", "Someting error", 500).Run()
	})
}

func TestProcessReturn(t *testing.T) {
	v := process.New("xiang.helper.Return", "hello", "world").Run().([]interface{})
	assert.Equal(t, "hello", v[0])
	assert.Equal(t, "world", v[1])
}
