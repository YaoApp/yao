package hook

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/context"
)

// Create create a new assistant
func (s *Script) Create(ctx *context.Context, messages []context.Message) (*context.HookCreateResponse, error) {
	res, err := s.Execute(ctx, "Create", messages)
	if err != nil {
		return nil, err
	}

	response, err := s.getHookCreateResponse(res)
	if err != nil {
		return nil, err
	}

	// Apply context adjustments from the response back to the context
	if response != nil {
		s.applyContextAdjustments(ctx, response)
	}

	return response, nil
}

// applyContextAdjustments applies context field overrides from the hook response back to the context
func (s *Script) applyContextAdjustments(ctx *context.Context, response *context.HookCreateResponse) {
	// Override assistant ID if provided
	if response.AssistantID != "" {
		ctx.AssistantID = response.AssistantID
	}

	// Override connector if provided
	if response.Connector != "" {
		ctx.Connector = response.Connector
	}

	// Override locale if provided
	if response.Locale != "" {
		ctx.Locale = response.Locale
	}

	// Override theme if provided
	if response.Theme != "" {
		ctx.Theme = response.Theme
	}

	// Override route if provided
	if response.Route != "" {
		ctx.Route = response.Route
	}

	// Merge or override metadata if provided
	if len(response.Metadata) > 0 {
		if ctx.Metadata == nil {
			ctx.Metadata = make(map[string]interface{})
		}
		// Merge metadata - response metadata takes precedence
		for key, value := range response.Metadata {
			ctx.Metadata[key] = value
		}
	}
}

// getHookCreateResponse convert the result to a HookCreateResponse
func (s *Script) getHookCreateResponse(res interface{}) (*context.HookCreateResponse, error) {
	// Handle nil result
	if res == nil {
		return nil, nil
	}

	// Handle undefined result (treat as nil)
	if _, ok := res.(bridge.UndefinedT); ok {
		return nil, nil
	}

	// Marshal to JSON and unmarshal to HookCreateResponse
	raw, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var response context.HookCreateResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to HookCreateResponse: %w", err)
	}

	return &response, nil
}
