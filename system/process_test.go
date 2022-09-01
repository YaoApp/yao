package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestProcessReturn(t *testing.T) {
	output, err := gou.NewProcess("yao.system.Exec", "echo", "hello").Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "hello", output)
}
