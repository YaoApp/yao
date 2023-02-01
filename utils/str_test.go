package utils

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestProcessStrJoin(t *testing.T) {
	res := process.New("utils.str.Join", []interface{}{"FOO", 20, "BAR"}, ",").Run().(string)
	assert.Equal(t, "FOO,20,BAR", res)
}

func TestProcessStrJoinPath(t *testing.T) {
	res := process.New("utils.str.JoinPath", "data", 20, "app").Run().(string)
	shouldBe := fmt.Sprintf("data%s20%sapp", string(os.PathSeparator), string(os.PathSeparator))
	assert.Equal(t, shouldBe, res)
}
