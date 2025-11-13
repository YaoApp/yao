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
	return s.getHookCreateResponse(res)
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
