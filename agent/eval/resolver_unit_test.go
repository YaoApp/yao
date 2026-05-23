//go:build unit

package eval_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaoapp/yao/agent/eval"
)

func TestDetectInputMode_Scripts(t *testing.T) {
	got := eval.DetectInputMode("scripts.myagent.tools")
	if got != eval.InputModeScript {
		t.Errorf("DetectInputMode(scripts.myagent.tools) = %s, want script", got)
	}
}

func TestDetectInputMode_JSONL(t *testing.T) {
	got := eval.DetectInputMode("tests/inputs.jsonl")
	if got != eval.InputModeFile {
		t.Errorf("DetectInputMode(tests/inputs.jsonl) = %s, want file", got)
	}
}

func TestDetectInputMode_Message(t *testing.T) {
	got := eval.DetectInputMode("what is the weather")
	if got != eval.InputModeMessage {
		t.Errorf("DetectInputMode(message) = %s, want message", got)
	}
}

func TestValidateOptions_RequiresInput(t *testing.T) {
	err := eval.ValidateOptions(&eval.Options{})
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestValidateOptions_NegativeTimeout(t *testing.T) {
	err := eval.ValidateOptions(&eval.Options{
		Input:   "hello",
		Timeout: -1,
	})
	if err == nil {
		t.Error("expected error for negative timeout")
	}
}

func TestValidateOptions_Valid(t *testing.T) {
	err := eval.ValidateOptions(&eval.Options{
		Input:     "hello",
		InputMode: eval.InputModeMessage,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetOutputFormat(t *testing.T) {
	tests := []struct {
		path string
		want eval.OutputFormat
	}{
		{"report.json", eval.FormatJSON},
		{"report.html", eval.FormatHTML},
		{"report.md", eval.FormatMarkdown},
		{"report.txt", eval.FormatJSON},
	}
	for _, tt := range tests {
		got := eval.GetOutputFormat(tt.path)
		if got != tt.want {
			t.Errorf("GetOutputFormat(%s) = %s, want %s", tt.path, got, tt.want)
		}
	}
}

func TestCreateTestCaseFromMessage(t *testing.T) {
	tc := eval.CreateTestCaseFromMessage("hello world")
	if tc.ID != "T001" {
		t.Errorf("ID = %s, want T001", tc.ID)
	}
	if tc.Input != "hello world" {
		t.Errorf("Input = %v, want hello world", tc.Input)
	}
}

func TestResolvePathWithYaoRoot_Absolute(t *testing.T) {
	got := eval.ResolvePathWithYaoRoot("/absolute/path/file.jsonl")
	if got != "/absolute/path/file.jsonl" {
		t.Errorf("absolute path changed: %s", got)
	}
}

func TestResolvePathWithYaoRoot_Relative(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.jsonl")
	os.WriteFile(tmpFile, []byte("{}"), 0o644)

	got := eval.ResolvePathWithYaoRoot(tmpFile)
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %s", got)
	}
}
