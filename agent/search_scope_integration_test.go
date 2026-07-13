//go:build integration

package agent_test

import (
	"testing"

	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestSearchScopeIsolation(t *testing.T) {
	testprepare.PrepareSandbox(t)

	systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
	teamScope := setting.ScopeID{Scope: setting.ScopeTeam, TeamID: "isolation-test-team"}

	setting.Global.Set(systemScope, "search.providers.serper", map[string]interface{}{
		"status":  "connected",
		"enabled": true,
		"field_values": map[string]interface{}{
			"api_key": "encrypted-system-key",
		},
	})
	setting.Global.Set(systemScope, "search.tool_assignment", map[string]interface{}{
		"web_search": "serper",
	})
	t.Cleanup(func() {
		setting.Global.Delete(systemScope, "search.providers.serper")
		setting.Global.Delete(systemScope, "search.tool_assignment")
		setting.Global.Delete(teamScope, "search.providers.serper")
		setting.Global.Delete(teamScope, "search.tool_assignment")
	})

	t.Run("get_team_scope_excludes_system", func(t *testing.T) {
		data, err := setting.Global.Get(teamScope, "search.providers.serper")
		if err == nil && data != nil {
			if status, _ := data["status"].(string); status == "connected" {
				t.Error("Get(teamScope) returned system-scope provider data; expected nil or empty")
			}
		}

		assignment, err := setting.Global.Get(teamScope, "search.tool_assignment")
		if err == nil && assignment != nil {
			if v, _ := assignment["web_search"].(string); v == "serper" {
				t.Error("Get(teamScope) returned system-scope tool_assignment; expected nil or empty")
			}
		}
	})

	t.Run("get_merged_includes_system", func(t *testing.T) {
		data, err := setting.Global.GetMerged("", "isolation-test-team", "search.providers.serper")
		if err != nil {
			t.Fatalf("GetMerged: %v", err)
		}
		if data == nil {
			t.Fatal("GetMerged returned nil; expected system-scope fallback data")
		}
		if status, _ := data["status"].(string); status != "connected" {
			t.Errorf("GetMerged status = %q, want 'connected'", status)
		}

		assignment, err := setting.Global.GetMerged("", "isolation-test-team", "search.tool_assignment")
		if err != nil {
			t.Fatalf("GetMerged: %v", err)
		}
		if v, _ := assignment["web_search"].(string); v != "serper" {
			t.Errorf("GetMerged web_search = %q, want 'serper'", v)
		}
	})

	t.Run("team_override_visible_in_get", func(t *testing.T) {
		setting.Global.Set(teamScope, "search.providers.serper", map[string]interface{}{
			"status":  "connected",
			"enabled": true,
			"field_values": map[string]interface{}{
				"api_key": "encrypted-team-key",
			},
		})

		data, err := setting.Global.Get(teamScope, "search.providers.serper")
		if err != nil {
			t.Fatalf("Get(teamScope): %v", err)
		}
		if data == nil {
			t.Fatal("Get(teamScope) returned nil after team-level Set")
		}
		fv, _ := data["field_values"].(map[string]interface{})
		if key, _ := fv["api_key"].(string); key != "encrypted-team-key" {
			t.Errorf("api_key = %q, want 'encrypted-team-key'", key)
		}
	})
}
