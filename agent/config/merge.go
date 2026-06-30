package config

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/yao/setting"
)

// Resolve merges configuration across three progressive layers:
//
//   - L1 (DSL): Assistant defaults from package.yao / sandbox.yao
//   - L2 (User): Registry "agent.{assistantID}" (system→team→user cascade via GetMerged)
//   - L3 (Task): Registry "task-config.task.{chatID}" (user scope, optional)
func Resolve(opts ResolveOptions) (*Resolved, error) {
	if opts.AssistantID == "" {
		return nil, fmt.Errorf("config.Resolve: AssistantID is required")
	}

	resolved := &Resolved{
		ResolvedFrom: make(map[string]string),
	}

	// L1: Assistant DSL defaults
	defaults, err := loadAssistantDefaults(opts.AssistantID)
	if err != nil {
		return nil, fmt.Errorf("config.Resolve: load assistant defaults: %w", err)
	}
	if defaults != nil {
		applyDefaults(resolved, defaults)
	}

	reg := setting.Global
	if reg == nil {
		return resolved, nil
	}

	// L2: User preferences for this agent (system→team→user cascade)
	agentCfg, _ := reg.GetMerged(opts.UserID, opts.TeamID, "agent."+opts.AssistantID)
	if agentCfg != nil {
		mergeLayer(resolved, agentCfg, "user")
	}

	// L3: Task-level override (user scope, only when ChatID is set)
	if opts.ChatID != "" && opts.UserID != "" {
		taskScope := setting.ScopeID{Scope: setting.ScopeUser, UserID: opts.UserID}
		taskCfg, _ := reg.Get(taskScope, "task-config.task."+opts.ChatID)
		if taskCfg != nil {
			mergeLayer(resolved, taskCfg, "task")
		}
	}

	return resolved, nil
}

// applyDefaults applies Layer 0 assistant DSL values.
func applyDefaults(dst *Resolved, src *AssistantDefaults) {
	if src.Connector != "" {
		dst.Model = src.Connector
		dst.ResolvedFrom["model"] = "dsl"
	}
	if src.Runner != "" {
		dst.Runner = src.Runner
		dst.Runners = []string{src.Runner}
		dst.ResolvedFrom["runner"] = "dsl"
	}
	if src.Image != "" {
		dst.Image = src.Image
		dst.ResolvedFrom["image"] = "dsl"
	}
	if src.MaxTurns > 0 {
		dst.MaxTurns = src.MaxTurns
		dst.ResolvedFrom["max_turns"] = "dsl"
	}
	if len(src.Secrets) > 0 {
		dst.Secrets = make(map[string]SecretInfo, len(src.Secrets))
		for k, meta := range src.Secrets {
			dst.Secrets[k] = SecretInfo{
				HasValue:    false,
				Predefined:  true,
				Label:       meta.Label,
				Description: meta.Description,
				Required:    meta.Required,
				Multiline:   meta.Multiline,
				Source:      "dsl",
			}
		}
		dst.ResolvedFrom["secrets"] = "dsl"
	}
	if len(src.Services) > 0 {
		dst.Services = src.Services
		dst.ResolvedFrom["services"] = "dsl"
	}
	if len(src.Skills) > 0 {
		dst.Skills = src.Skills
		dst.ResolvedFrom["skills"] = "dsl"
	}
}

