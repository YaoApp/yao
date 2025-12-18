package context

// ToMap converts Options struct to map for JSON serialization
func (opts *Options) ToMap() map[string]interface{} {
	if opts == nil {
		return nil
	}

	result := make(map[string]interface{})

	// Add configurable fields (with json tags)
	if opts.Connector != "" {
		result["connector"] = opts.Connector
	}
	if opts.Mode != "" {
		result["mode"] = opts.Mode
	}
	if opts.Search != nil {
		result["search"] = opts.Search
	}
	if opts.Skip != nil {
		result["skip"] = opts.Skip
	}
	// Only add DisableGlobalPrompts if true (avoid false values in map)
	if opts.DisableGlobalPrompts {
		result["disable_global_prompts"] = opts.DisableGlobalPrompts
	}
	if opts.Metadata != nil {
		result["metadata"] = opts.Metadata
	}

	// Note: Runtime fields (Context, Writer) are not serialized (json:"-")
	// They should not be included in the map

	return result
}

// OptionsFromMap creates Options struct from map (e.g., from JS Hook)
func OptionsFromMap(m map[string]interface{}) *Options {
	if m == nil {
		return &Options{}
	}

	opts := &Options{}

	// Extract configurable fields
	if connector, ok := m["connector"].(string); ok {
		opts.Connector = connector
	}
	if mode, ok := m["mode"].(string); ok {
		opts.Mode = mode
	}
	// Search supports: bool | SearchIntent | map[string]any | nil
	if search := m["search"]; search != nil {
		opts.Search = search
	}
	if skipMap, ok := m["skip"].(map[string]interface{}); ok {
		skip := &Skip{}
		if history, ok := skipMap["history"].(bool); ok {
			skip.History = history
		}
		if trace, ok := skipMap["trace"].(bool); ok {
			skip.Trace = trace
		}
		if output, ok := skipMap["output"].(bool); ok {
			skip.Output = output
		}
		opts.Skip = skip
	}
	if disableGlobalPrompts, ok := m["disable_global_prompts"].(bool); ok {
		opts.DisableGlobalPrompts = disableGlobalPrompts
	}
	if metadata, ok := m["metadata"].(map[string]interface{}); ok {
		opts.Metadata = metadata
	}

	// Note: Context and Writer are runtime fields, not restored from map
	// They should be set by the caller if needed

	return opts
}
