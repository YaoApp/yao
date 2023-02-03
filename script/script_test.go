package script

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range v8.Scripts {
		ids[id] = true
	}
	assert.True(t, ids["tests.task.mail"])
	assert.True(t, ids["tests.api"])
	assert.True(t, ids["runtime.basic"])
	assert.True(t, ids["runtime.bridge"])
	assert.True(t, ids["__yao_service.foo"])
}
