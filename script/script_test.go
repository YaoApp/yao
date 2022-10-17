package script

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func init() {
	rootLib := path.Join(os.Getenv("YAO_DEV"), "/tests/scripts")
	LoadFrom(rootLib, "")
}
func TestScript(t *testing.T) {
	res, err := gou.Yao.New("time", "hello").Call("world")
	assert.Nil(t, err)
	assert.Equal(t, "name:world", res)
}
