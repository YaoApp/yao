package image

import (
	"testing"

	"github.com/yaoapp/gou/process"
)

func TestModelHasCapability_Found(t *testing.T) {
	caps := []string{"chat", "image_generation", "vision"}
	if !modelHasCapability(caps, "image_generation") {
		t.Error("expected true for image_generation")
	}
	if !modelHasCapability(caps, "vision") {
		t.Error("expected true for vision")
	}
}

func TestModelHasCapability_NotFound(t *testing.T) {
	caps := []string{"chat", "embedding"}
	if modelHasCapability(caps, "image_generation") {
		t.Error("expected false for image_generation")
	}
}

func TestModelHasCapability_Empty(t *testing.T) {
	if modelHasCapability(nil, "image_generation") {
		t.Error("expected false for nil caps")
	}
	if modelHasCapability([]string{}, "image_generation") {
		t.Error("expected false for empty caps")
	}
}

func TestProvidersHandler_NoAuth(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{"image_generation"},
	}
	result := ProvidersHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if _, hasErr := m["error"]; !hasErr {
		t.Error("expected error when no auth info")
	}
}

func TestFindFirstImageGenConnector_NoGlobal(t *testing.T) {
	result := findFirstImageGenConnector(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
