//go:build unit

package eval_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yaoapp/yao/agent/eval"
)

type bufferEventWriter struct {
	buf bytes.Buffer
}

func (w *bufferEventWriter) WriteEvent(data []byte) error {
	w.buf.Write(data)
	w.buf.WriteByte('\n')
	return nil
}

func TestOutputWriter_JSONMode_EventSequence(t *testing.T) {
	ew := &bufferEventWriter{}
	out := eval.NewOutputWriterWithWriter(false, &bytes.Buffer{}, ew)

	out.Info("starting")
	out.Error("something failed")
	out.Header("Header")

	lines := strings.Split(strings.TrimSpace(ew.buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 events, got %d: %v", len(lines), lines)
	}

	var ev0 map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &ev0); err != nil {
		t.Fatalf("unmarshal line 0: %v", err)
	}
	if ev0["type"] != "log" || ev0["level"] != "info" {
		t.Errorf("line 0: type=%v level=%v", ev0["type"], ev0["level"])
	}

	var ev1 map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &ev1); err != nil {
		t.Fatalf("unmarshal line 1: %v", err)
	}
	if ev1["type"] != "log" || ev1["level"] != "error" {
		t.Errorf("line 1: type=%v level=%v", ev1["type"], ev1["level"])
	}
}

func TestOutputWriter_TerminalMode_ContainsSymbols(t *testing.T) {
	var buf bytes.Buffer
	out := eval.NewOutputWriterWithWriter(false, &buf, nil)

	out.Info("hello")
	out.Error("oops")
	out.Warning("watch out")

	output := buf.String()
	if !strings.Contains(output, "ℹ") {
		t.Error("missing info symbol ℹ")
	}
	if !strings.Contains(output, "✗") {
		t.Error("missing error symbol ✗")
	}
	if !strings.Contains(output, "⚠") {
		t.Error("missing warning symbol ⚠")
	}
}

func TestOutputWriter_DryRunCase(t *testing.T) {
	var buf bytes.Buffer
	out := eval.NewOutputWriterWithWriter(false, &buf, nil)

	out.DryRunCase("T001", "what is the weather")
	output := buf.String()
	if !strings.Contains(output, "T001") {
		t.Error("missing test ID in dry-run output")
	}
	if !strings.Contains(output, "what is the weather") {
		t.Error("missing input in dry-run output")
	}
}

func TestOutputWriter_EmitTypedEvent(t *testing.T) {
	ew := &bufferEventWriter{}
	out := eval.NewOutputWriterWithWriter(false, &bytes.Buffer{}, ew)

	ev := eval.NewStartEvent("myagent", "gpt4o", nil, "", 1)
	ok := out.EmitTypedEvent(ev)
	if !ok {
		t.Error("EmitTypedEvent returned false")
	}

	var m map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(ew.buf.Bytes()), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["type"] != "start" {
		t.Errorf("type = %v, want start", m["type"])
	}
}
