package script

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/runtime"
)

func init() {
	runtime.Load(config.Conf)
	rootLib := path.Join(os.Getenv("YAO_DEV"), "/tests/scripts")
	LoadFrom(rootLib, "")
}
func TestScript(t *testing.T) {
	res, err := gou.Yao.New("time", "hello").Call("world")
	assert.Nil(t, err)
	assert.Equal(t, "name:world", res)
}
