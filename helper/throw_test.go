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
