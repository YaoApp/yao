//go:build unit

package task_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/task"
)

// --- configReqToMap tests ---

func TestConfigReqToMap_AllFieldsSet(t *testing.T) {
	runner := "openai"
	model := "gpt-4o"
	image := "node:20"
	secretVal := "secret-val"

	req := &task.ConfigReq{
		Runner:   &runner,
		Model:    &model,
		Image:    &image,
		Secrets:  map[string]*string{"API_KEY": &secretVal},
		Services: []task.ServiceDecl{{Name: "web", Port: 3000, Protocol: "http", Public: true}},
		Skills:   []string{"search"},
		Schedule: &task.ScheduleConfig{Enabled: true, Mode: "interval"},
	}

	data := task.ExportConfigReqToMap(req)

	require.Equal(t, []string{"openai"}, data["runners"])
	assert.Equal(t, "gpt-4o", data["model"])
	assert.Equal(t, "node:20", data["image"])
	assert.NotNil(t, data["secrets"])
	assert.NotNil(t, data["services"])
	assert.NotNil(t, data["skills"])
	assert.NotNil(t, data["schedule"])
}

func TestConfigReqToMap_NilFieldsOmitted(t *testing.T) {
	req := &task.ConfigReq{}
	data := task.ExportConfigReqToMap(req)

	assert.Empty(t, data, "nil fields should not appear in map")
}

func TestConfigReqToMap_NullSecretValue(t *testing.T) {
	req := &task.ConfigReq{
		Secrets: map[string]*string{"DELETE_ME": nil},
	}

	data := task.ExportConfigReqToMap(req)
	secrets := data["secrets"].(map[string]interface{})
	assert.Nil(t, secrets["DELETE_ME"], "nil pointer should map to nil (null)")
}

func TestConfigReqToMap_PartialUpdate(t *testing.T) {
	model := "claude-sonnet"
	req := &task.ConfigReq{
		Model: &model,
	}

	data := task.ExportConfigReqToMap(req)

	assert.Equal(t, "claude-sonnet", data["model"])
	assert.NotContains(t, data, "runners")
	assert.NotContains(t, data, "image")
	assert.NotContains(t, data, "secrets")
}

func TestConfigReqToMap_SecretsNestedFormat(t *testing.T) {
	val := "my-key"
	req := &task.ConfigReq{
		Secrets: map[string]*string{"API_KEY": &val},
	}

	data := task.ExportConfigReqToMap(req)
	secrets := data["secrets"].(map[string]interface{})
	entry := secrets["API_KEY"].(map[string]interface{})
	assert.Equal(t, "my-key", entry["value"])
}

func TestConfigReqToMap_TimeoutAndMaxTurns(t *testing.T) {
	timeout := "45m"
	maxTurns := 200
	req := &task.ConfigReq{
		Timeout:  &timeout,
		MaxTurns: &maxTurns,
	}
	m := task.ExportConfigReqToMap(req)
	assert.Equal(t, "45m", m["timeout"])
	assert.Equal(t, 200, m["max_turns"])
}
