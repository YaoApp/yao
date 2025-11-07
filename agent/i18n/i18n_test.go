package i18n

import (
	"testing"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestParseString tests the parseString method
func TestParseString(t *testing.T) {
	i18n := I18n{
		Locale: "en",
		Messages: map[string]any{
			"hello":       "Hello",
			"world":       "World",
			"greeting":    "Hello, World!",
			"description": "This is a test",
		},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Template expression with match",
			input:    "{{greeting}}",
			expected: "Hello, World!",
		},
		{
			name:     "Template expression with spaces",
			input:    "{{ greeting }}",
			expected: "Hello, World!",
		},
		{
			name:     "Template expression without match",
			input:    "{{notfound}}",
			expected: "{{notfound}}",
		},
		{
			name:     "Direct message key",
			input:    "hello",
			expected: "Hello",
		},
		{
			name:     "Direct message key with spaces",
			input:    " world ",
			expected: "World",
		},
		{
			name:     "Non-existent key",
			input:    "notfound",
			expected: "notfound",
		},
		{
			name:     "Regular text",
			input:    "Just some text",
			expected: "Just some text",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := i18n.parseString(tt.input)
			if result != tt.expected {
				t.Errorf("parseString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseStringNonStringValue tests parseString when message value is not a string
func TestParseStringNonStringValue(t *testing.T) {
	i18n := I18n{
		Locale: "en",
		Messages: map[string]any{
			"number": 123,
			"object": map[string]any{"key": "value"},
		},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Template with number value",
			input:    "{{number}}",
			expected: "{{number}}",
		},
		{
			name:     "Direct key with number value",
			input:    "number",
			expected: "number",
		},
		{
			name:     "Template with object value",
			input:    "{{object}}",
			expected: "{{object}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := i18n.parseString(tt.input)
			if result != tt.expected {
				t.Errorf("parseString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParse tests the Parse method with various input types
func TestParse(t *testing.T) {
	i18n := I18n{
		Locale: "en",
		Messages: map[string]any{
			"name":        "John",
			"description": "A developer",
			"title":       "Welcome",
		},
	}

	t.Run("Nil input", func(t *testing.T) {
		result := i18n.Parse(nil)
		if result != nil {
			t.Errorf("Parse(nil) = %v, want nil", result)
		}
	})

	t.Run("String input", func(t *testing.T) {
		result := i18n.Parse("{{name}}")
		if result != "John" {
			t.Errorf("Parse({{name}}) = %v, want 'John'", result)
		}
	})

	t.Run("Map input", func(t *testing.T) {
		input := map[string]any{
			"name":        "{{name}}",
			"description": "{{description}}",
			"age":         30,
		}
		result := i18n.Parse(input)
		if resultMap, ok := result.(map[string]any); ok {
			if resultMap["name"] != "John" {
				t.Errorf("Expected name 'John', got %v", resultMap["name"])
			}
			if resultMap["description"] != "A developer" {
				t.Errorf("Expected description 'A developer', got %v", resultMap["description"])
			}
			if resultMap["age"] != 30 {
				t.Errorf("Expected age 30, got %v", resultMap["age"])
			}
		} else {
			t.Errorf("Expected map[string]any, got %T", result)
		}
	})

	t.Run("Slice of any", func(t *testing.T) {
		input := []any{"{{name}}", "{{description}}", 123}
		result := i18n.Parse(input)
		if resultSlice, ok := result.([]any); ok {
			if len(resultSlice) != 3 {
				t.Errorf("Expected 3 elements, got %d", len(resultSlice))
			}
			if resultSlice[0] != "John" {
				t.Errorf("Expected 'John', got %v", resultSlice[0])
			}
			if resultSlice[1] != "A developer" {
				t.Errorf("Expected 'A developer', got %v", resultSlice[1])
			}
			if resultSlice[2] != 123 {
				t.Errorf("Expected 123, got %v", resultSlice[2])
			}
		} else {
			t.Errorf("Expected []any, got %T", result)
		}
	})

	t.Run("Slice of strings", func(t *testing.T) {
		input := []string{"{{name}}", "{{description}}", "plain text"}
		result := i18n.Parse(input)
		if resultSlice, ok := result.([]string); ok {
			if len(resultSlice) != 3 {
				t.Errorf("Expected 3 elements, got %d", len(resultSlice))
			}
			if resultSlice[0] != "John" {
				t.Errorf("Expected 'John', got %v", resultSlice[0])
			}
			if resultSlice[1] != "A developer" {
				t.Errorf("Expected 'A developer', got %v", resultSlice[1])
			}
			if resultSlice[2] != "plain text" {
				t.Errorf("Expected 'plain text', got %v", resultSlice[2])
			}
		} else {
			t.Errorf("Expected []string, got %T", result)
		}
	})

	t.Run("Nested structures", func(t *testing.T) {
		input := map[string]any{
			"user": map[string]any{
				"name": "{{name}}",
				"info": []string{"{{title}}", "{{description}}"},
			},
		}
		result := i18n.Parse(input)
		if resultMap, ok := result.(map[string]any); ok {
			if userMap, ok := resultMap["user"].(map[string]any); ok {
				if userMap["name"] != "John" {
					t.Errorf("Expected nested name 'John', got %v", userMap["name"])
				}
				if infoSlice, ok := userMap["info"].([]any); ok {
					if infoSlice[0] != "Welcome" {
						t.Errorf("Expected 'Welcome', got %v", infoSlice[0])
					}
				}
			}
		}
	})

	t.Run("Other types pass through", func(t *testing.T) {
		input := 12345
		result := i18n.Parse(input)
		if result != input {
			t.Errorf("Expected %v, got %v", input, result)
		}
	})
}

// TestParseSliceStringWithNilAndNonString tests []string parsing edge cases
func TestParseSliceStringWithNilAndNonString(t *testing.T) {
	i18n := I18n{
		Locale: "en",
		Messages: map[string]any{
			"key1": "value1",
			"key2": 123, // Non-string value
			"key3": nil, // Nil value
		},
	}

	t.Run("String slice with fallback", func(t *testing.T) {
		input := []string{"{{key1}}", "{{key2}}", "{{notfound}}"}
		result := i18n.Parse(input)
		if resultSlice, ok := result.([]string); ok {
			if resultSlice[0] != "value1" {
				t.Errorf("Expected 'value1', got %v", resultSlice[0])
			}
			// key2 has non-string value, should fallback to original
			if resultSlice[1] != "{{key2}}" {
				t.Errorf("Expected '{{key2}}', got %v", resultSlice[1])
			}
			if resultSlice[2] != "{{notfound}}" {
				t.Errorf("Expected '{{notfound}}', got %v", resultSlice[2])
			}
		} else {
			t.Errorf("Expected []string, got %T", result)
		}
	})

	t.Run("String slice with nil parsed result", func(t *testing.T) {
		// This tests the case where Parse returns nil for a string
		input := []string{"{{key3}}", "normal"}
		result := i18n.Parse(input)
		if resultSlice, ok := result.([]string); ok {
			// When parsed is nil, should fallback to original
			if resultSlice[0] != "{{key3}}" {
				t.Errorf("Expected '{{key3}}' (tests nil parsed branch), got %v", resultSlice[0])
			}
			if resultSlice[1] != "normal" {
				t.Errorf("Expected 'normal', got %v", resultSlice[1])
			}
		} else {
			t.Errorf("Expected []string, got %T", result)
		}
	})

	t.Run("String slice with non-string parsed result from map", func(t *testing.T) {
		i18nWithMap := I18n{
			Locale: "en",
			Messages: map[string]any{
				"map_key": map[string]any{"nested": "value"},
			},
		}
		// When Parse returns a non-string type (like a map), should fallback
		input := []string{"{{map_key}}", "text"}
		result := i18nWithMap.Parse(input)
		if resultSlice, ok := result.([]string); ok {
			// Should fallback to original when parsed is not string
			if resultSlice[0] != "{{map_key}}" {
				t.Errorf("Expected '{{map_key}}' (tests non-string parsed branch), got %v", resultSlice[0])
			}
		} else {
			t.Errorf("Expected []string, got %T", result)
		}
	})
}

// TestMapFlatten tests the Flatten method
func TestMapFlatten(t *testing.T) {
	i18ns := Map{
		"en-us": I18n{
			Locale: "en-us",
			Messages: map[string]any{
				"greeting": "Hello",
			},
		},
		"zh-cn": I18n{
			Locale: "zh-cn",
			Messages: map[string]any{
				"greeting": "你好",
			},
		},
	}

	flattened := i18ns.Flatten()

	// Should have original keys
	if _, ok := flattened["en-us"]; !ok {
		t.Error("Expected 'en-us' key in flattened map")
	}
	if _, ok := flattened["zh-cn"]; !ok {
		t.Error("Expected 'zh-cn' key in flattened map")
	}

	// Should have short lang codes
	if _, ok := flattened["en"]; !ok {
		t.Error("Expected 'en' short code in flattened map")
	}
	if _, ok := flattened["us"]; !ok {
		t.Error("Expected 'us' region code in flattened map")
	}
	if _, ok := flattened["zh"]; !ok {
		t.Error("Expected 'zh' short code in flattened map")
	}
	if _, ok := flattened["cn"]; !ok {
		t.Error("Expected 'cn' region code in flattened map")
	}

	// Verify messages are preserved
	if msg, ok := flattened["en"].Messages["greeting"].(string); !ok || msg != "Hello" {
		t.Errorf("Expected 'Hello', got %v", flattened["en"].Messages["greeting"])
	}
	if msg, ok := flattened["zh"].Messages["greeting"].(string); !ok || msg != "你好" {
		t.Errorf("Expected '你好', got %v", flattened["zh"].Messages["greeting"])
	}
}

// TestMapFlattenWithGlobal tests the FlattenWithGlobal method
func TestMapFlattenWithGlobal(t *testing.T) {
	// Setup global locales
	Locales["__global__"] = map[string]I18n{
		"en": {
			Locale: "en",
			Messages: map[string]any{
				"global.key": "Global Value",
				"common":     "Common",
			},
		},
	}

	defer delete(Locales, "__global__")

	i18ns := Map{
		"en": I18n{
			Locale: "en",
			Messages: map[string]any{
				"local.key": "Local Value",
				"common":    "Local Common", // Should override global
			},
		},
	}

	flattened := i18ns.FlattenWithGlobal()

	if _, ok := flattened["en"]; !ok {
		t.Fatal("Expected 'en' key in flattened map")
	}

	// Should have local key
	if val, ok := flattened["en"].Messages["local.key"].(string); !ok || val != "Local Value" {
		t.Errorf("Expected 'Local Value', got %v", flattened["en"].Messages["local.key"])
	}

	// Should have global key
	if val, ok := flattened["en"].Messages["global.key"].(string); !ok || val != "Global Value" {
		t.Errorf("Expected 'Global Value', got %v", flattened["en"].Messages["global.key"])
	}

	// Local should override global
	if val, ok := flattened["en"].Messages["common"].(string); !ok || val != "Local Common" {
		t.Errorf("Expected 'Local Common', got %v", flattened["en"].Messages["common"])
	}
}

// TestMapFlattenWithGlobalNoGlobal tests FlattenWithGlobal when no global exists
func TestMapFlattenWithGlobalNoGlobal(t *testing.T) {
	// Make sure no global exists
	delete(Locales, "__global__")

	i18ns := Map{
		"en": I18n{
			Locale: "en",
			Messages: map[string]any{
				"key": "value",
			},
		},
	}

	flattened := i18ns.FlattenWithGlobal()

	if _, ok := flattened["en"]; !ok {
		t.Fatal("Expected 'en' key in flattened map")
	}

	if val, ok := flattened["en"].Messages["key"].(string); !ok || val != "value" {
		t.Errorf("Expected 'value', got %v", flattened["en"].Messages["key"])
	}
}

// TestMapFlattenWithGlobalKeyConflict tests FlattenWithGlobal when local keys already exist
func TestMapFlattenWithGlobalKeyConflict(t *testing.T) {
	// Setup global with keys in flat format (after Dot())
	Locales["__global__"] = map[string]I18n{
		"en": {
			Locale: "en",
			Messages: map[string]any{
				"shared.key":  "Global Shared",
				"global.only": "Global Only",
				"local.key":   "Global Local", // Will be overridden
			},
		},
	}
	defer delete(Locales, "__global__")

	// Local messages in nested format (will be flattened by Dot())
	i18ns := Map{
		"en": I18n{
			Locale: "en",
			Messages: map[string]any{
				"local": map[string]any{
					"key": "Local Value", // After Dot() becomes "local.key", should override global
				},
				"unique": map[string]any{
					"key": "Local Unique",
				},
			},
		},
	}

	flattened := i18ns.FlattenWithGlobal()

	if _, ok := flattened["en"]; !ok {
		t.Fatal("Expected 'en' key in flattened map")
	}

	// Local key should exist and NOT be overridden by global
	if val, ok := flattened["en"].Messages["local.key"].(string); !ok || val != "Local Value" {
		t.Errorf("Expected 'Local Value' (local should override global), got %v", flattened["en"].Messages["local.key"])
	}

	// Global only key should exist
	if val, ok := flattened["en"].Messages["global.only"].(string); !ok || val != "Global Only" {
		t.Errorf("Expected 'Global Only', got %v", flattened["en"].Messages["global.only"])
	}

	// Unique local key should exist
	if val, ok := flattened["en"].Messages["unique.key"].(string); !ok || val != "Local Unique" {
		t.Errorf("Expected 'Local Unique', got %v", flattened["en"].Messages["unique.key"])
	}

	// Shared key from global should exist
	if val, ok := flattened["en"].Messages["shared.key"].(string); !ok || val != "Global Shared" {
		t.Errorf("Expected 'Global Shared', got %v", flattened["en"].Messages["shared.key"])
	}
}

// TestTranslate tests the Translate function
func TestTranslate(t *testing.T) {
	assistantID := "test-assistant"
	Locales[assistantID] = map[string]I18n{
		"en": {
			Locale: "en",
			Messages: map[string]any{
				"greeting": "Hello",
				"name":     "John",
			},
		},
		"zh-cn": {
			Locale: "zh-cn",
			Messages: map[string]any{
				"greeting": "你好",
				"name":     "张三",
			},
		},
	}
	defer delete(Locales, assistantID)

	t.Run("Translate with exact locale match", func(t *testing.T) {
		result := Translate(assistantID, "en", "{{greeting}}")
		if result != "Hello" {
			t.Errorf("Expected 'Hello', got %v", result)
		}
	})

	t.Run("Translate with locale variant", func(t *testing.T) {
		result := Translate(assistantID, "zh-CN", "{{greeting}}")
		if result != "你好" {
			t.Errorf("Expected '你好', got %v", result)
		}
	})

	t.Run("Translate with short locale code", func(t *testing.T) {
		result := Translate(assistantID, "en-us", "{{name}}")
		if result != "John" {
			t.Errorf("Expected 'John', got %v", result)
		}
	})

	t.Run("Translate without locale match", func(t *testing.T) {
		result := Translate(assistantID, "fr", "{{greeting}}")
		// Should return original when no locale found
		if result != "{{greeting}}" {
			t.Errorf("Expected '{{greeting}}', got %v", result)
		}
	})

	t.Run("Translate non-existent assistant", func(t *testing.T) {
		result := Translate("nonexistent", "en", "{{greeting}}")
		if result != "{{greeting}}" {
			t.Errorf("Expected '{{greeting}}', got %v", result)
		}
	})

	t.Run("Translate with fallback to global", func(t *testing.T) {
		Locales["__global__"] = map[string]I18n{
			"es": {
				Locale: "es",
				Messages: map[string]any{
					"greeting": "Hola",
				},
			},
		}
		defer delete(Locales, "__global__")

		result := Translate(assistantID, "es", "{{greeting}}")
		if result != "Hola" {
			t.Errorf("Expected 'Hola', got %v", result)
		}
	})

	t.Run("Translate complex structure", func(t *testing.T) {
		input := map[string]any{
			"title": "{{greeting}}",
			"user":  "{{name}}",
		}
		result := Translate(assistantID, "zh-cn", input)
		if resultMap, ok := result.(map[string]any); ok {
			if resultMap["title"] != "你好" {
				t.Errorf("Expected '你好', got %v", resultMap["title"])
			}
			if resultMap["user"] != "张三" {
				t.Errorf("Expected '张三', got %v", resultMap["user"])
			}
		} else {
			t.Errorf("Expected map[string]any, got %T", result)
		}
	})
}

// TestTranslateGlobal tests the TranslateGlobal function
func TestTranslateGlobal(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	Locales["__global__"] = map[string]I18n{
		"en": {
			Locale: "en",
			Messages: map[string]any{
				"button.ok":     "OK",
				"button.cancel": "Cancel",
			},
		},
		"zh-cn": {
			Locale: "zh-cn",
			Messages: map[string]any{
				"button.ok":     "确定",
				"button.cancel": "取消",
			},
		},
		"zh": {
			Locale: "zh",
			Messages: map[string]any{
				"button.ok":     "确定",
				"button.cancel": "取消",
			},
		},
	}
	defer delete(Locales, "__global__")

	t.Run("TranslateGlobal with match", func(t *testing.T) {
		result := TranslateGlobal("en", "{{button.ok}}")
		if result != "OK" {
			t.Errorf("Expected 'OK', got %v", result)
		}
	})

	t.Run("TranslateGlobal with Chinese", func(t *testing.T) {
		result := TranslateGlobal("zh-cn", "{{button.cancel}}")
		if result != "取消" {
			t.Errorf("Expected '取消', got %v", result)
		}
	})

	t.Run("TranslateGlobal with short code", func(t *testing.T) {
		result := TranslateGlobal("zh-TW", "{{button.ok}}")
		if result != "确定" {
			t.Errorf("Expected '确定', got %v", result)
		}
	})

	t.Run("TranslateGlobal without match", func(t *testing.T) {
		result := TranslateGlobal("fr", "{{button.ok}}")
		if result != "{{button.ok}}" {
			t.Errorf("Expected '{{button.ok}}', got %v", result)
		}
	})

	t.Run("TranslateGlobal no global", func(t *testing.T) {
		delete(Locales, "__global__")
		result := TranslateGlobal("en", "{{button.ok}}")
		if result != "{{button.ok}}" {
			t.Errorf("Expected '{{button.ok}}', got %v", result)
		}
		// Restore for cleanup
		Locales["__global__"] = map[string]I18n{}
	})
}

// TestGetLocalesIntegration tests GetLocales with real assistant data
func TestGetLocalesIntegration(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Use the real mohe assistant path (relative to app root)
	assistantPath := "/assistants/mohe"

	t.Run("Load real locale files", func(t *testing.T) {
		locales, err := GetLocales(assistantPath)
		if err != nil {
			t.Skipf("Skipping: %v", err)
			return
		}

		// Should have at least 2 locales (en-us and zh-cn)
		if len(locales) < 2 {
			t.Errorf("Expected at least 2 locales, got %d", len(locales))
		}

		// Check en-us locale
		if enUS, ok := locales["en-us"]; ok {
			if enUS.Locale != "en-us" {
				t.Errorf("Expected locale 'en-us', got %s", enUS.Locale)
			}

			// Check some messages
			if desc, ok := enUS.Messages["description"].(string); ok {
				if desc == "" {
					t.Error("Expected non-empty description")
				}
				t.Logf("English description: %s", desc)
			}

			if chat, ok := enUS.Messages["chat"].(map[string]interface{}); ok {
				if title, ok := chat["title"].(string); ok {
					if title != "New Chat" {
						t.Errorf("Expected 'New Chat', got %s", title)
					}
				}
			}
		} else {
			t.Error("Expected 'en-us' locale")
		}

		// Check zh-cn locale
		if zhCN, ok := locales["zh-cn"]; ok {
			if zhCN.Locale != "zh-cn" {
				t.Errorf("Expected locale 'zh-cn', got %s", zhCN.Locale)
			}

			// Check some messages
			if desc, ok := zhCN.Messages["description"].(string); ok {
				if desc == "" {
					t.Error("Expected non-empty description")
				}
				t.Logf("Chinese description: %s", desc)
			}

			if chat, ok := zhCN.Messages["chat"].(map[string]interface{}); ok {
				if title, ok := chat["title"].(string); ok {
					if title != "新对话" {
						t.Errorf("Expected '新对话', got %s", title)
					}
				}
			}
		} else {
			t.Error("Expected 'zh-cn' locale")
		}

		t.Logf("Loaded %d locales successfully", len(locales))
	})

	t.Run("Flatten loaded locales", func(t *testing.T) {
		locales, err := GetLocales(assistantPath)
		if err != nil {
			t.Skipf("Skipping: %v", err)
			return
		}

		flattened := locales.Flatten()

		// Should have short codes
		if _, ok := flattened["en"]; !ok {
			t.Error("Expected 'en' short code after flatten")
		}
		if _, ok := flattened["zh"]; !ok {
			t.Error("Expected 'zh' short code after flatten")
		}
		if _, ok := flattened["us"]; !ok {
			t.Error("Expected 'us' region code after flatten")
		}
		if _, ok := flattened["cn"]; !ok {
			t.Error("Expected 'cn' region code after flatten")
		}

		// Verify flattened messages structure
		if en, ok := flattened["en"]; ok {
			if _, ok := en.Messages["chat.title"]; !ok {
				t.Error("Expected flattened 'chat.title' key")
			}
			if _, ok := en.Messages["chat.description"]; !ok {
				t.Error("Expected flattened 'chat.description' key")
			}
			if _, ok := en.Messages["chat.prompts.0"]; !ok {
				t.Error("Expected flattened 'chat.prompts.0' key")
			}
		}

		t.Logf("Flattened to %d locale codes", len(flattened))
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("Empty Messages map", func(t *testing.T) {
		i18n := I18n{
			Locale:   "en",
			Messages: map[string]any{},
		}
		result := i18n.Parse("{{key}}")
		if result != "{{key}}" {
			t.Errorf("Expected '{{key}}', got %v", result)
		}
	})

	t.Run("Nil Messages map", func(t *testing.T) {
		i18n := I18n{
			Locale:   "en",
			Messages: nil,
		}
		result := i18n.Parse("{{key}}")
		if result != "{{key}}" {
			t.Errorf("Expected '{{key}}', got %v", result)
		}
	})

	t.Run("Empty locale string", func(t *testing.T) {
		Locales["test"] = map[string]I18n{
			"en": {
				Locale:   "en",
				Messages: map[string]any{"key": "value"},
			},
		}
		defer delete(Locales, "test")

		result := Translate("test", "", "{{key}}")
		// Should still work with empty string after trim
		if result != "{{key}}" {
			t.Logf("Result: %v", result)
		}
	})

	t.Run("Locale with only spaces", func(t *testing.T) {
		Locales["test"] = map[string]I18n{
			"": {
				Locale:   "",
				Messages: map[string]any{"key": "value"},
			},
		}
		defer delete(Locales, "test")

		result := Translate("test", "   ", "{{key}}")
		if result != "value" {
			t.Errorf("Expected 'value', got %v", result)
		}
	})
}
