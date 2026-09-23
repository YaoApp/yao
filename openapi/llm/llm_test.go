package llm

import (
	"testing"

	"github.com/yaoapp/yao/llmprovider"
)

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

func TestBuildGroups_GroupsByFamily(t *testing.T) {
	providers := []Provider{
		{Label: "DeepSeek V4 Flash", Value: "deepseek.v4-flash", Type: "openai", Builtin: true},
		{Label: "DeepSeek V4 Flash Thinking", Value: "deepseek.v4-flash-thinking", Type: "openai", Builtin: true},
		{Label: "Claude Sonnet 4.6", Value: "claude.sonnet-4_6", Type: "anthropic", Builtin: true},
	}

	dsProvider := &llmprovider.Provider{Name: "DeepSeek V4 Flash", Source: llmprovider.ProviderSourceBuiltIn}
	clProvider := &llmprovider.Provider{Name: "Claude Sonnet 4.6", Source: llmprovider.ProviderSourceBuiltIn}

	index := map[string]*modelEntry{
		"deepseek.v4-flash": {
			model: &llmprovider.ModelInfo{
				ID: "deepseek.v4-flash",
				Metadata: map[string]interface{}{
					"model_name":       "DeepSeek V4 Flash",
					"model_family":     "deepseek-v4-flash",
					"reasoning_effort": "none",
					"group_name":       "DeepSeek",
				},
			},
			provider: dsProvider,
		},
		"deepseek.v4-flash-thinking": {
			model: &llmprovider.ModelInfo{
				ID: "deepseek.v4-flash-thinking",
				Metadata: map[string]interface{}{
					"model_name":       "DeepSeek V4 Flash",
					"model_family":     "deepseek-v4-flash",
					"reasoning_effort": "high",
					"group_name":       "DeepSeek",
				},
			},
			provider: dsProvider,
		},
		"claude.sonnet-4_6": {
			model: &llmprovider.ModelInfo{
				ID: "claude.sonnet-4_6",
				Metadata: map[string]interface{}{
					"model_name":       "Claude Sonnet 4.6",
					"model_family":     "claude-sonnet-4.6",
					"reasoning_effort": "none",
					"group_name":       "Claude",
				},
			},
			provider: clProvider,
		},
	}

	groups := buildGroups(providers, index)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// First group: DeepSeek — name from metadata group_name
	ds := groups[0]
	if ds.Name != "DeepSeek" {
		t.Errorf("group[0].Name = %q, want 'DeepSeek'", ds.Name)
	}
	if len(ds.Models) != 1 {
		t.Fatalf("DeepSeek group: expected 1 model family, got %d", len(ds.Models))
	}
	mg := ds.Models[0]
	if mg.ModelFamily != "deepseek-v4-flash" {
		t.Errorf("ModelFamily = %q, want deepseek-v4-flash", mg.ModelFamily)
	}
	if len(mg.Connectors) != 2 {
		t.Errorf("expected 2 connectors, got %d", len(mg.Connectors))
	}
	if mg.Connectors["none"] != "deepseek.v4-flash" {
		t.Errorf("none connector = %q", mg.Connectors["none"])
	}
	if mg.Connectors["high"] != "deepseek.v4-flash-thinking" {
		t.Errorf("high connector = %q", mg.Connectors["high"])
	}
	if len(mg.Levels) != 2 || mg.Levels[0] != "none" || mg.Levels[1] != "high" {
		t.Errorf("Levels = %v, want [none high]", mg.Levels)
	}

	// Second group: Claude — name from metadata group_name
	cl := groups[1]
	if cl.Name != "Claude" {
		t.Errorf("group[1].Name = %q, want 'Claude'", cl.Name)
	}
	if len(cl.Models) != 1 || cl.Models[0].ModelFamily != "claude-sonnet-4.6" {
		t.Errorf("Claude group models unexpected: %+v", cl.Models)
	}
}

func TestBuildGroups_DynamicProvider(t *testing.T) {
	providers := []Provider{
		{Label: "DeepSeek V4.1 Flash", Value: "deepseek-flash", Type: "openai", Builtin: false},
		{Label: "DeepSeek V4.1 Flash Thinking High", Value: "deepseek-flash-thinking", Type: "openai", Builtin: false},
	}

	index := map[string]*modelEntry{
		"deepseek-flash": {
			model: &llmprovider.ModelInfo{
				ID: "deepseek-flash",
				Metadata: map[string]interface{}{
					"model_name":       "DeepSeek V4.1 Flash",
					"model_family":     "deepseek-v4.1-flash",
					"reasoning_effort": "none",
				},
			},
			provider: &llmprovider.Provider{Name: "DeepSeek (OpenAI)", Source: llmprovider.ProviderSourceDynamic},
		},
		"deepseek-flash-thinking": {
			model: &llmprovider.ModelInfo{
				ID: "deepseek-flash-thinking",
				Metadata: map[string]interface{}{
					"model_name":       "DeepSeek V4.1 Flash",
					"model_family":     "deepseek-v4.1-flash",
					"reasoning_effort": "high",
				},
			},
			provider: &llmprovider.Provider{Name: "DeepSeek (OpenAI)", Source: llmprovider.ProviderSourceDynamic},
		},
	}

	groups := buildGroups(providers, index)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Name != "DeepSeek (OpenAI)" {
		t.Errorf("group name = %q, want 'DeepSeek (OpenAI)'", groups[0].Name)
	}
	mg := groups[0].Models[0]
	if len(mg.Connectors) != 2 {
		t.Errorf("expected 2 connectors, got %d", len(mg.Connectors))
	}
	if mg.Connectors["none"] != "deepseek-flash" || mg.Connectors["high"] != "deepseek-flash-thinking" {
		t.Errorf("connectors = %v", mg.Connectors)
	}
}

func TestBuildGroups_NoMetadata(t *testing.T) {
	providers := []Provider{
		{Label: "My Custom Model", Value: "custom:model-1", Type: "openai", Builtin: false},
	}

	groups := buildGroups(providers, nil)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	mg := groups[0].Models[0]
	if mg.ModelName != "My Custom Model" {
		t.Errorf("ModelName = %q, want label fallback", mg.ModelName)
	}
	if mg.Connectors["none"] != "custom:model-1" {
		t.Errorf("expected connector mapped to 'none', got %v", mg.Connectors)
	}
}

func TestSortedEffortLevels(t *testing.T) {
	connectors := map[string]string{
		"high": "c1",
		"none": "c2",
		"max":  "c3",
		"low":  "c4",
	}
	levels := sortedEffortLevels(connectors)
	expected := []string{"none", "low", "high", "max"}
	if len(levels) != len(expected) {
		t.Fatalf("len = %d, want %d", len(levels), len(expected))
	}
	for i, e := range expected {
		if levels[i] != e {
			t.Errorf("levels[%d] = %q, want %q", i, levels[i], e)
		}
	}
}
