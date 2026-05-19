//go:build unit

package i18n_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	i18n "github.com/yaoapp/yao/agent/i18n"
)

func TestParseString(t *testing.T) {
	inst := i18n.I18n{
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
		{"Template expression with match", "{{greeting}}", "Hello, World!"},
		{"Template expression with spaces", "{{ greeting }}", "Hello, World!"},
		{"Template expression without match", "{{notfound}}", "{{notfound}}"},
		{"Direct message key", "hello", "Hello"},
		{"Direct message key with spaces", " world ", "World"},
		{"Non-existent key", "notfound", "notfound"},
		{"Regular text", "Just some text", "Just some text"},
		{"Empty string", "", ""},
		{"Embedded single template", "Hello {{hello}}", "Hello Hello"},
		{"Embedded multiple templates", "{{hello}} {{world}}!", "Hello World!"},
		{"Embedded template with spaces", "Say {{ hello }} to the {{ world }}", "Say Hello to the World"},
		{"Embedded template mixed with text", "Message: {{greeting}} - {{description}}", "Message: Hello, World! - This is a test"},
		{"Embedded template not found", "Hello {{notfound}} World", "Hello {{notfound}} World"},
		{"Embedded template partial match", "{{hello}} {{notfound}} {{world}}", "Hello {{notfound}} World"},
		{"Only opening braces", "{{hello", "{{hello"},
		{"Only closing braces", "hello}}", "hello}}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := i18n.ParseStringForTest(&inst, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseStringNonStringValue(t *testing.T) {
	inst := i18n.I18n{
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
		{"Template with number value", "{{number}}", "{{number}}"},
		{"Direct key with number value", "number", "number"},
		{"Template with object value", "{{object}}", "{{object}}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := i18n.ParseStringForTest(&inst, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParse(t *testing.T) {
	inst := i18n.I18n{
		Locale: "en",
		Messages: map[string]any{
			"name":        "John",
			"description": "A developer",
			"title":       "Welcome",
		},
	}

	t.Run("Nil input", func(t *testing.T) {
		assert.Nil(t, inst.Parse(nil))
	})

	t.Run("String input", func(t *testing.T) {
		assert.Equal(t, "John", inst.Parse("{{name}}"))
	})

	t.Run("Map input", func(t *testing.T) {
		input := map[string]any{
			"name":        "{{name}}",
			"description": "{{description}}",
			"age":         30,
		}
		result, ok := inst.Parse(input).(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "John", result["name"])
		assert.Equal(t, "A developer", result["description"])
		assert.Equal(t, 30, result["age"])
	})

	t.Run("Slice of any", func(t *testing.T) {
		input := []any{"{{name}}", "{{description}}", 123}
		result, ok := inst.Parse(input).([]any)
		require.True(t, ok)
		assert.Len(t, result, 3)
		assert.Equal(t, "John", result[0])
		assert.Equal(t, "A developer", result[1])
		assert.Equal(t, 123, result[2])
	})

	t.Run("Slice of strings", func(t *testing.T) {
		input := []string{"{{name}}", "{{description}}", "plain text"}
		result, ok := inst.Parse(input).([]string)
		require.True(t, ok)
		assert.Equal(t, []string{"John", "A developer", "plain text"}, result)
	})

	t.Run("Nested structures", func(t *testing.T) {
		input := map[string]any{
			"user": map[string]any{
				"name": "{{name}}",
				"info": []string{"{{title}}", "{{description}}"},
			},
		}
		result, ok := inst.Parse(input).(map[string]any)
		require.True(t, ok)
		userMap, ok := result["user"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "John", userMap["name"])
		infoSlice, ok := userMap["info"].([]string)
		require.True(t, ok)
		assert.Equal(t, "Welcome", infoSlice[0])
		assert.Equal(t, "A developer", infoSlice[1])
	})

	t.Run("Other types pass through", func(t *testing.T) {
		assert.Equal(t, 12345, inst.Parse(12345))
	})
}

func TestParseSliceStringWithNilAndNonString(t *testing.T) {
	inst := i18n.I18n{
		Locale: "en",
		Messages: map[string]any{
			"key1": "value1",
			"key2": 123,
			"key3": nil,
		},
	}

	t.Run("Non-string value fallback", func(t *testing.T) {
		input := []string{"{{key1}}", "{{key2}}", "{{notfound}}"}
		result, ok := inst.Parse(input).([]string)
		require.True(t, ok)
		assert.Equal(t, "value1", result[0])
		assert.Equal(t, "{{key2}}", result[1])
		assert.Equal(t, "{{notfound}}", result[2])
	})

	t.Run("Nil parsed result", func(t *testing.T) {
		input := []string{"{{key3}}", "normal"}
		result, ok := inst.Parse(input).([]string)
		require.True(t, ok)
		assert.Equal(t, "{{key3}}", result[0])
		assert.Equal(t, "normal", result[1])
	})

	t.Run("Non-string parsed result from map", func(t *testing.T) {
		instWithMap := i18n.I18n{
			Locale: "en",
			Messages: map[string]any{
				"map_key": map[string]any{"nested": "value"},
			},
		}
		input := []string{"{{map_key}}", "text"}
		result, ok := instWithMap.Parse(input).([]string)
		require.True(t, ok)
		assert.Equal(t, "{{map_key}}", result[0])
	})
}

func TestMapFlatten(t *testing.T) {
	orig := i18n.Locales["__global__"]
	t.Cleanup(func() { i18n.Locales["__global__"] = orig })
	delete(i18n.Locales, "__global__")

	m := i18n.Map{
		"en-us": i18n.I18n{
			Locale:   "en-us",
			Messages: map[string]any{"greeting": "Hello"},
		},
		"zh-cn": i18n.I18n{
			Locale:   "zh-cn",
			Messages: map[string]any{"greeting": "你好"},
		},
	}

	flattened := m.Flatten()

	assert.Contains(t, flattened, "en-us")
	assert.Contains(t, flattened, "zh-cn")
	assert.Contains(t, flattened, "en")
	assert.Contains(t, flattened, "us")
	assert.Contains(t, flattened, "zh")
	assert.Contains(t, flattened, "cn")

	assert.Equal(t, "Hello", flattened["en"].Messages["greeting"])
	assert.Equal(t, "你好", flattened["zh"].Messages["greeting"])
}

func TestMapFlattenWithGlobal(t *testing.T) {
	orig := i18n.Locales["__global__"]
	t.Cleanup(func() { i18n.Locales["__global__"] = orig })

	i18n.Locales["__global__"] = map[string]i18n.I18n{
		"en": {
			Locale: "en",
			Messages: map[string]any{
				"global.key": "Global Value",
				"common":     "Common",
			},
		},
	}

	m := i18n.Map{
		"en": i18n.I18n{
			Locale: "en",
			Messages: map[string]any{
				"local.key": "Local Value",
				"common":    "Local Common",
			},
		},
	}

	flattened := m.FlattenWithGlobal()
	require.Contains(t, flattened, "en")

	assert.Equal(t, "Local Value", flattened["en"].Messages["local.key"])
	assert.Equal(t, "Global Value", flattened["en"].Messages["global.key"])
	assert.Equal(t, "Local Common", flattened["en"].Messages["common"])
}

func TestMapFlattenWithGlobalNoGlobal(t *testing.T) {
	orig := i18n.Locales["__global__"]
	t.Cleanup(func() { i18n.Locales["__global__"] = orig })

	delete(i18n.Locales, "__global__")

	m := i18n.Map{
		"en": i18n.I18n{
			Locale:   "en",
			Messages: map[string]any{"key": "value"},
		},
	}

	flattened := m.FlattenWithGlobal()
	require.Contains(t, flattened, "en")
	assert.Equal(t, "value", flattened["en"].Messages["key"])
}

func TestMapFlattenWithGlobalKeyConflict(t *testing.T) {
	orig := i18n.Locales["__global__"]
	t.Cleanup(func() { i18n.Locales["__global__"] = orig })

	i18n.Locales["__global__"] = map[string]i18n.I18n{
		"en": {
			Locale: "en",
			Messages: map[string]any{
				"shared.key":  "Global Shared",
				"global.only": "Global Only",
				"local.key":   "Global Local",
			},
		},
	}

	m := i18n.Map{
		"en": i18n.I18n{
			Locale: "en",
			Messages: map[string]any{
				"local": map[string]any{
					"key": "Local Value",
				},
				"unique": map[string]any{
					"key": "Local Unique",
				},
			},
		},
	}

	flattened := m.FlattenWithGlobal()
	require.Contains(t, flattened, "en")

	assert.Equal(t, "Local Value", flattened["en"].Messages["local.key"])
	assert.Equal(t, "Global Only", flattened["en"].Messages["global.only"])
	assert.Equal(t, "Local Unique", flattened["en"].Messages["unique.key"])
	assert.Equal(t, "Global Shared", flattened["en"].Messages["shared.key"])
}

func TestTranslate(t *testing.T) {
	assistantID := "test-translate-assistant"
	i18n.Locales[assistantID] = map[string]i18n.I18n{
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
	t.Cleanup(func() { delete(i18n.Locales, assistantID) })

	t.Run("Exact locale match", func(t *testing.T) {
		assert.Equal(t, "Hello", i18n.Translate(assistantID, "en", "{{greeting}}"))
	})

	t.Run("Locale variant zh-CN", func(t *testing.T) {
		assert.Equal(t, "你好", i18n.Translate(assistantID, "zh-CN", "{{greeting}}"))
	})

	t.Run("Short code en-us", func(t *testing.T) {
		assert.Equal(t, "John", i18n.Translate(assistantID, "en-us", "{{name}}"))
	})

	t.Run("No match fr", func(t *testing.T) {
		assert.Equal(t, "{{greeting}}", i18n.Translate(assistantID, "fr", "{{greeting}}"))
	})

	t.Run("Non-existent assistant", func(t *testing.T) {
		assert.Equal(t, "{{greeting}}", i18n.Translate("nonexistent", "en", "{{greeting}}"))
	})

	t.Run("Fallback to global", func(t *testing.T) {
		origGlobal := i18n.Locales["__global__"]
		t.Cleanup(func() { i18n.Locales["__global__"] = origGlobal })

		i18n.Locales["__global__"] = map[string]i18n.I18n{
			"es": {
				Locale:   "es",
				Messages: map[string]any{"greeting": "Hola"},
			},
		}
		assert.Equal(t, "Hola", i18n.Translate(assistantID, "es", "{{greeting}}"))
	})

	t.Run("Complex structure", func(t *testing.T) {
		input := map[string]any{
			"title": "{{greeting}}",
			"user":  "{{name}}",
		}
		result, ok := i18n.Translate(assistantID, "zh-cn", input).(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "你好", result["title"])
		assert.Equal(t, "张三", result["user"])
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("Empty Messages map", func(t *testing.T) {
		inst := i18n.I18n{Locale: "en", Messages: map[string]any{}}
		assert.Equal(t, "{{key}}", inst.Parse("{{key}}"))
	})

	t.Run("Nil Messages map", func(t *testing.T) {
		inst := i18n.I18n{Locale: "en", Messages: nil}
		assert.Equal(t, "{{key}}", inst.Parse("{{key}}"))
	})

	t.Run("Empty locale string", func(t *testing.T) {
		i18n.Locales["edge-test"] = map[string]i18n.I18n{
			"en": {Locale: "en", Messages: map[string]any{"key": "value"}},
		}
		t.Cleanup(func() { delete(i18n.Locales, "edge-test") })

		result := i18n.Translate("edge-test", "", "{{key}}")
		assert.Equal(t, "{{key}}", result)
	})

	t.Run("Locale with only spaces", func(t *testing.T) {
		i18n.Locales["edge-test2"] = map[string]i18n.I18n{
			"": {Locale: "", Messages: map[string]any{"key": "value"}},
		}
		t.Cleanup(func() { delete(i18n.Locales, "edge-test2") })

		result := i18n.Translate("edge-test2", "   ", "{{key}}")
		assert.Equal(t, "value", result)
	})
}

func TestBuiltinMessages(t *testing.T) {
	origGlobal := i18n.Locales["__global__"]
	t.Cleanup(func() { i18n.Locales["__global__"] = origGlobal })

	t.Run("English built-in messages", func(t *testing.T) {
		assert.Equal(t, "{{name}}", i18n.TranslateGlobal("en", "{{assistant.agent.stream.label}}"))
		assert.Equal(t, "{{name}} is processing the request", i18n.TranslateGlobal("en", "{{assistant.agent.stream.description}}"))
		assert.Equal(t, "Get Chat History", i18n.TranslateGlobal("en", "{{assistant.agent.stream.history}}"))
		assert.Equal(t, "LLM %s", i18n.TranslateGlobal("en", "{{llm.openai.stream.label}}"))
		assert.Equal(t, "LLM Stream", i18n.TranslateGlobal("en", "{{llm.handlers.stream.info}}"))
		assert.Equal(t, "Processing", i18n.TranslateGlobal("en", "{{common.status.processing}}"))
	})

	t.Run("Chinese zh-cn built-in messages", func(t *testing.T) {
		assert.Equal(t, "{{name}}", i18n.TranslateGlobal("zh-cn", "{{assistant.agent.stream.label}}"))
		assert.Equal(t, "{{name}} 正在处理请求", i18n.TranslateGlobal("zh-cn", "{{assistant.agent.stream.description}}"))
		assert.Equal(t, "获取聊天历史", i18n.TranslateGlobal("zh-cn", "{{assistant.agent.stream.history}}"))
		assert.Equal(t, "LLM 流式输出", i18n.TranslateGlobal("zh-cn", "{{llm.handlers.stream.info}}"))
		assert.Equal(t, "处理中", i18n.TranslateGlobal("zh-cn", "{{common.status.processing}}"))
	})

	t.Run("Chinese zh short code", func(t *testing.T) {
		assert.Equal(t, "{{name}}", i18n.TranslateGlobal("zh", "{{assistant.agent.stream.label}}"))
		assert.Equal(t, "处理中", i18n.TranslateGlobal("zh", "{{common.status.processing}}"))
	})

	t.Run("Embedded template", func(t *testing.T) {
		assert.Equal(t, "Status: Processing", i18n.TranslateGlobal("en", "Status: {{common.status.processing}}"))
		assert.Equal(t, "状态: 处理中", i18n.TranslateGlobal("zh-cn", "状态: {{common.status.processing}}"))
	})

	t.Run("Non-existent key", func(t *testing.T) {
		assert.Equal(t, "{{unknown.key}}", i18n.TranslateGlobal("en", "{{unknown.key}}"))
	})
}

func TestTr(t *testing.T) {
	origGlobal := i18n.Locales["__global__"]
	t.Cleanup(func() { i18n.Locales["__global__"] = origGlobal })

	i18n.Locales["__global__"] = map[string]i18n.I18n{
		"en": {
			Locale: "en",
			Messages: map[string]any{
				"assistant.label":       "Assistant {{assistant.name}}",
				"assistant.name":        "AI Helper",
				"assistant.description": "{{assistant.label}} is processing",
				"llm.label":             "LLM {{model.deepseek}}",
				"model.deepseek":        "DeepSeek",
				"deeply.nested":         "Level1 {{level2}}",
				"level2":                "Level2 {{level3}}",
				"level3":                "Level3 End",
				"simple.message":        "Hello World",
			},
		},
		"zh-cn": {
			Locale: "zh-cn",
			Messages: map[string]any{
				"assistant.label":       "助手 {{assistant.name}}",
				"assistant.name":        "智能助手",
				"assistant.description": "{{assistant.label}} 正在处理",
				"llm.label":             "模型 {{model.deepseek}}",
				"model.deepseek":        "深度求索",
				"deeply.nested":         "第一层 {{level2}}",
				"level2":                "第二层 {{level3}}",
				"level3":                "第三层结束",
				"simple.message":        "你好世界",
			},
		},
	}

	i18n.Locales["test-assistant"] = map[string]i18n.I18n{
		"en": {
			Locale:   "en",
			Messages: map[string]any{"assistant.name": "Custom Assistant"},
		},
	}
	t.Cleanup(func() { delete(i18n.Locales, "test-assistant") })

	t.Run("Simple translation", func(t *testing.T) {
		assert.Equal(t, "Hello World", i18n.Tr("__global__", "en", "simple.message"))
		assert.Equal(t, "你好世界", i18n.Tr("__global__", "zh-cn", "simple.message"))
	})

	t.Run("One level nested", func(t *testing.T) {
		assert.Equal(t, "Assistant AI Helper", i18n.Tr("__global__", "en", "assistant.label"))
		assert.Equal(t, "助手 智能助手", i18n.Tr("__global__", "zh-cn", "assistant.label"))
	})

	t.Run("Two levels nested", func(t *testing.T) {
		assert.Equal(t, "Assistant AI Helper is processing", i18n.Tr("__global__", "en", "assistant.description"))
		assert.Equal(t, "助手 智能助手 正在处理", i18n.Tr("__global__", "zh-cn", "assistant.description"))
	})

	t.Run("Three levels deeply nested", func(t *testing.T) {
		assert.Equal(t, "Level1 Level2 Level3 End", i18n.Tr("__global__", "en", "deeply.nested"))
		assert.Equal(t, "第一层 第二层 第三层结束", i18n.Tr("__global__", "zh-cn", "deeply.nested"))
	})

	t.Run("Assistant-specific override", func(t *testing.T) {
		assert.Equal(t, "Assistant Custom Assistant", i18n.Tr("test-assistant", "en", "assistant.label"))
		assert.Equal(t, "Hello World", i18n.Tr("test-assistant", "en", "simple.message"))
		assert.Equal(t, "Custom Assistant", i18n.Tr("test-assistant", "en", "assistant.name"))
	})

	t.Run("Non-existent key returns original", func(t *testing.T) {
		assert.Equal(t, "non.existent.key", i18n.Tr("__global__", "en", "non.existent.key"))
	})

	t.Run("LLM with model variable", func(t *testing.T) {
		assert.Equal(t, "LLM DeepSeek", i18n.Tr("__global__", "en", "llm.label"))
		assert.Equal(t, "模型 深度求索", i18n.Tr("__global__", "zh-cn", "llm.label"))
	})
}

func TestTAlias(t *testing.T) {
	origGlobal := i18n.Locales["__global__"]
	t.Cleanup(func() { i18n.Locales["__global__"] = origGlobal })

	t.Run("T matches TranslateGlobal", func(t *testing.T) {
		input := "{{assistant.agent.stream.label}}"
		resultT := i18n.T("en", input)
		resultGlobal := i18n.TranslateGlobal("en", input)
		assert.Equal(t, resultGlobal, resultT)
		assert.Equal(t, "{{name}}", resultT)
	})

	t.Run("T with Chinese", func(t *testing.T) {
		assert.Equal(t, "获取聊天历史", i18n.T("zh-cn", "{{assistant.agent.stream.history}}"))
	})

	t.Run("T with embedded template", func(t *testing.T) {
		assert.Equal(t, "Status: Completed", i18n.T("en", "Status: {{common.status.completed}}"))
	})

	t.Run("T with nested template", func(t *testing.T) {
		assert.Equal(t, "{{name}}", i18n.T("en", "{{assistant.agent.stream.label}}"))
		assert.Equal(t, "{{name}}", i18n.T("zh-cn", "{{assistant.agent.stream.label}}"))
	})
}
