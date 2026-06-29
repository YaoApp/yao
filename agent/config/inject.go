package config

import "fmt"

// LoadAssistantFunc loads assistant DSL defaults by ID.
// Must be injected at init-time (e.g. from agent/load.go) to avoid import cycles.
var LoadAssistantFunc func(assistantID string) (*AssistantDefaults, error)

func loadAssistantDefaults(assistantID string) (*AssistantDefaults, error) {
	if LoadAssistantFunc == nil {
		return nil, fmt.Errorf("config: LoadAssistantFunc not injected")
	}
	return LoadAssistantFunc(assistantID)
}
