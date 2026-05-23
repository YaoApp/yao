//go:build unit

package eval_test

import (
	"encoding/json"
	"testing"

	"github.com/yaoapp/yao/agent/eval"
)

func TestStartEvent_JSON(t *testing.T) {
	ev := eval.NewStartEvent("myagent", "gpt4o", &eval.TaiStatus{
		Bin: "/usr/local/bin/tai", HostExec: true, Docker: false,
	}, "127.0.0.1:9090", 5)

	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal StartEvent: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal StartEvent: %v", err)
	}
	if m["type"] != "start" {
		t.Errorf("type = %v, want start", m["type"])
	}
	if m["agent"] != "myagent" {
		t.Errorf("agent = %v, want myagent", m["agent"])
	}
	if int(m["total"].(float64)) != 5 {
		t.Errorf("total = %v, want 5", m["total"])
	}
}

func TestResultEvent_FromResult(t *testing.T) {
	r := &eval.Result{
		ID:         "T001",
		Status:     eval.StatusPassed,
		DurationMs: 1234,
		Input:      "hello",
		Output:     "world",
	}
	ev := eval.NewResultEvent(r)
	if ev.Type != "result" {
		t.Errorf("type = %s, want result", ev.Type)
	}
	if ev.ID != "T001" {
		t.Errorf("id = %s, want T001", ev.ID)
	}
	if ev.Output != "world" {
		t.Errorf("output = %s, want world", ev.Output)
	}
}

func TestSummaryEvent_FromReport(t *testing.T) {
	report := &eval.Report{
		Summary: &eval.Summary{
			Total:      3,
			Passed:     2,
			Failed:     1,
			Errors:     0,
			Timeouts:   0,
			DurationMs: 5000,
		},
		Results: []*eval.Result{
			{ID: "T001", Status: eval.StatusPassed},
			{ID: "T002", Status: eval.StatusFailed},
			{ID: "T003", Status: eval.StatusPassed},
		},
	}
	ev := eval.NewSummaryEvent(report, "Some assertions failed.")
	if ev.Type != "summary" {
		t.Errorf("type = %s, want summary", ev.Type)
	}
	if ev.Total != 3 {
		t.Errorf("total = %d, want 3", ev.Total)
	}
	if ev.Passed != 2 {
		t.Errorf("passed = %d, want 2", ev.Passed)
	}
	if ev.Failed != 1 {
		t.Errorf("failed = %d, want 1", ev.Failed)
	}
	if ev.ExitCode != 1 {
		t.Errorf("exit_code = %d, want 1", ev.ExitCode)
	}
	if len(ev.FailedIDs) != 1 || ev.FailedIDs[0] != "T002" {
		t.Errorf("failed_ids = %v, want [T002]", ev.FailedIDs)
	}
}

func TestSummaryEvent_AllPassed(t *testing.T) {
	report := &eval.Report{
		Summary: &eval.Summary{Total: 2, Passed: 2},
		Results: []*eval.Result{
			{ID: "T001", Status: eval.StatusPassed},
			{ID: "T002", Status: eval.StatusPassed},
		},
	}
	ev := eval.NewSummaryEvent(report, "All tests passed.")
	if ev.ExitCode != 0 {
		t.Errorf("exit_code = %d, want 0", ev.ExitCode)
	}
	if len(ev.FailedIDs) != 0 {
		t.Errorf("failed_ids = %v, want empty", ev.FailedIDs)
	}
}

func TestGenerateSuggestion(t *testing.T) {
	tests := []struct {
		name     string
		report   *eval.Report
		taiAvail bool
		want     string
	}{
		{
			name:     "all passed",
			report:   &eval.Report{Summary: &eval.Summary{Total: 1, Passed: 1}},
			taiAvail: true,
			want:     "All tests passed.",
		},
		{
			name:     "timeout",
			report:   &eval.Report{Summary: &eval.Summary{Total: 1, Timeouts: 1}},
			taiAvail: true,
			want:     "Some tests timed out. Consider increasing --timeout or checking agent responsiveness.",
		},
		{
			name:     "errors without tai",
			report:   &eval.Report{Summary: &eval.Summary{Total: 1, Errors: 1}},
			taiAvail: false,
			want:     "Runtime errors detected and Tai is not available. Install Tai or use --tai to specify its path.",
		},
		{
			name:     "errors with tai",
			report:   &eval.Report{Summary: &eval.Summary{Total: 1, Errors: 1}},
			taiAvail: true,
			want:     "Runtime errors detected. Re-run with --verbose or --json for full trace diagnostics.",
		},
		{
			name:     "assertions failed",
			report:   &eval.Report{Summary: &eval.Summary{Total: 2, Passed: 1, Failed: 1}},
			taiAvail: true,
			want:     "Some assertions failed. Re-run with --json to inspect trace details.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eval.GenerateSuggestion(tt.report, tt.taiAvail)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
