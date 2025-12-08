package testutils

import (
	"testing"

	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/test"
)

// Prepare prepare the test environment with optional V8 mode configuration
// Usage:
//
//	testutils.Prepare(t)                                              // standard mode (default)
//	testutils.Prepare(t, test.PrepareOption{V8Mode: "performance"})  // performance mode for benchmarks
func Prepare(t *testing.T, opts ...interface{}) {
	test.Prepare(t, config.Conf, opts...)

	// Load KB (required for agent KB features)
	_, err := kb.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// Load agent
	err = agent.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}

// Clean clean the test environment
func Clean(t *testing.T) {
	test.Clean()
}
