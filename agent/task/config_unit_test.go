//go:build unit

package task_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/task"
)

// --- mergeLayer tests ---

func TestMergeLayer_ScalarOverride(t *testing.T) {
	dst := &task.TaskSetting{}
	resolved := map[string]string{}

	task.ExportMergeLayer(dst, map[string]interface{}{
		"runner": "openai",
		"model":  "gpt-4o",
		"image":  "node:20",
	}, "system/team/user", resolved)

	assert.Equal(t, "openai", dst.Runner)
	assert.Equal(t, "gpt-4o", dst.Model)
	assert.Equal(t, "node:20", dst.Image)
	assert.Equal(t, "system/team/user", resolved["runner"])
	assert.Equal(t, "system/team/user", resolved["model"])
	assert.Equal(t, "system/team/user", resolved["image"])
}

func TestMergeLayer_HigherLayerOverridesLower(t *testing.T) {
	dst := &task.TaskSetting{}
	resolved := map[string]string{}

	task.ExportMergeLayer(dst, map[string]interface{}{
		"runner": "openai",
		"model":  "gpt-4o",
	}, "system/team/user", resolved)

	task.ExportMergeLayer(dst, map[string]interface{}{
		"model": "claude-sonnet",
	}, "agent", resolved)

	task.ExportMergeLayer(dst, map[string]interface{}{
		"model": "local-llama",
	}, "task", resolved)

	assert.Equal(t, "openai", dst.Runner, "runner unchanged")
	assert.Equal(t, "local-llama", dst.Model, "model overridden by highest layer")
	assert.Equal(t, "system/team/user", resolved["runner"])
	assert.Equal(t, "task", resolved["model"])
}

func TestMergeLayer_EmptyStringDoesNotOverride(t *testing.T) {
	dst := &task.TaskSetting{}
	resolved := map[string]string{}

	task.ExportMergeLayer(dst, map[string]interface{}{"runner": "openai"}, "system/team/user", resolved)
	task.ExportMergeLayer(dst, map[string]interface{}{"runner": ""}, "agent", resolved)

	assert.Equal(t, "openai", dst.Runner, "empty string should not override")
	assert.Equal(t, "system/team/user", resolved["runner"], "resolved_from unchanged")
}

func TestMergeLayer_SecretsMergeByKey(t *testing.T) {
	dst := &task.TaskSetting{}
	resolved := map[string]string{}

	task.ExportMergeLayer(dst, map[string]interface{}{
		"secrets": map[string]interface{}{"API_KEY": "key1", "DB_PASS": "pass1"},
	}, "system/team/user", resolved)

	task.ExportMergeLayer(dst, map[string]interface{}{
		"secrets": map[string]interface{}{"API_KEY": "key2", "NEW_KEY": "new"},
	}, "task", resolved)

	assert.Equal(t, "key2", dst.Secrets["API_KEY"], "API_KEY overridden")
	assert.Equal(t, "pass1", dst.Secrets["DB_PASS"], "DB_PASS preserved from lower layer")
	assert.Equal(t, "new", dst.Secrets["NEW_KEY"], "NEW_KEY added")
	assert.Equal(t, "task", resolved["secrets"])
}

func TestMergeLayer_ServicesReplaceEntirely(t *testing.T) {
	dst := &task.TaskSetting{}
	resolved := map[string]string{}

	task.ExportMergeLayer(dst, map[string]interface{}{
		"services": []interface{}{
			map[string]interface{}{"name": "web", "port": float64(3000), "protocol": "http", "public": true},
			map[string]interface{}{"name": "api", "port": float64(8080), "protocol": "http", "public": false},
		},
	}, "agent", resolved)

	require.Len(t, dst.Services, 2)

	task.ExportMergeLayer(dst, map[string]interface{}{
		"services": []interface{}{
			map[string]interface{}{"name": "custom", "port": float64(9000), "protocol": "tcp", "public": true},
		},
	}, "task", resolved)

	require.Len(t, dst.Services, 1, "services should be replaced entirely, not appended")
	assert.Equal(t, "custom", dst.Services[0].Name)
	assert.Equal(t, 9000, dst.Services[0].Port)
	assert.Equal(t, "task", resolved["services"])
}

func TestMergeLayer_SkillsReplaceEntirely(t *testing.T) {
	dst := &task.TaskSetting{}
	resolved := map[string]string{}

	task.ExportMergeLayer(dst, map[string]interface{}{
		"skills": []interface{}{"web-search", "code-review"},
	}, "agent", resolved)

	assert.Equal(t, []string{"web-search", "code-review"}, dst.Skills)

	task.ExportMergeLayer(dst, map[string]interface{}{
		"skills": []interface{}{"data-analysis"},
	}, "task", resolved)

	assert.Equal(t, []string{"data-analysis"}, dst.Skills, "skills should be replaced entirely")
	assert.Equal(t, "task", resolved["skills"])
}

func TestMergeLayer_ScheduleOverride(t *testing.T) {
	dst := &task.TaskSetting{}
	resolved := map[string]string{}

	task.ExportMergeLayer(dst, map[string]interface{}{
		"schedule": map[string]interface{}{
			"enabled": true,
			"mode":    "interval",
		},
	}, "agent", resolved)

	require.NotNil(t, dst.Schedule)
	assert.Equal(t, "interval", dst.Schedule.Mode)

	task.ExportMergeLayer(dst, map[string]interface{}{
		"schedule": map[string]interface{}{
			"enabled": true,
			"mode":    "times",
			"times":   []interface{}{"09:00", "14:00"},
		},
	}, "task", resolved)

	assert.Equal(t, "times", dst.Schedule.Mode)
	assert.Equal(t, "task", resolved["schedule"])
}

