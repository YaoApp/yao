//go:build unit

package agent_test

import (
	"testing"

	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestToolConfigIsPasswordField(t *testing.T) {
	testprepare.PrepareUnit(t)

	tests := []struct {
		key  string
		want bool
	}{
		{"api_key", true},
		{"zone", false},
		{"api_url", false},
		{"", false},
		{"API_KEY", false},
	}
	for _, tt := range tests {
		if got := agent.ExportIsPasswordField(tt.key); got != tt.want {
			t.Errorf("isPasswordField(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestToolConfigValidationMaps(t *testing.T) {
	testprepare.PrepareUnit(t)

	wsDefaults := agent.ExportValidWebSearchDefaults
	for _, key := range []string{"tavily", "serper", "cloud"} {
		if !wsDefaults[key] {
			t.Errorf("validWebSearchDefaults missing %q", key)
		}
	}
	if wsDefaults["brightdata"] {
		t.Error("validWebSearchDefaults should not contain 'brightdata'")
	}

	wfDefaults := agent.ExportValidWebFetchDefaults
	for _, key := range []string{"brightdata", "cloud", "direct"} {
		if !wfDefaults[key] {
			t.Errorf("validWebFetchDefaults missing %q", key)
		}
	}
	if wfDefaults["tavily"] {
		t.Error("validWebFetchDefaults should not contain 'tavily'")
	}

	providerKeys := agent.ExportValidProviderKeys
	for _, key := range []string{"tavily", "serper", "brightdata"} {
		if !providerKeys[key] {
			t.Errorf("validProviderKeys missing %q", key)
		}
	}
	if providerKeys["cloud"] {
		t.Error("validProviderKeys should not contain 'cloud'")
	}
}
