package utils

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	_ "github.com/yaoapp/yao/helper"
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

func TestProcessStrHex(t *testing.T) {
	res, err := process.New("utils.str.Hex", []byte{0x0, 0x1}).Exec()
	assert.Nil(t, err)
	assert.Equal(t, "0001", res)

	res, err = process.New("utils.str.Hex", string([]byte{0x0, 0x1})).Exec()
	assert.Nil(t, err)
	assert.Equal(t, "0001", res)

	res, err = process.New("utils.str.Hex", 1024).Exec()
	assert.Nil(t, err)
	assert.Nil(t, res)
}