func TestMergeLayer_NilValueSkipped(t *testing.T) {
	dst := &task.TaskSetting{}
	resolved := map[string]string{}

	task.ExportMergeLayer(dst, map[string]interface{}{
		"runner": "openai",
	}, "system/team/user", resolved)

	task.ExportMergeLayer(dst, map[string]interface{}{
		"runner":   nil,
		"services": nil,
	}, "task", resolved)

	assert.Equal(t, "openai", dst.Runner, "nil value should not override")
	assert.Equal(t, "system/team/user", resolved["runner"])
}

func TestMergeLayer_SecretsMapStringString(t *testing.T) {
	dst := &task.TaskSetting{}
	resolved := map[string]string{}

	task.ExportMergeLayer(dst, map[string]interface{}{
		"secrets": map[string]string{"KEY": "val"},
	}, "system/team/user", resolved)

	assert.Equal(t, "val", dst.Secrets["KEY"])
}

func TestMergeLayer_EmptyServiceSliceDoesNotOverride(t *testing.T) {
	dst := &task.TaskSetting{}
	resolved := map[string]string{}

	task.ExportMergeLayer(dst, map[string]interface{}{
		"services": []interface{}{
			map[string]interface{}{"name": "web", "port": float64(3000), "protocol": "http", "public": true},
		},
	}, "agent", resolved)

	task.ExportMergeLayer(dst, map[string]interface{}{
		"services": []interface{}{},
	}, "task", resolved)

	assert.Len(t, dst.Services, 1, "empty slice should not override existing services")
	assert.Equal(t, "agent", resolved["services"])
}

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

	assert.Equal(t, "openai", data["runner"])
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
	assert.NotContains(t, data, "runner")
	assert.NotContains(t, data, "image")
	assert.NotContains(t, data, "secrets")
}

// --- toStringMap tests ---

func TestToStringMap_MapStringString(t *testing.T) {
	input := map[string]string{"a": "1", "b": "2"}
	result, ok := task.ExportToStringMap(input)
	assert.True(t, ok)
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, result)
}

func TestToStringMap_MapStringInterface(t *testing.T) {
	input := map[string]interface{}{"a": "1", "b": "2", "c": 3}
	result, ok := task.ExportToStringMap(input)
	assert.True(t, ok)
	assert.Equal(t, "1", result["a"])
	assert.Equal(t, "2", result["b"])
	assert.NotContains(t, result, "c", "non-string values should be skipped")
}

func TestToStringMap_UnsupportedType(t *testing.T) {
	_, ok := task.ExportToStringMap("not a map")
	assert.False(t, ok)
}

func TestToStringMap_Nil(t *testing.T) {
	_, ok := task.ExportToStringMap(nil)
	assert.False(t, ok)
}

// --- Full merge priority simulation ---

func TestMergePriority_TaskOverridesAgentOverridesBase(t *testing.T) {
	dst := &task.TaskSetting{}
	resolved := map[string]string{}

	// system/team/user (base)
	task.ExportMergeLayer(dst, map[string]interface{}{
		"runner":   "openai",
		"model":    "gpt-4o",
		"image":    "python:3.12",
		"secrets":  map[string]interface{}{"GLOBAL": "g1"},
		"services": []interface{}{map[string]interface{}{"name": "web", "port": float64(80), "protocol": "http", "public": true}},
		"skills":   []interface{}{"search"},
	}, "system/team/user", resolved)

	// agent override
	task.ExportMergeLayer(dst, map[string]interface{}{
		"model":   "claude-sonnet",
		"secrets": map[string]interface{}{"AGENT_KEY": "ak1"},
	}, "agent", resolved)

	// task override
	task.ExportMergeLayer(dst, map[string]interface{}{
		"model":  "local-llama",
		"skills": []interface{}{"code-review", "data-analysis"},
		"schedule": map[string]interface{}{
			"enabled": true,
			"mode":    "times",
			"times":   []interface{}{"09:00"},
		},
	}, "task", resolved)

	// Verify final state
	assert.Equal(t, "openai", dst.Runner, "runner from base")
	assert.Equal(t, "local-llama", dst.Model, "model from task (highest)")
	assert.Equal(t, "python:3.12", dst.Image, "image from base")
	assert.Equal(t, "g1", dst.Secrets["GLOBAL"], "base secret preserved")
	assert.Equal(t, "ak1", dst.Secrets["AGENT_KEY"], "agent secret added")
	assert.Len(t, dst.Services, 1, "services from base")
	assert.Equal(t, []string{"code-review", "data-analysis"}, dst.Skills, "skills from task")
	require.NotNil(t, dst.Schedule)
	assert.Equal(t, "times", dst.Schedule.Mode)

	// Verify resolved_from tracking
	assert.Equal(t, "system/team/user", resolved["runner"])
	assert.Equal(t, "task", resolved["model"])
	assert.Equal(t, "system/team/user", resolved["image"])
	assert.Equal(t, "agent", resolved["secrets"])
	assert.Equal(t, "system/team/user", resolved["services"])
	assert.Equal(t, "task", resolved["skills"])
	assert.Equal(t, "task", resolved["schedule"])
}
