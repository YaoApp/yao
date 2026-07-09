package image

import (
	"testing"

	"github.com/yaoapp/gou/process"
)

func TestEditHandler_NoImage(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{""},
	}
	result := EditHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if errMsg, _ := m["error"].(string); errMsg == "" {
		t.Error("expected error when image is empty")
	}
}

func TestEditHandler_NoPrompt(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{"https://example.com/photo.png", ""},
	}
	result := EditHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if errMsg, _ := m["error"].(string); errMsg != "prompt is required" {
		t.Errorf("expected 'prompt is required', got %q", errMsg)
	}
}

func TestEditHandler_NoAuth(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{"https://example.com/photo.png", "make it blue", "", "1024x1024"},
	}
	result := EditHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if _, hasErr := m["error"]; !hasErr {
		t.Error("expected error when no auth info")
	}
}
