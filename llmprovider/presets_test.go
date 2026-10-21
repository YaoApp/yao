package llmprovider

import "testing"

func TestMiniMaxPresetsUseFlashFirstAndM3Second(t *testing.T) {
	keys := []string{
		"minimax_intl",
		"minimax_cn",
		"minimax_anthropic_intl",
		"minimax_anthropic_cn",
	}

	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			preset := GetPreset(key)
			if preset == nil {
				t.Fatalf("preset %q is missing", key)
			}
			if len(preset.DefaultModels) < 2 {
				t.Fatalf("preset %q has %d default models, want at least 2", key, len(preset.DefaultModels))
			}

			models := preset.DefaultModels
			if models[0].ID != "MiniMax-M3.1-Flash" || !models[0].Enabled {
				t.Fatalf("first model = %#v, want enabled MiniMax-M3.1-Flash", models[0])
			}
			if models[1].ID != "MiniMax-M3" || !models[1].Enabled {
				t.Fatalf("second model = %#v, want enabled MiniMax-M3", models[1])
			}

			for _, model := range models[2:] {
				if model.Enabled {
					t.Errorf("legacy model %q is enabled; ordinary choices must stop at Flash and M3", model.ID)
				}
			}

			var foundLegacyThinking bool
			for _, model := range models {
				if model.ID == "MiniMax-M3-thinking" {
					foundLegacyThinking = true
					if model.Model != "MiniMax-M3" || model.Enabled {
						t.Errorf("legacy MiniMax-M3-thinking model = %#v, want preserved and disabled", model)
					}
				}
			}
			if !foundLegacyThinking {
				t.Error("legacy MiniMax-M3-thinking model was removed")
			}
		})
	}
}
