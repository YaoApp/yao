package hook

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/context"
)

// Next next hook for the next action after the completion
func (s *Script) Next(ctx *context.Context, payload *context.NextHookPayload) (*context.NextHookResponse, error) {
	// Convert payload to map for JS (use JSON tag names)
	payloadMap := map[string]interface{}{
		"messages":   payload.Messages,
		"completion": payload.Completion,
		"tools":      payload.Tools,
		"error":      payload.Error,
	}

	res, err := s.Execute(ctx, "Next", payloadMap)
	if err != nil {
		return nil, err
	}

	return s.getNextHookResponse(res)
}

// getNextHookResponse convert the result to a NextHookResponse
func (s *Script) getNextHookResponse(res interface{}) (*context.NextHookResponse, error) {
	// Handle nil result
	if res == nil {
		return nil, nil
	}

	// Handle undefined result (treat as nil)
	if _, ok := res.(bridge.UndefinedT); ok {
		return nil, nil
	}

	// Marshal to JSON and unmarshal to NextHookResponse
	raw, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Next hook result: %w", err)
	}

	var response context.NextHookResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to NextHookResponse: %w", err)
	}

	return &response, nil
}
