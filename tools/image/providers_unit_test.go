//go:build unit

package image_test

import (
	"testing"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/llmprovider"
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

func TestProvidersHandler_NoRegistry(t *testing.T) {
	saved := llmprovider.Global
	llmprovider.Global = nil
	t.Cleanup(func() { llmprovider.Global = saved })

	proc := &process.Process{
		Args: []interface{}{"image_generation"},
	}
	result := image.ProvidersHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if _, hasErr := m["error"]; !hasErr {
		t.Error("expected error when llmprovider registry is nil")
	}
}

func TestFindFirstImageGenConnector_NoGlobal(t *testing.T) {
	result := image.ExportFindFirstImageGenConn(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestSplitModelConnector_ConnectorIDNoProvider(t *testing.T) {
	p, m := image.ExportSplitModelConnector("", "t123.taoservice:image2.5")
	if p != "t123.taoservice" || m != "image2.5" {
		t.Errorf("expected (t123.taoservice, image2.5), got (%q, %q)", p, m)
	}
}

func TestSplitModelConnector_ProviderAlreadySet(t *testing.T) {
	p, m := image.ExportSplitModelConnector("explicit-provider", "t123.taoservice:image2.5")
	if p != "explicit-provider" || m != "t123.taoservice:image2.5" {
		t.Errorf("expected (explicit-provider, t123.taoservice:image2.5), got (%q, %q)", p, m)
	}
}

func TestSplitModelConnector_CleanModel(t *testing.T) {
	p, m := image.ExportSplitModelConnector("", "gpt-image-1")
	if p != "" || m != "gpt-image-1" {
		t.Errorf("expected (, gpt-image-1), got (%q, %q)", p, m)
	}
}

func TestSplitModelConnector_BothEmpty(t *testing.T) {
	p, m := image.ExportSplitModelConnector("", "")
	if p != "" || m != "" {
		t.Errorf("expected (, ), got (%q, %q)", p, m)
	}
}
