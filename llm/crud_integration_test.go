//go:build integration

package llm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/llm"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// ---------------------------------------------------------------------------
// Provider CRUD
// ---------------------------------------------------------------------------

func TestCreateProvider(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{
		ProviderID: "test-create",
		Name:       "Test Create Provider",
		API:        llm.APIOpenAICompletions,
		BaseURL:    "https://api.test.com/v1",
		APIKey:     "sk-test-create",
		Enabled:    true,
		Scope:      llm.ScopeSystem,
	}
	id, err := llm.CreateProvider(p)
	require.NoError(t, err)
	assert.Greater(t, id, 0)
	assert.Equal(t, id, p.ID)

	t.Cleanup(func() { _ = llm.DeleteProvider(id) })

	got, err := llm.GetProvider(id)
	require.NoError(t, err)
	assert.Equal(t, llm.ProviderID("test-create"), got.ProviderID)
	assert.Equal(t, "Test Create Provider", got.Name)
	assert.Equal(t, llm.APIOpenAICompletions, got.API)
	assert.Equal(t, "https://api.test.com/v1", got.BaseURL)
	assert.True(t, got.Enabled)
}

func TestGetProvider_NotFound(t *testing.T) {
	testprepare.PrepareSandbox(t)
	_, err := llm.GetProvider(99999)
	assert.Error(t, err)
}

func TestUpdateProvider(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{
		ProviderID: "test-update",
		Name:       "Before Update",
		API:        llm.APIOpenAICompletions,
		Enabled:    true,
		Scope:      llm.ScopeSystem,
	}
	id, err := llm.CreateProvider(p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteProvider(id) })

	p.Name = "After Update"
	p.BaseURL = "https://updated.test.com/v1"
	err = llm.UpdateProvider(id, p)
	require.NoError(t, err)

	got, err := llm.GetProvider(id)
	require.NoError(t, err)
	assert.Equal(t, "After Update", got.Name)
	assert.Equal(t, "https://updated.test.com/v1", got.BaseURL)
}

func TestDeleteProvider(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{
		ProviderID: "test-delete",
		Name:       "To Delete",
		API:        llm.APIOpenAICompletions,
		Scope:      llm.ScopeSystem,
	}
	id, err := llm.CreateProvider(p)
	require.NoError(t, err)

	err = llm.DeleteProvider(id)
	require.NoError(t, err)

	_, err = llm.GetProvider(id)
	assert.Error(t, err)
}

func TestListProviders(t *testing.T) {
	testprepare.PrepareSandbox(t)

	p1 := &llm.Provider{ProviderID: "test-list-1", Name: "List 1", API: llm.APIOpenAICompletions, Enabled: true, Scope: llm.ScopeSystem}
	p2 := &llm.Provider{ProviderID: "test-list-2", Name: "List 2", API: llm.APIAnthropicMessages, Enabled: true, Scope: llm.ScopeSystem}
	id1, err := llm.CreateProvider(p1)
	require.NoError(t, err)
	id2, err := llm.CreateProvider(p2)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = llm.DeleteProvider(id1)
		_ = llm.DeleteProvider(id2)
	})

	providers, err := llm.ListProviders(llm.ScopeSystem, "", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(providers), 2)

	found := map[llm.ProviderID]bool{}
	for _, p := range providers {
		found[p.ProviderID] = true
	}
	assert.True(t, found["test-list-1"])
	assert.True(t, found["test-list-2"])
}

func TestFindProviderByKey(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{ProviderID: "test-findkey", Name: "Find By Key", API: llm.APIOpenAICompletions, Enabled: true, Scope: llm.ScopeSystem}
	id, err := llm.CreateProvider(p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteProvider(id) })

	// No explicit Reload() needed — CreateProvider refreshes the cache automatically.
	found, ok := llm.Global.FindProviderByKey("test-findkey")
	require.True(t, ok, "FindProviderByKey should find provider after CreateProvider auto-refresh")
	assert.Equal(t, id, found.ID)
}

// ---------------------------------------------------------------------------
// Model CRUD
// ---------------------------------------------------------------------------

func TestCreateModel(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{ProviderID: "test-model-prov", Name: "Model Provider", API: llm.APIOpenAICompletions, Scope: llm.ScopeSystem}
	pid, err := llm.CreateProvider(p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteProvider(pid) })

	m := &llm.Model{
		ProviderRef:     pid,
		ModelID:         "test-model-1",
		Name:            "Test Model 1",
		Type:            llm.ModelTypeChat,
		Enabled:         true,
		ContextWindow:   128000,
		MaxOutputTokens: 4096,
		SupportsToolUse: true,
	}
	mid, err := llm.CreateModel(m)
	require.NoError(t, err)
	assert.Greater(t, mid, 0)
	t.Cleanup(func() { _ = llm.DeleteModel(mid) })

	got, err := llm.GetModel(mid)
	require.NoError(t, err)
	assert.Equal(t, "test-model-1", got.ModelID)
	assert.Equal(t, "Test Model 1", got.Name)
	assert.Equal(t, llm.ModelTypeChat, got.Type)
	assert.Equal(t, 128000, got.ContextWindow)
	assert.True(t, got.SupportsToolUse)
}

