package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/task"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	Load(config.Conf)
	LoadFrom("not a path", "404.")
	check(t)
}

func check(t *testing.T) {
	assert.Equal(t, 1, len(task.Tasks))
}
