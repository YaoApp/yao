//go:build integration

package config_test

import (
	stdctx "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/config"
	agentctx "github.com/yaoapp/yao/agent/context"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestResolve_DSLOnly(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	config.LoadAssistantFunc = func(id string) (*config.AssistantDefaults, error) {
		return &config.AssistantDefaults{
			Connector: "openai.mock",
			Runner:    "claude-code",
			Image:     "yao/sandbox:latest",
			MaxTurns:  50,
			Secrets:   map[string]string{"API_KEY": ""},
			Skills:    []string{"web-search"},
		}, nil
	}
	t.Cleanup(func() { config.LoadAssistantFunc = nil })

	opts := config.ResolveOptions{
		AssistantID: "tests.config-dsl",
		UserID:      identity.AlphaOwnerUserID,
		TeamID:      identity.AlphaTeamID,
	}
	resolved, err := config.Resolve(opts)
	require.NoError(t, err)

	assert.Equal(t, "claude-code", resolved.Runner)
	assert.Equal(t, "openai.mock", resolved.Model)
	assert.Equal(t, "yao/sandbox:latest", resolved.Image)
	assert.Equal(t, 50, resolved.MaxTurns)
	assert.Equal(t, map[string]string{"API_KEY": ""}, resolved.Secrets)
	assert.Equal(t, []string{"web-search"}, resolved.Skills)
	assert.Equal(t, "dsl", resolved.ResolvedFrom["runner"])
}

func TestResolve_UserOverridesDSL(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	config.LoadAssistantFunc = func(id string) (*config.AssistantDefaults, error) {
		return &config.AssistantDefaults{
			Connector: "openai.mock",
			Runner:    "claude-code",
			Image:     "yao/sandbox:latest",
		}, nil
	}
	t.Cleanup(func() { config.LoadAssistantFunc = nil })

	reg := setting.Global
	require.NotNil(t, reg, "setting.Global must be initialized by PrepareSandbox")

	assistantID := "tests.config-override"
	userScope := setting.ScopeID{Scope: setting.ScopeUser, UserID: identity.AlphaOwnerUserID}
	_, err := reg.Set(userScope, "agent."+assistantID, map[string]interface{}{
		"runners":   []interface{}{"open-code"},
		"image":     "custom/image:v2",
		"max_turns": float64(100),
	})
	require.NoError(t, err)
	t.Cleanup(func() { reg.Delete(userScope, "agent."+assistantID) })

	opts := config.ResolveOptions{
		AssistantID: assistantID,
		UserID:      identity.AlphaOwnerUserID,
		TeamID:      identity.AlphaTeamID,
	}
	resolved, err := config.Resolve(opts)
	require.NoError(t, err)

	assert.Equal(t, "open-code", resolved.Runner)
	assert.Equal(t, "custom/image:v2", resolved.Image)
	assert.Equal(t, 100, resolved.MaxTurns)
	assert.Equal(t, "user", resolved.ResolvedFrom["runner"])
	assert.Equal(t, "openai.mock", resolved.Model, "model not overridden by registry")
}

func TestResolve_UserAndTaskLayers(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	config.LoadAssistantFunc = func(id string) (*config.AssistantDefaults, error) {
		return &config.AssistantDefaults{
			Connector: "openai.mock",
			Runner:    "claude-code",
		}, nil
	}
	t.Cleanup(func() { config.LoadAssistantFunc = nil })

	reg := setting.Global
	require.NotNil(t, reg)

	userScope := setting.ScopeID{Scope: setting.ScopeUser, UserID: identity.AlphaOwnerUserID}
	assistantID := "tests.config-layers"
	chatID := "chat-123-test"

	// L2: user preferences via agent.{id}
	_, err := reg.Set(userScope, "agent."+assistantID, map[string]interface{}{
		"runners": []interface{}{"user-runner"},
		"timeout": "30m",
		"secrets": map[string]interface{}{
			"DEPLOY_KEY": map[string]interface{}{"value": "deploy-val"},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { reg.Delete(userScope, "agent."+assistantID) })

	// L3: task-level override via task-config.task.{chatID}
	_, err = reg.Set(userScope, "task-config.task."+chatID, map[string]interface{}{
		"runners":   []interface{}{"task-runner"},
		"max_turns": float64(200),
	})
	require.NoError(t, err)
	t.Cleanup(func() { reg.Delete(userScope, "task-config.task."+chatID) })

	opts := config.ResolveOptions{
		AssistantID: assistantID,
		ChatID:      chatID,
		UserID:      identity.AlphaOwnerUserID,
		TeamID:      identity.AlphaTeamID,
	}
	resolved, err := config.Resolve(opts)
	require.NoError(t, err)

	assert.Equal(t, "task-runner", resolved.Runner, "task layer wins")
	assert.Equal(t, "30m", resolved.Timeout, "timeout from user layer")
	assert.Equal(t, 200, resolved.MaxTurns, "max_turns from task layer")
	assert.Equal(t, "deploy-val", resolved.Secrets["DEPLOY_KEY"], "secrets from user layer")
	assert.Equal(t, "task", resolved.ResolvedFrom["runner"])
	assert.Equal(t, "user", resolved.ResolvedFrom["timeout"])
	assert.Equal(t, "task", resolved.ResolvedFrom["max_turns"])
	assert.Equal(t, "user", resolved.ResolvedFrom["secrets"])
}

func TestResolve_NoChatID_SkipsTaskLayer(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	config.LoadAssistantFunc = func(id string) (*config.AssistantDefaults, error) {
		return &config.AssistantDefaults{
			Connector: "openai.mock",
			Runner:    "claude-code",
		}, nil
	}
	t.Cleanup(func() { config.LoadAssistantFunc = nil })

	reg := setting.Global
	require.NotNil(t, reg)

	userScope := setting.ScopeID{Scope: setting.ScopeUser, UserID: identity.AlphaOwnerUserID}
	assistantID := "tests.config-nochat"
	chatID := "chat-nochat-test"

	// L2: user preferences
	_, err := reg.Set(userScope, "agent."+assistantID, map[string]interface{}{
		"runners": []interface{}{"user-runner"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { reg.Delete(userScope, "agent."+assistantID) })

	// L3: task override (should NOT apply without ChatID)
	_, err = reg.Set(userScope, "task-config.task."+chatID, map[string]interface{}{
		"runners": []interface{}{"task-runner-should-not-apply"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { reg.Delete(userScope, "task-config.task."+chatID) })

	opts := config.ResolveOptions{
		AssistantID: assistantID,
		UserID:      identity.AlphaOwnerUserID,
		TeamID:      identity.AlphaTeamID,
		// ChatID intentionally empty → L3 skipped
	}
	resolved, err := config.Resolve(opts)
	require.NoError(t, err)

	assert.Equal(t, "user-runner", resolved.Runner, "user layer applies without ChatID")
	assert.Equal(t, "user", resolved.ResolvedFrom["runner"])
}

func TestResolve_MissingAssistantID(t *testing.T) {
	testprepare.PrepareSandbox(t)

	_, err := config.Resolve(config.ResolveOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AssistantID is required")
}

func TestGet_FromContext(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	config.LoadAssistantFunc = func(id string) (*config.AssistantDefaults, error) {
		return &config.AssistantDefaults{
			Connector: "openai.mock",
			Runner:    "claude-code",
			Image:     "yao/sandbox:latest",
			MaxTurns:  50,
			Skills:    []string{"web-search", "deploy"},
		}, nil
	}
	t.Cleanup(func() { config.LoadAssistantFunc = nil })

	reg := setting.Global
	require.NotNil(t, reg)

	userScope := setting.ScopeID{Scope: setting.ScopeUser, UserID: identity.AlphaOwnerUserID}
	assistantID := "tests.config-get-ctx"
	chatID := "chat-get-ctx-test"

	_, err := reg.Set(userScope, "task-config.task."+chatID, map[string]interface{}{
		"runners":   []interface{}{"task-override-runner"},
		"max_turns": float64(999),
	})
	require.NoError(t, err)
	t.Cleanup(func() { reg.Delete(userScope, "task-config.task."+chatID) })

	ctx := agentctx.New(
		stdctx.Background(),
		&oauthtypes.AuthorizedInfo{
			UserID: identity.AlphaOwnerUserID,
			TeamID: identity.AlphaTeamID,
		},
		chatID,
	)
	ctx.AssistantID = assistantID

	resolved, err := config.Get(ctx)
	require.NoError(t, err)

	assert.Equal(t, "task-override-runner", resolved.Runner)
	assert.Equal(t, 999, resolved.MaxTurns)
	assert.Equal(t, "openai.mock", resolved.Model)
	assert.Equal(t, "yao/sandbox:latest", resolved.Image)
	assert.Equal(t, []string{"web-search", "deploy"}, resolved.Skills)
	assert.Equal(t, "task", resolved.ResolvedFrom["runner"])
	assert.Equal(t, "dsl", resolved.ResolvedFrom["model"])
}

func TestGet_FromContext_NoAuth(t *testing.T) {
	testprepare.PrepareSandbox(t)

	config.LoadAssistantFunc = func(id string) (*config.AssistantDefaults, error) {
		return &config.AssistantDefaults{
			Connector: "openai.mock",
			Runner:    "default-runner",
		}, nil
	}
	t.Cleanup(func() { config.LoadAssistantFunc = nil })

	ctx := agentctx.New(stdctx.Background(), nil, "chat-no-auth")
	ctx.AssistantID = "tests.config-no-auth"

	resolved, err := config.Get(ctx)
	require.NoError(t, err)

	assert.Equal(t, "default-runner", resolved.Runner)
	assert.Equal(t, "openai.mock", resolved.Model)
	assert.Equal(t, "dsl", resolved.ResolvedFrom["runner"])
}
