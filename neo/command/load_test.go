package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/command/driver"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	Commands = map[string]*Command{}
	Load(config.Conf)
	check(t)
}

func TestLoadWithStore(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	Commands = map[string]*Command{}
	mem, err := driver.NewMemory("gpt-3_5-turbo", nil)
	if err != nil {
		t.Fatal(err)
	}

	SetStore(mem)
	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range Commands {
		ids[id] = true
	}
	assert.True(t, ids["table.data"])
	assert.GreaterOrEqual(t, len(Autopilots), 1)
}