func TestUpdateModel(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{ProviderID: "test-model-upd", Name: "Model Upd Provider", API: llm.APIOpenAICompletions, Scope: llm.ScopeSystem}
	pid, err := llm.CreateProvider(p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteProvider(pid) })

	m := &llm.Model{ProviderRef: pid, ModelID: "upd-model", Name: "Before", Type: llm.ModelTypeChat, Enabled: true}
	mid, err := llm.CreateModel(m)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteModel(mid) })

	m.Name = "After"
	m.ContextWindow = 200000
	err = llm.UpdateModel(mid, m)
	require.NoError(t, err)

	got, err := llm.GetModel(mid)
	require.NoError(t, err)
	assert.Equal(t, "After", got.Name)
	assert.Equal(t, 200000, got.ContextWindow)
}

func TestDeleteModel(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{ProviderID: "test-model-del", Name: "Model Del Provider", API: llm.APIOpenAICompletions, Scope: llm.ScopeSystem}
	pid, err := llm.CreateProvider(p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteProvider(pid) })

	m := &llm.Model{ProviderRef: pid, ModelID: "del-model", Name: "To Delete", Type: llm.ModelTypeChat}
	mid, err := llm.CreateModel(m)
	require.NoError(t, err)

	err = llm.DeleteModel(mid)
	require.NoError(t, err)

	_, err = llm.GetModel(mid)
	assert.Error(t, err, "deleted model should not be found")
}

func TestListModels(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{ProviderID: "test-model-lst", Name: "Model List Provider", API: llm.APIOpenAICompletions, Scope: llm.ScopeSystem}
	pid, err := llm.CreateProvider(p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteProvider(pid) })

	m1 := &llm.Model{ProviderRef: pid, ModelID: "lst-1", Name: "List Model 1", Type: llm.ModelTypeChat, Enabled: true}
	m2 := &llm.Model{ProviderRef: pid, ModelID: "lst-2", Name: "List Model 2", Type: llm.ModelTypeChat, Enabled: false}
	id1, _ := llm.CreateModel(m1)
	id2, _ := llm.CreateModel(m2)
	t.Cleanup(func() {
		_ = llm.DeleteModel(id1)
		_ = llm.DeleteModel(id2)
	})

	all, err := llm.ListModels(pid)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	enabled, err := llm.ListEnabledModels(pid)
	require.NoError(t, err)
	assert.Len(t, enabled, 1)
	assert.Equal(t, "lst-1", enabled[0].ModelID)
}

func TestFindModelByModelID(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{ProviderID: "test-model-find", Name: "Model Find Provider", API: llm.APIOpenAICompletions, Scope: llm.ScopeSystem}
	pid, err := llm.CreateProvider(p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteProvider(pid) })

	m := &llm.Model{ProviderRef: pid, ModelID: "find-me", Name: "Find Me", Type: llm.ModelTypeChat, Enabled: true}
	mid, err := llm.CreateModel(m)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteModel(mid) })

	found, err := llm.FindModelByModelID(pid, "find-me")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, mid, found.ID)

	notFound, err := llm.FindModelByModelID(pid, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

// ---------------------------------------------------------------------------
// Role Binding CRUD + Cascade
// ---------------------------------------------------------------------------

func TestSetRoleBinding_CreateAndUpsert(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{ProviderID: "test-role-prov", Name: "Role Provider", API: llm.APIOpenAICompletions, Scope: llm.ScopeSystem}
	pid, err := llm.CreateProvider(p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteProvider(pid) })

	m := &llm.Model{ProviderRef: pid, ModelID: "role-model", Name: "Role Model", Type: llm.ModelTypeChat, Enabled: true}
	mid, err := llm.CreateModel(m)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteModel(mid) })

	rb := &llm.RoleBinding{
		Role:        "default",
		ProviderRef: pid,
		ModelID:     "role-model",
		Scope:       llm.ScopeSystem,
	}
	id1, err := llm.SetRoleBinding(rb)
	require.NoError(t, err)
	assert.Greater(t, id1, 0)
	t.Cleanup(func() { _ = llm.DeleteRoleBinding(id1) })

	rb2 := &llm.RoleBinding{
		Role:        "default",
		ProviderRef: pid,
		ModelID:     "role-model",
		Scope:       llm.ScopeSystem,
	}
	id2, err := llm.SetRoleBinding(rb2)
	require.NoError(t, err)
	assert.Equal(t, id1, id2, "upsert should return the same ID")
}