// mergeLayer applies a settings map onto Resolved, tracking the source layer.
func mergeLayer(dst *Resolved, src map[string]interface{}, layer string) {
	if v, ok := src["runner"].(string); ok && v != "" {
		dst.Runner = v
		dst.Runners = []string{v}
		dst.ResolvedFrom["runner"] = layer
	} else if arr, ok := src["runners"]; ok && arr != nil {
		if runners := toStringSlice(arr); len(runners) > 0 {
			dst.Runners = runners
			dst.Runner = runners[0]
			dst.ResolvedFrom["runner"] = layer
		}
	}
	if v, ok := src["model"].(string); ok && v != "" {
		dst.Model = v
		dst.ResolvedFrom["model"] = layer
	}
	if v, ok := src["image"].(string); ok && v != "" {
		dst.Image = v
		dst.ResolvedFrom["image"] = layer
	}
	if v, ok := src["timeout"].(string); ok && v != "" {
		dst.Timeout = v
		dst.ResolvedFrom["timeout"] = layer
	}
	if v, ok := src["max_turns"]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			if int(n) > 0 {
				dst.MaxTurns = int(n)
				dst.ResolvedFrom["max_turns"] = layer
			}
		case int:
			if n > 0 {
				dst.MaxTurns = n
				dst.ResolvedFrom["max_turns"] = layer
			}
		case json.Number:
			if i, err := n.Int64(); err == nil && i > 0 {
				dst.MaxTurns = int(i)
				dst.ResolvedFrom["max_turns"] = layer
			}
		}
	}
	if v, ok := src["secrets"]; ok && v != nil {
		mergeSecrets(dst, v, layer)
	}
	if v, ok := src["services"]; ok && v != nil {
		if svcs := toServiceDecls(v); len(svcs) > 0 {
			dst.Services = svcs
			dst.ResolvedFrom["services"] = layer
		}
	}
	if v, ok := src["skills"]; ok && v != nil {
		if data, err := json.Marshal(v); err == nil {
			var skills []string
			if json.Unmarshal(data, &skills) == nil && len(skills) > 0 {
				dst.Skills = skills
				dst.ResolvedFrom["skills"] = layer
			}
		}
	}
	if v, ok := src["schedule"]; ok && v != nil {
		if data, err := json.Marshal(v); err == nil {
			var sched ScheduleConfig
			if json.Unmarshal(data, &sched) == nil {
				dst.Schedule = &sched
				dst.ResolvedFrom["schedule"] = layer
			}
		}
	}
}

// mergeSecrets merges per-key secret info from a registry layer.
// Preserves DSL metadata for predefined keys, adds new keys as non-predefined.
func mergeSecrets(dst *Resolved, raw interface{}, layer string) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return
	}
	if dst.Secrets == nil {
		dst.Secrets = make(map[string]SecretInfo)
	}
	for k, val := range m {
		if val == nil {
			if existing, ok := dst.Secrets[k]; ok {
				existing.HasValue = false
				existing.Source = layer
				dst.Secrets[k] = existing
			}
			continue
		}
		entry, isMap := val.(map[string]interface{})
		if !isMap {
			continue
		}

		value, _ := entry["value"].(string)
		hasValue := value != ""

		if existing, ok := dst.Secrets[k]; ok {
			existing.HasValue = hasValue
			existing.Source = layer
			dst.Secrets[k] = existing
		} else {
			dst.Secrets[k] = SecretInfo{
				HasValue:    hasValue,
				Predefined:  false,
				Label:       stringFromMap(entry, "label"),
				Description: stringFromMap(entry, "description"),
				Required:    boolFromMap(entry, "required"),
				Multiline:   boolFromMap(entry, "multiline"),
				Source:      layer,
			}
		}
	}
	dst.ResolvedFrom["secrets"] = layer
}

// toStringSlice converts an array-like value to []string.
func toStringSlice(v interface{}) []string {
	switch arr := v.(type) {
	case []string:
		return arr
	case []interface{}:
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok && s != "" {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// toServiceDecls converts services from either ServiceDecl format ({name,port,...})
// or ServiceConfig format ({label,port}).
func toServiceDecls(v interface{}) []ServiceDecl {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	var raw []map[string]interface{}
	if json.Unmarshal(data, &raw) != nil || len(raw) == 0 {
		return nil
	}

	result := make([]ServiceDecl, 0, len(raw))
	for _, item := range raw {
		svc := ServiceDecl{}
		if name, ok := item["name"].(string); ok {
			svc.Name = name
		} else if label, ok := item["label"].(string); ok {
			svc.Name = label
		}
		switch p := item["port"].(type) {
		case float64:
			svc.Port = int(p)
		case int:
			svc.Port = p
		case json.Number:
			if i, err := p.Int64(); err == nil {
				svc.Port = int(i)
			}
		}
		if proto, ok := item["protocol"].(string); ok {
			svc.Protocol = proto
		}
		if pub, ok := item["public"].(bool); ok {
			svc.Public = pub
		}
		if svc.Port > 0 {
			result = append(result, svc)
		}
	}
	return result
}

func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func boolFromMap(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
