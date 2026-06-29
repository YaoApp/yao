package task

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/process"
	agentconfig "github.com/yaoapp/yao/agent/config"
	"github.com/yaoapp/yao/setting"
)

// GetConfig returns the merged task configuration across all layers.
// Delegates to the unified agent/config.Resolve and converts to the legacy *Config type.
func GetConfig(ctx context.Context, auth *process.AuthorizedInfo, chatID string) (*Config, error) {
	task, err := Get(ctx, auth, chatID)
	if err != nil {
		return nil, fmt.Errorf("task.GetConfig: %w", err)
	}

	resolved, err := agentconfig.Resolve(agentconfig.ResolveOptions{
		AssistantID: task.AssistantID,
		ChatID:      chatID,
		UserID:      auth.UserID,
		TeamID:      auth.TeamID,
	})
	if err != nil {
		return nil, fmt.Errorf("task.GetConfig: %w", err)
	}

	ts := &TaskSetting{
		Runner:   resolved.Runner,
		Model:    resolved.Model,
		Image:    resolved.Image,
		Timeout:  resolved.Timeout,
		MaxTurns: resolved.MaxTurns,
		Secrets:  resolved.Secrets,
		Skills:   resolved.Skills,
	}

	// Convert services
	for _, svc := range resolved.Services {
		ts.Services = append(ts.Services, ServiceDecl{
			Name:     svc.Name,
			Port:     svc.Port,
			Protocol: svc.Protocol,
			Public:   svc.Public,
		})
	}

	// Convert schedule
	if resolved.Schedule != nil {
		ts.Schedule = &ScheduleConfig{
			Enabled:       resolved.Schedule.Enabled,
			Mode:          resolved.Schedule.Mode,
			Times:         resolved.Schedule.Times,
			Days:          resolved.Schedule.Days,
			IntervalValue: resolved.Schedule.IntervalValue,
			IntervalUnit:  resolved.Schedule.IntervalUnit,
			Timezone:      resolved.Schedule.Timezone,
			StartDate:     resolved.Schedule.StartDate,
			EndDate:       resolved.Schedule.EndDate,
		}
	}

	return &Config{
		Setting:      ts,
		ResolvedFrom: resolved.ResolvedFrom,
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

// configReqToMap converts ConfigReq to a map for storage.
func configReqToMap(req *ConfigReq) map[string]interface{} {
	data := map[string]interface{}{}
	if req.Runner != nil {
		data["runners"] = []string{*req.Runner}
	}
	if req.Model != nil {
		data["model"] = *req.Model
	}
	if req.Image != nil {
		data["image"] = *req.Image
	}
	if req.Timeout != nil {
		data["timeout"] = *req.Timeout
	}
	if req.MaxTurns != nil {
		data["max_turns"] = *req.MaxTurns
	}
	if req.Secrets != nil {
		secrets := map[string]interface{}{}
		for k, v := range req.Secrets {
			if v == nil {
				secrets[k] = nil
			} else {
				secrets[k] = map[string]interface{}{"value": setting.Encrypt(*v)}
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