func TestResolveRole_ScopeCascade(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{ProviderID: "test-cascade-prov", Name: "Cascade Provider", API: llm.APIOpenAICompletions, Scope: llm.ScopeSystem}
	pid, err := llm.CreateProvider(p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteProvider(pid) })

	m1 := &llm.Model{ProviderRef: pid, ModelID: "sys-model", Name: "System Model", Type: llm.ModelTypeChat, Enabled: true}
	m2 := &llm.Model{ProviderRef: pid, ModelID: "team-model", Name: "Team Model", Type: llm.ModelTypeChat, Enabled: true}
	m3 := &llm.Model{ProviderRef: pid, ModelID: "user-model", Name: "User Model", Type: llm.ModelTypeChat, Enabled: true}
	mid1, _ := llm.CreateModel(m1)
	mid2, _ := llm.CreateModel(m2)
	mid3, _ := llm.CreateModel(m3)
	t.Cleanup(func() {
		_ = llm.DeleteModel(mid1)
		_ = llm.DeleteModel(mid2)
		_ = llm.DeleteModel(mid3)
	})

	sysRB := &llm.RoleBinding{Role: "heavy", ProviderRef: pid, ModelID: "sys-model", Scope: llm.ScopeSystem}
	sysID, err := llm.SetRoleBinding(sysRB)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteRoleBinding(sysID) })

	rb, err := llm.ResolveRole("heavy", "team-1", "user-1")
	require.NoError(t, err)
	require.NotNil(t, rb)
	assert.Equal(t, "sys-model", rb.ModelID, "should fall back to system scope")

	teamRB := &llm.RoleBinding{Role: "heavy", ProviderRef: pid, ModelID: "team-model", Scope: llm.ScopeTeam, TeamID: "team-1"}
	teamID, err := llm.SetRoleBinding(teamRB)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteRoleBinding(teamID) })

	rb, err = llm.ResolveRole("heavy", "team-1", "user-1")
	require.NoError(t, err)
	require.NotNil(t, rb)
	assert.Equal(t, "team-model", rb.ModelID, "team scope should override system")

	userRB := &llm.RoleBinding{Role: "heavy", ProviderRef: pid, ModelID: "user-model", Scope: llm.ScopeUser, UserID: "user-1"}
	userID, err := llm.SetRoleBinding(userRB)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteRoleBinding(userID) })

	rb, err = llm.ResolveRole("heavy", "team-1", "user-1")
	require.NoError(t, err)
	require.NotNil(t, rb)
	assert.Equal(t, "user-model", rb.ModelID, "user scope should override team")
}

func TestResolveRole_NoBinding(t *testing.T) {
	testprepare.PrepareSandbox(t)
	rb, err := llm.ResolveRole("nonexistent-role", "", "")
	require.NoError(t, err)
	assert.Nil(t, rb)
}

func TestGetRoleBindings(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{ProviderID: "test-getrb-prov", Name: "GetRB Provider", API: llm.APIOpenAICompletions, Scope: llm.ScopeSystem}
	pid, err := llm.CreateProvider(p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteProvider(pid) })

	rb1 := &llm.RoleBinding{Role: "default", ProviderRef: pid, ModelID: "m1", Scope: llm.ScopeSystem}
	rb2 := &llm.RoleBinding{Role: "heavy", ProviderRef: pid, ModelID: "m2", Scope: llm.ScopeSystem}
	id1, _ := llm.SetRoleBinding(rb1)
	id2, _ := llm.SetRoleBinding(rb2)
	t.Cleanup(func() {
		_ = llm.DeleteRoleBinding(id1)
		_ = llm.DeleteRoleBinding(id2)
	})

	bindings, err := llm.GetRoleBindings(llm.ScopeSystem, "", "")
	require.NoError(t, err)
	found := map[string]bool{}
	for _, b := range bindings {
		if b.ProviderRef == pid {
			found[b.Role] = true
		}
	}
	assert.True(t, found["default"])
	assert.True(t, found["heavy"])
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

func TestRegistryReload(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{ProviderID: "test-reload", Name: "Reload Provider", API: llm.APIOpenAICompletions, Enabled: true, Scope: llm.ScopeSystem}
	id, err := llm.CreateProvider(p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteProvider(id) })

	err = llm.Global.Reload()
	require.NoError(t, err)

	found, ok := llm.Global.FindProvider(id)
	assert.True(t, ok)
	assert.Equal(t, llm.ProviderID("test-reload"), found.ProviderID)
}

func TestResolveRoleModel(t *testing.T) {
	testprepare.PrepareSandbox(t)
	p := &llm.Provider{ProviderID: "test-resolve-prov", Name: "Resolve Provider", API: llm.APIOpenAICompletions, Enabled: true, Scope: llm.ScopeSystem}
	pid, err := llm.CreateProvider(p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteProvider(pid) })

	m := &llm.Model{ProviderRef: pid, ModelID: "resolve-model", Name: "Resolve Model", Type: llm.ModelTypeChat, Enabled: true}
	mid, err := llm.CreateModel(m)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteModel(mid) })

	rb := &llm.RoleBinding{Role: "decision", ProviderRef: pid, ModelID: "resolve-model", Scope: llm.ScopeSystem}
	rbid, err := llm.SetRoleBinding(rb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = llm.DeleteRoleBinding(rbid) })

	err = llm.Global.Reload()
	require.NoError(t, err)

	prov, model, err := llm.Global.ResolveRoleModel("decision", "", "")
	require.NoError(t, err)
	assert.Equal(t, llm.ProviderID("test-resolve-prov"), prov.ProviderID)
	assert.Equal(t, "resolve-model", model.ModelID)
}
