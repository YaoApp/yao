package aigc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestProcessAigcs(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)

	args := []interface{}{"你好"}
	res := process.New("aigcs.translate", args...).Run()
	assert.Contains(t, res, "Hello")
}
