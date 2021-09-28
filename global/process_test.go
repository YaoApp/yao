package global

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestProcessPing(t *testing.T) {
	process := gou.NewProcess("xiang.global.ping")
	res, ok := processPing(process).(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, res["version"], VERSION)
}
