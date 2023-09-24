package sui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	prepare(t)
	defer clean()

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	check(t)
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range SUIs {
		ids[id] = true
	}
	assert.True(t, ids["azure"])
	assert.True(t, ids["demo"])
	assert.True(t, ids["screen"])
}

func prepare(t *testing.T) {
	test.Prepare(t, config.Conf, "YAO_TEST_BUILDER_APPLICATION")
}

func clean() {
	test.Clean()
}
