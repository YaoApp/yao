package hook

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/context"
)

// Next next hook for the next action after the completion
// opts is optional - if provided, will be passed to the hook
func (s *Script) Next(ctx *context.Context, payload *context.NextHookPayload, opts ...*context.Options) (*context.NextHookResponse, *context.Options, error) {
	// Get or create options
	var options *context.Options
	if len(opts) > 0 && opts[0] != nil {
		options = opts[0]
	} else {
		options = &context.Options{}
	}

	// Convert payload to map for JS (use JSON tag names)
	payloadMap := map[string]interface{}{
		"messages":   payload.Messages,
		"completion": payload.Completion,
		"tools":      payload.Tools,
		"error":      payload.Error,
	}

	// Execute hook with ctx, payload, and options (convert options to map for JS)
	optionsMap := options.ToMap()
	res, err := s.Execute(ctx, "Next", payloadMap, optionsMap)
	if err != nil {
		return nil, nil, err
	}

	response, err := s.getNextHookResponse(res)
	if err != nil {
		return nil, nil, err
	}

	return response, options, nil
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
