package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/sui/core"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	prepare(t)
	defer clean()
	check(t)
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range core.SUIs {
		ids[id] = true
	}
	assert.False(t, ids["not-exist"])
	assert.True(t, ids["test"])
	assert.True(t, ids["web"])
}

func prepare(t *testing.T) {
	test.Prepare(t, config.Conf, "YAO_SUI_TEST_APPLICATION")
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	advanced, err := core.SUIs["test"].GetTemplate("advanced")
	if err != nil {
		t.Fatal(err)
	}

	warnings, err := advanced.Build(&core.BuildOption{SSR: true, AssetRoot: "/unit-test/assets"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, warnings, 0)
}

func clean() {
	test.Clean()
}
