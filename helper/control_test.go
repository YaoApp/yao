package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

func TestProcessThrow(t *testing.T) {
	e := exception.New("Someting error", 500)
	assert.PanicsWithValue(t, *e, func() {
		gou.NewProcess("xiang.helper.Throw", "Someting error", 500).Run()
	})
}

func TestProcessReturn(t *testing.T) {
	v := gou.NewProcess("xiang.helper.Return", "hello", "world").Run().([]interface{})
	assert.Equal(t, "hello", v[0])
	assert.Equal(t, "world", v[1])
}
