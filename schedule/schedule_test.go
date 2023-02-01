package schedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/schedule"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/task"
)

func TestLoad(t *testing.T) {
	task.Load(config.Conf)
	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	assert.Equal(t, 2, len(schedule.Schedules))
}
