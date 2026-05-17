package testutils

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

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

	_ "github.com/yaoapp/yao/agent/assistant"
)

// Prepare prepare the test environment with optional V8 mode configuration.
// Uses YAO_TEST_APPLICATION for the app root.
//
//	testutils.Prepare(t)
//	testutils.Prepare(t, test.PrepareOption{V8Mode: "performance"})
func Prepare(t *testing.T, opts ...interface{}) {
	test.Prepare(t, config.Conf, opts...)
	loadAgentStack(t)
}

// PrepareAgent prepares the test environment for agent-specific tests.
// It reads YAO_AGENT_TEST_APPLICATION first, falling back to YAO_TEST_APPLICATION.
// Tests calling this should t.Skip when the env var is absent (Tier 2+).
func PrepareAgent(t *testing.T, opts ...interface{}) {
	t.Helper()
	appRoot := os.Getenv("YAO_AGENT_TEST_APPLICATION")
	if appRoot == "" {
		appRoot = os.Getenv("YAO_TEST_APPLICATION")
	}
	if appRoot == "" {
		t.Skip("neither YAO_AGENT_TEST_APPLICATION nor YAO_TEST_APPLICATION is set")
	}
	os.Setenv("YAO_AGENT_TEST_APPLICATION", appRoot)
	test.Prepare(t, config.Conf, "YAO_AGENT_TEST_APPLICATION")
	loadAgentStack(t)
}

// PrepareAgentMinimal loads only DB + system models + stores.
// Skips scripts, connectors, messenger, and V8 runtime for fast Tier-1 tests.
func PrepareAgentMinimal(t *testing.T) {
	t.Helper()
	appRoot := os.Getenv("YAO_AGENT_TEST_APPLICATION")
	if appRoot == "" {
		appRoot = os.Getenv("YAO_TEST_APPLICATION")
	}
	if appRoot == "" {
		t.Skip("neither YAO_AGENT_TEST_APPLICATION nor YAO_TEST_APPLICATION is set")
	}

	// PrepareAgentMinimal intentionally uses the lightweight test.Prepare path.
	// Full connector/script loading is unnecessary for pure unit tests.
	os.Setenv("YAO_AGENT_TEST_APPLICATION", appRoot)
	test.Prepare(t, config.Conf, "YAO_AGENT_TEST_APPLICATION")
}

// SkipWithoutMockLLM skips the test when mock-llm is unreachable (Tier 3 guard).
func SkipWithoutMockLLM(t *testing.T) {
	t.Helper()
	host := os.Getenv("MOCK_LLM_HOST")
	if host == "" {
		host = "http://mock-llm:9999"
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(host + "/healthz")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Skipf("mock-llm not reachable at %s/healthz: %v", host, err)
	}
}

// SkipWithoutTai skips the test when the specified Tai gRPC address is unreachable (Tier 4 guard).
func SkipWithoutTai(t *testing.T, envVar string) {
	t.Helper()
	addr := os.Getenv(envVar)
	if addr == "" {
		t.Skipf("Tai address not set: $%s", envVar)
	}
}

// MockLLMHost returns the mock-llm base URL from MOCK_LLM_HOST or the default.
func MockLLMHost() string {
	if h := os.Getenv("MOCK_LLM_HOST"); h != "" {
		return h
	}
	return "http://mock-llm:9999"
}

// RequireE2EKeys ensures at least one real LLM API key is available (Tier 5 guard).
func RequireE2EKeys(t *testing.T) {
	t.Helper()
	keys := []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "DEEPSEEK_V4_API_KEY"}
	for _, k := range keys {
		if os.Getenv(k) != "" {
			return
		}
	}
	t.Skipf("no real LLM API key found (checked: %v)", keys)
}

func loadAgentStack(t *testing.T) {
	t.Helper()

	if _, err := kb.Load(config.Conf); err != nil {
		t.Fatal(fmt.Errorf("load KB: %w", err))
	}
	if err := agent.Load(config.Conf); err != nil {
		t.Fatal(fmt.Errorf("load agent: %w", err))
	}

	caller.SetJSAPIFactory()
	llm.SetJSAPIFactory()

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
