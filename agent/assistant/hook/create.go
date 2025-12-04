package hook

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/context"
)

// Create create a new assistant
// opts is optional - if provided, will be adjusted based on hook response
func (s *Script) Create(ctx *context.Context, messages []context.Message, opts ...*context.Options) (*context.HookCreateResponse, *context.Options, error) {
	// Get or create options
	var options *context.Options
	if len(opts) > 0 && opts[0] != nil {
		options = opts[0]
	} else {
		options = &context.Options{}
	}

	// Execute hook with ctx, messages, and options (convert options to map for JS)
	optionsMap := options.ToMap()
	res, err := s.Execute(ctx, "Create", messages, optionsMap)
	if err != nil {
		return nil, nil, err
	}

	response, err := s.getHookCreateResponse(res)
	if err != nil {
		return nil, nil, err
	}

	// Apply adjustments from the response
	if response != nil {
		s.applyContextAdjustments(ctx, response)
		s.applyOptionsAdjustments(options, response)
	}

	return response, options, nil
}

// applyContextAdjustments applies session-level field overrides from the hook response back to the context
func (s *Script) applyContextAdjustments(ctx *context.Context, response *context.HookCreateResponse) {
	// Note: AssistantID cannot be overridden - it's set at initialization and immutable

	// Override locale if provided (session-level)
	if response.Locale != "" {
		ctx.Locale = response.Locale
	}

	// Override theme if provided (session-level)
	if response.Theme != "" {
		ctx.Theme = response.Theme
	}

	// Override route if provided (session-level)
	if response.Route != "" {
		ctx.Route = response.Route
	}

	// Merge or override metadata if provided (session-level)
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

// applyOptionsAdjustments applies call-level field overrides from the hook response to options
func (s *Script) applyOptionsAdjustments(opts *context.Options, response *context.HookCreateResponse) {
	// Override connector if provided (call-level parameter)
	if response.Connector != "" {
		opts.Connector = response.Connector
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
