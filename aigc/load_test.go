package aigc

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	for id := range AIGCs {
		ids[id] = true
	}

	assert.True(t, ids["translate"])
	assert.True(t, ids["draw"])
	assert.GreaterOrEqual(t, len(Autopilots), 2)
}
