package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/share"
)

func TestProcessPing(t *testing.T) {
	process := process.New("xiang.main.ping")
	res, ok := processPing(process).(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, res["version"], share.VERSION)
}

func TestProcessAliasPing(t *testing.T) {
	res, ok := process.New("xiang.sys.Ping").Run().(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, res["version"], share.VERSION)
}

func TestProcessInspect(t *testing.T) {
	res, ok := process.New("xiang.sys.Inspect").Run().(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, res["VERSION"])
}
