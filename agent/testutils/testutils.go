package testutils

import (
	"testing"

	_ "github.com/yaoapp/gou/encoding"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/gou"
	_ "github.com/yaoapp/gou/text"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/caller"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/test"

	// Import assistant to trigger init() which registers AgentGetterFunc
	_ "github.com/yaoapp/yao/agent/assistant"
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

	// Ensure JSAPI factories are registered (may be called multiple times, idempotent)
	// This is needed because Go's init() order is not guaranteed across packages
	caller.SetJSAPIFactory()
	llm.SetJSAPIFactory()

	// Register default query engine (required for DB search)
	// capsule.Global is initialized by test.Prepare
	if _, has := query.Engines["default"]; !has && capsule.Global != nil {
		query.Register("default", &gou.Query{
			Query: capsule.Query(),
			GetTableName: func(s string) string {
				if mod, has := model.Models[s]; has {
					return mod.MetaData.Table.Name
				}
				return s
			},
			AESKey: config.Conf.DB.AESKey,
		})
	}
}

// Clean clean the test environment
func Clean(t *testing.T) {
	test.Clean()
}
