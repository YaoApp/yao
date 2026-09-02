package llm

import "testing"

func TestMatchesFilters_OCR(t *testing.T) {
	caps := map[string]interface{}{
		"vision":    "openai",
		"ocr":       true,
		"streaming": true,
	}
	if !matchesFilters(caps, []string{"ocr"}) {
		t.Error("ocr=true must match filter 'ocr'")
	}
	if !matchesFilters(caps, []string{"ocr", "vision"}) {
		t.Error("ocr+vision must match when both present")
	}
	if matchesFilters(caps, []string{"ocr", "embedding"}) {
		t.Error("ocr+embedding must not match when embedding absent")
	}
}

func TestMatchesFilters_VisionString(t *testing.T) {
	caps := map[string]interface{}{"vision": "openai", "streaming": true}
	if !matchesFilters(caps, []string{"vision"}) {
		t.Error("vision='openai' (string) must match filter 'vision'")
	}
}

func TestMatchesFilters_VisionBool(t *testing.T) {
	caps := map[string]interface{}{"vision": true}
	if !matchesFilters(caps, []string{"vision"}) {
		t.Error("vision=true must match filter 'vision'")
	}
}

func TestMatchesFilters_VisionFalse(t *testing.T) {
	caps := map[string]interface{}{"vision": false}
	if matchesFilters(caps, []string{"vision"}) {
		t.Error("vision=false must not match filter 'vision'")
	}
}

func TestMatchesFilters_VisionEmpty(t *testing.T) {
	caps := map[string]interface{}{"vision": ""}
	if matchesFilters(caps, []string{"vision"}) {
		t.Error("vision='' (empty string) must not match filter 'vision'")
	}
}

func TestMatchesFilters_OCRFalse(t *testing.T) {
	caps := map[string]interface{}{"ocr": false, "streaming": true}
	if matchesFilters(caps, []string{"ocr"}) {
		t.Error("ocr=false must not match filter 'ocr'")
	}
}

func TestMatchesFilters_NilCapabilities(t *testing.T) {
	if matchesFilters(nil, []string{"ocr"}) {
		t.Error("nil capabilities must not match any filter")
	}
}

func TestMatchesFilters_EmptyFilters(t *testing.T) {
	caps := map[string]interface{}{"ocr": true}
	if !matchesFilters(caps, []string{}) {
		t.Error("empty filters must match any non-nil capabilities")
	}
}

func TestMatchesFilters_MultipleAND(t *testing.T) {
	caps := map[string]interface{}{
		"tool_calls": true,
		"streaming":  true,
		"reasoning":  true,
	}
	if !matchesFilters(caps, []string{"tool_calls", "streaming", "reasoning"}) {
		t.Error("all three present, must match AND filter")
	}
	if matchesFilters(caps, []string{"tool_calls", "streaming", "ocr"}) {
		t.Error("ocr missing, AND filter must fail")
	}
}

func TestIsNonChatModel(t *testing.T) {
	tests := []struct {
		name string
		caps map[string]interface{}
		want bool
	}{
		{"embedding", map[string]interface{}{"embedding": true}, true},
		{"image_generation", map[string]interface{}{"image_generation": true}, true},
		{"ocr_is_chat", map[string]interface{}{"ocr": true, "streaming": true}, false},
		{"vision_is_chat", map[string]interface{}{"vision": true, "tool_calls": true}, false},
		{"empty", map[string]interface{}{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNonChatModel(tt.caps); got != tt.want {
				t.Errorf("isNonChatModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasFilter(t *testing.T) {
	filters := []string{"vision", "ocr", "streaming"}
	if !hasFilter(filters, "ocr") {
		t.Error("'ocr' must be found in filters")
	}
	if hasFilter(filters, "embedding") {
		t.Error("'embedding' must not be found in filters")
	}
	if hasFilter(nil, "ocr") {
		t.Error("nil filters must not match")
	}
}
