//go:build unit

package image_test

import (
	"testing"

	image "github.com/yaoapp/yao/tools/image"
)

// --- extractExtra tests ---

func TestExtractExtra_MapValue(t *testing.T) {
	allArgs := map[string]interface{}{
		"extra": map[string]interface{}{"moderation": "low"},
	}
	m := image.ExportExtractExtra(allArgs)
	if m == nil {
		t.Fatal("expected non-nil map")
	}
	if m["moderation"] != "low" {
		t.Errorf("moderation = %v, want low", m["moderation"])
	}
}

func TestExtractExtra_JSONString(t *testing.T) {
	allArgs := map[string]interface{}{
		"extra": `{"input_fidelity":"high"}`,
	}
	m := image.ExportExtractExtra(allArgs)
	if m == nil {
		t.Fatal("expected non-nil map")
	}
	if m["input_fidelity"] != "high" {
		t.Errorf("input_fidelity = %v, want high", m["input_fidelity"])
	}
}

func TestExtractExtra_EmptyString(t *testing.T) {
	allArgs := map[string]interface{}{"extra": ""}
	m := image.ExportExtractExtra(allArgs)
	if m != nil {
		t.Errorf("expected nil for empty string, got %v", m)
	}
}

func TestExtractExtra_InvalidJSON(t *testing.T) {
	allArgs := map[string]interface{}{"extra": "not json"}
	m := image.ExportExtractExtra(allArgs)
	if m != nil {
		t.Errorf("expected nil for invalid JSON, got %v", m)
	}
}

func TestExtractExtra_Nil(t *testing.T) {
	allArgs := map[string]interface{}{}
	m := image.ExportExtractExtra(allArgs)
	if m != nil {
		t.Errorf("expected nil for missing key, got %v", m)
	}
}

// --- extractIntArg tests ---

func TestExtractIntArg_Float64(t *testing.T) {
	allArgs := map[string]interface{}{"output_compression": float64(85)}
	got := image.ExportExtractIntArg(allArgs, "output_compression", -1)
	if got != 85 {
		t.Errorf("got %d, want 85", got)
	}
}

func TestExtractIntArg_Int(t *testing.T) {
	allArgs := map[string]interface{}{"output_compression": 90}
	got := image.ExportExtractIntArg(allArgs, "output_compression", -1)
	if got != 90 {
		t.Errorf("got %d, want 90", got)
	}
}

func TestExtractIntArg_String(t *testing.T) {
	allArgs := map[string]interface{}{"output_compression": "75"}
	got := image.ExportExtractIntArg(allArgs, "output_compression", -1)
	if got != 75 {
		t.Errorf("got %d, want 75", got)
	}
}

func TestExtractIntArg_InvalidString(t *testing.T) {
	allArgs := map[string]interface{}{"output_compression": "abc"}
	got := image.ExportExtractIntArg(allArgs, "output_compression", -1)
	if got != -1 {
		t.Errorf("got %d, want -1 (fallback)", got)
	}
}

func TestExtractIntArg_Missing(t *testing.T) {
	allArgs := map[string]interface{}{}
	got := image.ExportExtractIntArg(allArgs, "output_compression", 100)
	if got != 100 {
		t.Errorf("got %d, want 100 (fallback)", got)
	}
}
