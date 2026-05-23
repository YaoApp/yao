//go:build unit

package eval_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/eval"
)

func TestOptions_JSONRoundTrip(t *testing.T) {
	opts := eval.Options{
		Input:     "hello",
		InputMode: eval.InputModeMessage,
		AgentID:   "myagent",
		Connector: "gpt4o",
		UserID:    "user-001",
		TeamID:    "team-alpha",
		Timeout:   5 * time.Minute,
		Parallel:  2,
		Runs:      3,
		Run:       "TestSystem.*",
		BeforeAll: "scripts:tests.env.BeforeAll",
		AfterAll:  "scripts:tests.env.AfterAll",
		DryRun:    true,
		Scripts:   "tools",
		Remote:    true,
		TaiBin:    "/usr/local/bin/tai",
		AuthFile:  "~/.yao/credentials",
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded eval.Options
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.AgentID != "myagent" {
		t.Errorf("AgentID = %s, want myagent", decoded.AgentID)
	}
	if decoded.Run != "TestSystem.*" {
		t.Errorf("Run = %s, want TestSystem.*", decoded.Run)
	}
	if decoded.BeforeAll != "scripts:tests.env.BeforeAll" {
		t.Errorf("BeforeAll = %s", decoded.BeforeAll)
	}
	if decoded.InputMode != eval.InputModeMessage {
		t.Errorf("InputMode = %s, want message", decoded.InputMode)
	}
	if decoded.Scripts != "tools" {
		t.Errorf("Scripts = %s, want tools", decoded.Scripts)
	}
	if !decoded.Remote {
		t.Error("Remote = false, want true")
	}
}

func TestCase_GetEnvironment_Priority(t *testing.T) {
	tc := eval.Case{
		UserID: "case-user",
		TeamID: "case-team",
	}
	opts := &eval.Options{
		UserID: "cli-user",
	}
	env := tc.GetEnvironment(opts)

	if env.UserID != "cli-user" {
		t.Errorf("UserID = %s, want cli-user (CLI overrides case)", env.UserID)
	}
	if env.TeamID != "case-team" {
		t.Errorf("TeamID = %s, want case-team (case value used)", env.TeamID)
	}
}

func TestCase_GetEnvironment_Defaults(t *testing.T) {
	tc := eval.Case{}
	env := tc.GetEnvironment(nil)
	if env.UserID != "test-user" {
		t.Errorf("UserID = %s, want test-user", env.UserID)
	}
	if env.TeamID != "test-team" {
		t.Errorf("TeamID = %s, want test-team", env.TeamID)
	}
	if env.Locale != "en-us" {
		t.Errorf("Locale = %s, want en-us", env.Locale)
	}
}

func TestCase_GetTimeout(t *testing.T) {
	tc := eval.Case{Timeout: "30s"}
	d := tc.GetTimeout(5 * time.Minute)
	if d != 30*time.Second {
		t.Errorf("GetTimeout = %v, want 30s", d)
	}

	tc2 := eval.Case{}
	d2 := tc2.GetTimeout(5 * time.Minute)
	if d2 != 5*time.Minute {
		t.Errorf("GetTimeout default = %v, want 5m", d2)
	}
}

func TestReport_HasFailures(t *testing.T) {
	r := &eval.Report{Summary: &eval.Summary{Total: 2, Passed: 2}}
	if r.HasFailures() {
		t.Error("expected no failures")
	}

	r2 := &eval.Report{Summary: &eval.Summary{Total: 2, Passed: 1, Failed: 1}}
	if !r2.HasFailures() {
		t.Error("expected failures")
	}
}

func TestReport_PassRate(t *testing.T) {
	r := &eval.Report{Summary: &eval.Summary{Total: 4, Passed: 3}}
	rate := r.PassRate()
	if rate != 75.0 {
		t.Errorf("PassRate = %f, want 75.0", rate)
	}
}

func TestClassifyStability(t *testing.T) {
	tests := []struct {
		rate float64
		want eval.StabilityClass
	}{
		{100, eval.StabilityStable},
		{90, eval.StabilityMostlyStable},
		{60, eval.StabilityUnstable},
		{30, eval.StabilityHighlyUnstable},
	}
	for _, tt := range tests {
		got := eval.ClassifyStability(tt.rate)
		if got != tt.want {
			t.Errorf("ClassifyStability(%.0f) = %s, want %s", tt.rate, got, tt.want)
		}
	}
}
