package llmprovider_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
)

func TestProcessCreate(t *testing.T) {
	setupRegistry(t)

	p := process.New("llmprovider.create", map[string]interface{}{
		"key":         "proc-test",
		"name":        "Proc Test",
		"type":        "openai",
		"api_url":     "https://api.openai.com",
		"api_key":     "sk-proc-test",
		"enabled":     true,
		"require_key": true,
		"models":      []interface{}{map[string]interface{}{"id": "gpt-4o", "name": "GPT-4o", "capabilities": []interface{}{"streaming"}, "enabled": true}},
		"owner":       map[string]interface{}{"type": "system"},
	})
	result, err := p.Exec()
	require.NoError(t, err)
	require.NotNil(t, result)

	m := toMapResult(t, result)
	assert.Equal(t, "proc-test", m["key"])
	assert.NotEmpty(t, m["connector_id"])
	assert.Equal(t, "dynamic", m["source"])
}

func TestProcessGet(t *testing.T) {
	setupRegistry(t)
	createViaProcess(t, "proc-get")

	// Default: masked
	p := process.New("llmprovider.get", "proc-get")
	result, err := p.Exec()
	require.NoError(t, err)

	m := toMapResult(t, result)
	assert.Equal(t, "proc-get", m["key"])
	assert.NotEqual(t, "sk-proc-test", m["api_key"], "default should be masked")

	// withKey=true: plain text
	p2 := process.New("llmprovider.get", "proc-get", true)
	result2, err := p2.Exec()
	require.NoError(t, err)

	m2 := toMapResult(t, result2)
	assert.Equal(t, "sk-proc-test", m2["api_key"], "withKey=true should return plain text")
}

func TestProcessGetMasked(t *testing.T) {
	setupRegistry(t)
	createViaProcess(t, "proc-masked")

	p := process.New("llmprovider.getmasked", "proc-masked")
	result, err := p.Exec()
	require.NoError(t, err)

	m := toMapResult(t, result)
	apiKey, _ := m["api_key"].(string)
	assert.NotEqual(t, "sk-proc-test", apiKey)
	assert.Contains(t, apiKey, "test")
}

func TestProcessUpdate(t *testing.T) {
	setupRegistry(t)
	createViaProcess(t, "proc-upd")

	p := process.New("llmprovider.update", "proc-upd", map[string]interface{}{
		"name":    "Updated Name",
		"api_url": "https://custom.openai.com",
		"enabled": true,
		"models":  []interface{}{map[string]interface{}{"id": "gpt-4o", "name": "GPT-4o", "capabilities": []interface{}{"streaming"}, "enabled": true}},
	})
	result, err := p.Exec()
	require.NoError(t, err)

	m := toMapResult(t, result)
	assert.Equal(t, "Updated Name", m["name"])
	assert.Equal(t, "https://custom.openai.com", m["api_url"])
}

func TestProcessDelete(t *testing.T) {
	setupRegistry(t)
	createViaProcess(t, "proc-del")

	p := process.New("llmprovider.delete", "proc-del")
	_, err := p.Exec()
	require.NoError(t, err)

	pGet := process.New("llmprovider.get", "proc-del")
	_, err = pGet.Exec()
	assert.Error(t, err)
}

func TestProcessList(t *testing.T) {
	setupRegistry(t)
	createViaProcess(t, "proc-list-1")
	createViaProcess(t, "proc-list-2")

	p := process.New("llmprovider.list", map[string]interface{}{
		"source": "dynamic",
	})
	result, err := p.Exec()
	require.NoError(t, err)
	require.NotNil(t, result)
	t.Logf("list result type: %T", result)
}

func TestProcessGetSetting(t *testing.T) {
	setupRegistry(t)
	createViaProcess(t, "proc-setting")

	p := process.New("llmprovider.getsetting", "proc-setting")
	result, err := p.Exec()
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestProcessGetPresets(t *testing.T) {
	p := process.New("llmprovider.getpresets")
	result, err := p.Exec()
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestProcessGetPreset(t *testing.T) {
	p := process.New("llmprovider.getpreset", "openai")
	result, err := p.Exec()
	require.NoError(t, err)
	require.NotNil(t, result)

	m := toMapResult(t, result)
	assert.Equal(t, "openai", m["key"])
}

// --- helpers ---

func createViaProcess(t *testing.T, key string) {
	t.Helper()
	p := process.New("llmprovider.create", map[string]interface{}{
		"key":         key,
		"name":        "Test " + key,
		"type":        "openai",
		"api_url":     "https://api.openai.com",
		"api_key":     "sk-proc-test",
		"enabled":     true,
		"require_key": true,
		"models":      []interface{}{map[string]interface{}{"id": "gpt-4o", "name": "GPT-4o", "capabilities": []interface{}{"streaming"}, "enabled": true}},
		"owner":       map[string]interface{}{"type": "system"},
	})
	_, err := p.Exec()
	require.NoError(t, err)
}

func toMapResult(t *testing.T, v interface{}) map[string]interface{} {
	t.Helper()
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}
