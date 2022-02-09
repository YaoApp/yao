package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/share"
)

func TestProcessPing(t *testing.T) {
	process := gou.NewProcess("xiang.main.ping")
	res, ok := processPing(process).(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, res["version"], share.VERSION)
}
