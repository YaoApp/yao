package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestProcessNow(t *testing.T) {
	assert.LessOrEqual(t, time.Now().Unix(), gou.NewProcess("utils.now.Timestamp").Run().(int64))
	assert.LessOrEqual(t, time.Now().UnixMilli(), gou.NewProcess("utils.now.Timestampms").Run().(int64))
	assert.NotNil(t, gou.NewProcess("utils.now.Date").Run())
	assert.NotNil(t, gou.NewProcess("utils.now.Time").Run())
	assert.NotNil(t, gou.NewProcess("utils.now.DateTime").Run())
}
