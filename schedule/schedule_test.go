package schedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/schedule"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/task"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	err := task.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	Load(config.Conf)
	check(t)
}

func TestStartStop(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	Load(config.Conf)
	Start()
	defer Stop()
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range schedule.Schedules {
		ids[id] = true
	}
	assert.True(t, ids["mail"])
	assert.True(t, ids["sendmail"])
}
