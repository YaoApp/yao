//go:build unit

package hook_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/assistant/hook"
	agentContext "github.com/yaoapp/yao/agent/context"
)

func TestGetHookCreateResponse(t *testing.T) {
	s := &hook.Script{}

	t.Run("NilInput", func(t *testing.T) {
		res, err := hook.ExportGetHookCreateResponse(s, nil)
		require.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("UndefinedInput", func(t *testing.T) {
		res, err := hook.ExportGetHookCreateResponse(s, bridge.UndefinedT(0))
		require.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("ValidMap", func(t *testing.T) {
		input := map[string]interface{}{
			"locale":    "zh-cn",
			"theme":     "dark",
			"connector": "test-connector",
			"messages": []interface{}{
				map[string]interface{}{"role": "user", "content": "hello"},
			},
		}
		res, err := hook.ExportGetHookCreateResponse(s, input)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "zh-cn", res.Locale)
		assert.Equal(t, "dark", res.Theme)
		assert.Equal(t, "test-connector", res.Connector)
		assert.Len(t, res.Messages, 1)
	})

	t.Run("EmptyMap", func(t *testing.T) {
		res, err := hook.ExportGetHookCreateResponse(s, map[string]interface{}{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Locale)
		assert.Empty(t, res.Messages)
	})
}

func TestGetNextHookResponse(t *testing.T) {
	s := &hook.Script{}

	t.Run("NilInput", func(t *testing.T) {
		res, err := hook.ExportGetNextHookResponse(s, nil)
		require.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("UndefinedInput", func(t *testing.T) {
		res, err := hook.ExportGetNextHookResponse(s, bridge.UndefinedT(0))
		require.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("ValidDelegateMap", func(t *testing.T) {
		input := map[string]interface{}{
			"delegate": map[string]interface{}{
				"agent_id": "tests.hook-echo",
				"messages": []interface{}{
					map[string]interface{}{"role": "user", "content": "delegated"},
				},
			},
		}
		res, err := hook.ExportGetNextHookResponse(s, input)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Delegate)
		assert.Equal(t, "tests.hook-echo", res.Delegate.AgentID)
	})

	t.Run("ValidDataMap", func(t *testing.T) {
		input := map[string]interface{}{
			"data": map[string]interface{}{"key": "value"},
		}
		res, err := hook.ExportGetNextHookResponse(s, input)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotNil(t, res.Data)
		assert.Nil(t, res.Delegate)
	})

	t.Run("EmptyMap", func(t *testing.T) {
		res, err := hook.ExportGetNextHookResponse(s, map[string]interface{}{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Nil(t, res.Delegate)
		assert.Nil(t, res.Data)
	})
}

func TestApplyContextAdjustments(t *testing.T) {
	s := &hook.Script{}

	t.Run("OverrideLocaleThemeRoute", func(t *testing.T) {
		ctx := newTestContext("test-chat", "test-assistant")
		response := &agentContext.HookCreateResponse{
			Locale: "zh-cn",
			Theme:  "dark",
			Route:  "/new/route",
		}
		hook.ExportApplyContextAdjustments(s, ctx, response)
		assert.Equal(t, "zh-cn", ctx.Locale)
		assert.Equal(t, "dark", ctx.Theme)
		assert.Equal(t, "/new/route", ctx.Route)
	})

	t.Run("EmptyFieldsNoOverride", func(t *testing.T) {
		ctx := newTestContext("test-chat", "test-assistant")
		response := &agentContext.HookCreateResponse{}
		hook.ExportApplyContextAdjustments(s, ctx, response)
		assert.Equal(t, "en-us", ctx.Locale)
		assert.Equal(t, "light", ctx.Theme)
		assert.Equal(t, "", ctx.Route)
	})

	t.Run("MetadataMerge", func(t *testing.T) {
		ctx := newTestContext("test-chat", "test-assistant")
		ctx.Metadata["existing"] = "keep"
		response := &agentContext.HookCreateResponse{
			Metadata: map[string]interface{}{
				"new_key":  "new_value",
				"existing": "overwritten",
			},
		}
		hook.ExportApplyContextAdjustments(s, ctx, response)
		assert.Equal(t, "new_value", ctx.Metadata["new_key"])
		assert.Equal(t, "overwritten", ctx.Metadata["existing"])
	})

	t.Run("MetadataMergeNilInit", func(t *testing.T) {
		ctx := newTestContext("test-chat", "test-assistant")
		ctx.Metadata = nil
		response := &agentContext.HookCreateResponse{
			Metadata: map[string]interface{}{"key": "value"},
		}
		hook.ExportApplyContextAdjustments(s, ctx, response)
		require.NotNil(t, ctx.Metadata)
		assert.Equal(t, "value", ctx.Metadata["key"])
	})
}

func TestApplyOptionsAdjustments(t *testing.T) {
	s := &hook.Script{}

	t.Run("OverrideConnector", func(t *testing.T) {
		opts := &agentContext.Options{}
		response := &agentContext.HookCreateResponse{Connector: "new-connector"}
		hook.ExportApplyOptionsAdjustments(s, opts, response)
		assert.Equal(t, "new-connector", opts.Connector)
	})

	t.Run("EmptyConnectorNoOverride", func(t *testing.T) {
		opts := &agentContext.Options{Connector: "original"}
		response := &agentContext.HookCreateResponse{}
		hook.ExportApplyOptionsAdjustments(s, opts, response)
		assert.Equal(t, "original", opts.Connector)
	})
}
