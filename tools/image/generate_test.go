package image

import (
	"testing"

	"github.com/yaoapp/gou/process"
)

func TestGenerateHandler_NoPrompt(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{""},
	}
	result := GenerateHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if errMsg, _ := m["error"].(string); errMsg != "prompt is required" {
		t.Errorf("expected 'prompt is required', got %q", errMsg)
	}
}

func TestGenerateHandler_NoAuth(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{"A sunset", "", "1024x1024"},
	}
	result := GenerateHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if _, hasErr := m["error"]; !hasErr {
		t.Error("expected error when no auth info")
	}
}
