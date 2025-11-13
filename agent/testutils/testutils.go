package testutils

import (
	"testing"

	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// Prepare prepare the test environment
func Prepare(t *testing.T) {
	test.Prepare(t, config.Conf)

	// Load agent
	err := agent.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}

// Clean clean the test environment
func Clean(t *testing.T) {
	test.Clean()
}
