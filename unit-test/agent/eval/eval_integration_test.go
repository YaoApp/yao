//go:build integration

package eval_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/eval"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestEval_SingleMessage_E2E(t *testing.T) {
	testprepare.PrepareSandbox(t)

	var buf bytes.Buffer
	opts := &eval.Options{
		Input:     "hello",
		InputMode: eval.InputModeMessage,
		AgentID:   "tests.echo",
		UserID:    "test-user",
		TeamID:    "test-team",
		Timeout:   30 * time.Second,
		Parallel:  1,
		Writer:    &buf,
	}

	runner := eval.NewRunner(opts)
	report, err := runner.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report == nil {
		t.Fatal("report is nil")
	}
	if report.Summary == nil {
		t.Fatal("summary is nil")
	}
	if report.Summary.Total != 1 {
		t.Errorf("total = %d, want 1", report.Summary.Total)
	}
}

func TestEval_JSONOutput_NDJSON(t *testing.T) {
	testprepare.PrepareSandbox(t)

	var termBuf bytes.Buffer
	ew := &testEventWriter{}

	opts := &eval.Options{
		Input:       "hello",
		InputMode:   eval.InputModeMessage,
		AgentID:     "tests.echo",
		UserID:      "test-user",
		TeamID:      "test-team",
		Timeout:     30 * time.Second,
		Parallel:    1,
		JSONOutput:  true,
		Writer:      &termBuf,
		EventWriter: ew,
	}

	runner := eval.NewRunner(opts)
	_, err := runner.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	events := ew.events()
	if len(events) == 0 {
		t.Fatal("expected at least one NDJSON event")
	}

	for _, line := range events {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("invalid NDJSON: %v — line: %s", err, line)
		}
		if _, ok := m["type"]; !ok {
			t.Errorf("event missing 'type' field: %s", line)
		}
	}
}

func TestEval_DryRun(t *testing.T) {
	testprepare.PrepareSandbox(t)

	var buf bytes.Buffer
	opts := &eval.Options{
		Input:     "hello",
		InputMode: eval.InputModeMessage,
		AgentID:   "tests.echo",
		UserID:    "test-user",
		TeamID:    "test-team",
		Timeout:   30 * time.Second,
		DryRun:    true,
		Writer:    &buf,
	}

	runner := eval.NewRunner(opts)
	report, err := runner.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report == nil {
		t.Fatal("report is nil in dry-run")
	}
}

type testEventWriter struct {
	buf bytes.Buffer
}

func (w *testEventWriter) WriteEvent(data []byte) error {
	w.buf.Write(data)
	w.buf.WriteByte('\n')
	return nil
}

func (w *testEventWriter) events() []string {
	raw := strings.TrimSpace(w.buf.String())
	if raw == "" {
		return nil
	}
	return strings.Split(raw, "\n")
}
