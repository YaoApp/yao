package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/setting"
)

// GetConfig returns the merged task configuration across 5 layers:
// system -> team -> user -> agent -> task (later wins).
func GetConfig(ctx context.Context, auth *process.AuthorizedInfo, chatID string) (*Config, error) {
	task, err := Get(ctx, auth, chatID)
	if err != nil {
		return nil, fmt.Errorf("task.GetConfig: %w", err)
	}

	reg := setting.Global
	if reg == nil {
		return &Config{Setting: &TaskSetting{}}, nil
	}

	// Layers 1-3: system -> team -> user (via GetMerged)
	base, _ := reg.GetMerged(auth.UserID, auth.TeamID, "task-config")
	if base == nil {
		base = map[string]interface{}{}
	}

	// Layer 4: agent-level override
	agentScope := setting.ScopeID{Scope: setting.ScopeUser, UserID: auth.UserID}
	agentCfg, _ := reg.Get(agentScope, "task-config.agent."+task.AssistantID)

	// Layer 5: task-level override
	taskScope := setting.ScopeID{Scope: setting.ScopeUser, UserID: auth.UserID}
	taskCfg, _ := reg.Get(taskScope, "task-config.task."+chatID)

	// Merge: base <- agentCfg <- taskCfg
	merged := &TaskSetting{}
	resolvedFrom := map[string]string{}
	mergeLayer(merged, base, "system/team/user", resolvedFrom)
	if agentCfg != nil {
		mergeLayer(merged, agentCfg, "agent", resolvedFrom)
	}
	if taskCfg != nil {
		mergeLayer(merged, taskCfg, "task", resolvedFrom)
	}

	return &Config{
		Setting:      merged,
		ResolvedFrom: resolvedFrom,
	}, nil
}

// SetConfig writes configuration to the task layer.
func SetConfig(ctx context.Context, auth *process.AuthorizedInfo, chatID string, req *ConfigReq) error {
	reg := setting.Global
	if reg == nil {
		return fmt.Errorf("task.SetConfig: setting registry not initialized")
	}

	data := configReqToMap(req)
	scope := setting.ScopeID{Scope: setting.ScopeUser, UserID: auth.UserID}
	if _, err := reg.Set(scope, "task-config.task."+chatID, data); err != nil {
		return fmt.Errorf("task.SetConfig: %w", err)
	}

	if req.Schedule != nil {
		GlobalScheduleEngine.Update(chatID, *req.Schedule)
	}
	return nil
}

// mergeLayer applies a settings map onto the merged TaskSetting, tracking source.
func mergeLayer(dst *TaskSetting, src map[string]interface{}, layer string, resolved map[string]string) {
	if v, ok := src["runner"].(string); ok && v != "" {
		dst.Runner = v
		resolved["runner"] = layer
	}
	if v, ok := src["model"].(string); ok && v != "" {
		dst.Model = v
		resolved["model"] = layer
	}
	if v, ok := src["image"].(string); ok && v != "" {
		dst.Image = v
		resolved["image"] = layer
	}
	if v, ok := src["secrets"]; ok && v != nil {
		if sm, ok2 := toStringMap(v); ok2 {
			if dst.Secrets == nil {
				dst.Secrets = make(map[string]string)
			}
			for k, val := range sm {
				dst.Secrets[k] = val
			}
			resolved["secrets"] = layer
		}
	}
	if v, ok := src["services"]; ok && v != nil {
		if data, err := json.Marshal(v); err == nil {
			var svcs []ServiceDecl
			if json.Unmarshal(data, &svcs) == nil && len(svcs) > 0 {
				dst.Services = svcs
				resolved["services"] = layer
			}
		}
	}
	if v, ok := src["skills"]; ok && v != nil {
		if data, err := json.Marshal(v); err == nil {
			var skills []string
			if json.Unmarshal(data, &skills) == nil && len(skills) > 0 {
				dst.Skills = skills
				resolved["skills"] = layer
			}
		}
	}
	if v, ok := src["schedule"]; ok && v != nil {
		if data, err := json.Marshal(v); err == nil {
			var sched ScheduleConfig
			if json.Unmarshal(data, &sched) == nil {
				dst.Schedule = &sched
				resolved["schedule"] = layer
			}
		}
	}
}

// configReqToMap converts ConfigReq to a map for storage.
func configReqToMap(req *ConfigReq) map[string]interface{} {
	data := map[string]interface{}{}
	if req.Runner != nil {
		data["runner"] = *req.Runner
	}
	if req.Model != nil {
		data["model"] = *req.Model
	}
	if req.Image != nil {
		data["image"] = *req.Image
	}
	if req.Secrets != nil {
		secrets := map[string]interface{}{}
		for k, v := range req.Secrets {
			if v == nil {
				secrets[k] = nil
			} else {
				secrets[k] = *v
			}
		}
		data["secrets"] = secrets
	}
	if req.Services != nil {
		data["services"] = req.Services
	}
	if req.Skills != nil {
		data["skills"] = req.Skills
	}
	if req.Schedule != nil {
		data["schedule"] = req.Schedule
	}
	return data
}

// toStringMap attempts to convert an interface{} to map[string]string
func toStringMap(v interface{}) (map[string]string, bool) {
	switch m := v.(type) {
	case map[string]string:
		return m, true
	case map[string]interface{}:
		result := make(map[string]string, len(m))
		for k, val := range m {
			if s, ok := val.(string); ok {
				result[k] = s
			}
		}
		return result, true
	default:
		return nil, false
	}
}
