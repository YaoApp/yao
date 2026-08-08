//go:build unit

package image_test

import (
	"testing"

	"github.com/yaoapp/gou/process"
	image "github.com/yaoapp/yao/tools/image"
)

func TestModelHasCapability_Found(t *testing.T) {
	caps := []string{"chat", "image_generation", "vision"}
	if !image.ExportModelHasCapability(caps, "image_generation") {
		t.Error("expected true for image_generation")
	}
	if !image.ExportModelHasCapability(caps, "vision") {
		t.Error("expected true for vision")
	}
}

func TestModelHasCapability_NotFound(t *testing.T) {
	caps := []string{"chat", "embedding"}
	if image.ExportModelHasCapability(caps, "image_generation") {
		t.Error("expected false for image_generation")
	}
}

func TestModelHasCapability_Empty(t *testing.T) {
	if image.ExportModelHasCapability(nil, "image_generation") {
		t.Error("expected false for nil caps")
	}
	if image.ExportModelHasCapability([]string{}, "image_generation") {
		t.Error("expected false for empty caps")
	}
}

func TestProvidersHandler_NoAuth(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{"image_generation"},
	}
	result := image.ProvidersHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if _, hasErr := m["error"]; !hasErr {
		t.Error("expected error when no auth info")
	}
}

func TestFindFirstImageGenConnector_NoGlobal(t *testing.T) {
	result := image.ExportFindFirstImageGenConn(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
