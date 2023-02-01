package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestProcessNow(t *testing.T) {
	assert.LessOrEqual(t, time.Now().Unix(), process.New("utils.now.Timestamp").Run().(int64))
	assert.LessOrEqual(t, time.Now().UnixMilli(), process.New("utils.now.Timestampms").Run().(int64))
	assert.NotNil(t, process.New("utils.now.Date").Run())
	assert.NotNil(t, process.New("utils.now.Time").Run())
	assert.NotNil(t, process.New("utils.now.DateTime").Run())
}
