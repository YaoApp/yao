package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestProcessHexToString(t *testing.T) {
	res, err := process.New("xiang.helper.HexToString", []byte{0x0, 0x1}).Exec()
	assert.Nil(t, err)
	assert.Equal(t, "0001", res)

	res, err = process.New("xiang.helper.HexToString", string([]byte{0x0, 0x1})).Exec()
	assert.Nil(t, err)
	assert.Equal(t, "0001", res)

	res, err = process.New("xiang.helper.HexToString", 1024).Exec()
	assert.Nil(t, err)
	assert.Nil(t, res)
}
