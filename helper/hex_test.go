package helper

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/flow"
	"github.com/yaoapp/yao/share"
)

func TestProcessHexToString(t *testing.T) {
	res, err := gou.NewProcess("xiang.helper.HexToString", []byte{0x0, 0x1}).Exec()
	assert.Nil(t, err)
	assert.Equal(t, "0001", res)

	res, err = gou.NewProcess("xiang.helper.HexToString", string([]byte{0x0, 0x1})).Exec()
	assert.Nil(t, err)
	assert.Equal(t, "0001", res)

	res, err = gou.NewProcess("xiang.helper.HexToString", 1024).Exec()
	assert.Nil(t, err)
	assert.Nil(t, res)
}

func TestProcessHexToStringInFlow(t *testing.T) {
	testflow := path.Join(os.Getenv("YAO_DEV"), "tests", "flows", "helper")
	flow.LoadFrom(testflow, "helper.")

	res, err := gou.NewProcess("flows.helper.HexToString").Exec()
	assert.Nil(t, err)
	assert.Equal(t, "6162", res) // ab
}

func TestProcessHexToStringInScript(t *testing.T) {
	testscirpt := path.Join(os.Getenv("YAO_DEV"), "tests", "scripts")
	share.LoadFrom(testscirpt)

	res, err := gou.NewProcess("scripts.helper.HexToStringString").Exec()
	assert.Nil(t, err)
	assert.Equal(t, "6162", res) // ab

	res, err = gou.NewProcess("scripts.helper.HexToStringBytes", []byte{0x0, 0x1}).Exec()
	assert.Nil(t, err)
	assert.Equal(t, "0001", res) // []byte{0x0, 0x1}
}

func TestProcessBufferInScript(t *testing.T) {
	testscirpt := path.Join(os.Getenv("YAO_DEV"), "tests", "scripts")
	share.LoadFrom(testscirpt)

	res, err := gou.NewProcess("scripts.helper.Buffer").Exec()
	assert.Nil(t, err)

	str, ok := res.(string)
	assert.True(t, ok)
	assert.Equal(t, []byte{0x0, 0x1}, []byte(str))

}
