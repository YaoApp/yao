package schedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/task"
)

func TestLoad(t *testing.T) {
	task.Load(config.Conf)
	Load(config.Conf)
	LoadFrom("not a path", "404.")
	check(t)
}

func check(t *testing.T) {
	assert.Equal(t, 2, len(gou.Schedules))
}
