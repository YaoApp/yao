//go:build integration

package i18n_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	i18n "github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestTranslateGlobal(t *testing.T) {
	testprepare.PrepareSandbox(t)

	origGlobal := i18n.Locales["__global__"]
	t.Cleanup(func() {
		i18n.Locales["__global__"] = origGlobal
	})

	global := i18n.Locales["__global__"]
	if global == nil {
		global = map[string]i18n.I18n{}
		i18n.Locales["__global__"] = global
	}

	en := global["en"]
	if en.Messages == nil {
		en = i18n.I18n{Locale: "en", Messages: map[string]any{}}
	}
	en.Messages["button.ok"] = "OK"
	en.Messages["button.cancel"] = "Cancel"
	global["en"] = en

	zhcn := global["zh-cn"]
	if zhcn.Messages == nil {
		zhcn = i18n.I18n{Locale: "zh-cn", Messages: map[string]any{}}
	}
	zhcn.Messages["button.ok"] = "确定"
	zhcn.Messages["button.cancel"] = "取消"
	global["zh-cn"] = zhcn

	zh := global["zh"]
	if zh.Messages == nil {
		zh = i18n.I18n{Locale: "zh", Messages: map[string]any{}}
	}
	zh.Messages["button.ok"] = "确定"
	zh.Messages["button.cancel"] = "取消"
	global["zh"] = zh

	t.Run("TranslateGlobal with match", func(t *testing.T) {
		result := i18n.TranslateGlobal("en", "button.ok")
		assert.Equal(t, "OK", result)
	})

	t.Run("TranslateGlobal with Chinese", func(t *testing.T) {
		result := i18n.TranslateGlobal("zh-cn", "button.cancel")
		assert.Equal(t, "取消", result)
	})

	t.Run("TranslateGlobal with short code", func(t *testing.T) {
		result := i18n.TranslateGlobal("zh-TW", "button.ok")
		assert.Equal(t, "确定", result)
	})

	t.Run("TranslateGlobal without match", func(t *testing.T) {
		result := i18n.TranslateGlobal("fr", "button.ok")
		assert.Equal(t, "button.ok", result)
	})

	t.Run("TranslateGlobal no global", func(t *testing.T) {
		saved := i18n.Locales["__global__"]
		delete(i18n.Locales, "__global__")
		defer func() { i18n.Locales["__global__"] = saved }()

		result := i18n.TranslateGlobal("en", "button.ok")
		assert.Equal(t, "button.ok", result)
	})

	t.Run("TranslateGlobal fallback from en-us to en", func(t *testing.T) {
		global["en-us"] = i18n.I18n{
			Locale: "en-us",
			Messages: map[string]any{
				"button.ok":   "OK (US)",
				"app.name":    "MyApp",
				"app.version": "1.0",
			},
		}

		result := i18n.TranslateGlobal("en-us", "button.ok")
		assert.Equal(t, "OK (US)", result)

		result = i18n.TranslateGlobal("en-us", "button.cancel")
		assert.Equal(t, "Cancel", result)

		result = i18n.TranslateGlobal("en-us", "llm.handlers.stream.info")
		assert.Equal(t, "LLM Stream", result)
	})
}

func TestGetLocalesIntegration(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("Load locale files", func(t *testing.T) {
		locales, err := i18n.GetLocales("/assistants/tests/i18n-multilang")
		require.NoError(t, err)
		require.Len(t, locales, 2)

		enUS, ok := locales["en-us"]
		require.True(t, ok)
		assert.Equal(t, "Hello", enUS.Messages["greeting"])
		assert.Equal(t, "Goodbye", enUS.Messages["farewell"])

		zhCN, ok := locales["zh-cn"]
		require.True(t, ok)
		assert.Equal(t, "你好", zhCN.Messages["greeting"])
		assert.Equal(t, "再见", zhCN.Messages["farewell"])

		require.NotNil(t, enUS.Messages["chat"])
	})

	t.Run("Flatten loaded locales", func(t *testing.T) {
		locales, err := i18n.GetLocales("/assistants/tests/i18n-multilang")
		require.NoError(t, err)

		flattened := locales.Flatten()

		_, hasEn := flattened["en"]
		assert.True(t, hasEn)
		_, hasZh := flattened["zh"]
		assert.True(t, hasZh)
		_, hasUS := flattened["us"]
		assert.True(t, hasUS)
		_, hasCN := flattened["cn"]
		assert.True(t, hasCN)

		enLocale := flattened["en"]
		assert.Equal(t, "New Chat", enLocale.Messages["chat.title"])
		assert.Equal(t, "Start a new conversation", enLocale.Messages["chat.description"])
	})
}
