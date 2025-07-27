package providers

import (
	"github.com/yaoapp/gou/graphrag/chunking"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// Structured is a structured chunking provider
type Structured struct{}

// Semantic is a semantic chunking provider
type Semantic struct{}

// AutoRegister registers the chunking providers
func init() {
	factory.Chunkings["__yao.structured"] = &Structured{}
	factory.Chunkings["__yao.semantic"] = &Semantic{}
}

// === Structured Chunking ===

// Make creates a structured chunking provider
func (s *Structured) Make(_ *kbtypes.ProviderOption) (types.Chunking, error) {
	return chunking.NewStructuredChunker(), nil
}

// Options returns the options for the structured chunking provider
func (s *Structured) Options(option *kbtypes.ProviderOption) (*types.ChunkingOptions, error) {
	if option == nil {
		// Return default structured options
		return &types.ChunkingOptions{
			Size:           300,
			Overlap:        20,
			MaxDepth:       3,
			SizeMultiplier: 3,
			MaxConcurrent:  10,
		}, nil
	}

	// Start with default values
	options := &types.ChunkingOptions{
		Size:           300,
		Overlap:        20,
		MaxDepth:       3,
		SizeMultiplier: 3,
		MaxConcurrent:  10,
	}

	// Extract values from Properties map
	if option.Properties != nil {
		if size, ok := option.Properties["size"]; ok {
			if sizeInt, ok := size.(int); ok {
				options.Size = sizeInt
			} else if sizeFloat, ok := size.(float64); ok {
				options.Size = int(sizeFloat)
			}
		}

		if overlap, ok := option.Properties["overlap"]; ok {
			if overlapInt, ok := overlap.(int); ok {
				options.Overlap = overlapInt
			} else if overlapFloat, ok := overlap.(float64); ok {
				options.Overlap = int(overlapFloat)
			}
		}

		if maxDepth, ok := option.Properties["max_depth"]; ok {
			if maxDepthInt, ok := maxDepth.(int); ok {
				options.MaxDepth = maxDepthInt
			} else if maxDepthFloat, ok := maxDepth.(float64); ok {
				options.MaxDepth = int(maxDepthFloat)
			}
		}

		if sizeMultiplier, ok := option.Properties["size_multiplier"]; ok {
			if sizeMultiplierInt, ok := sizeMultiplier.(int); ok {
				options.SizeMultiplier = sizeMultiplierInt
			} else if sizeMultiplierFloat, ok := sizeMultiplier.(float64); ok {
				options.SizeMultiplier = int(sizeMultiplierFloat)
			}
		}

		if maxConcurrent, ok := option.Properties["max_concurrent"]; ok {
			if maxConcurrentInt, ok := maxConcurrent.(int); ok {
				options.MaxConcurrent = maxConcurrentInt
			} else if maxConcurrentFloat, ok := maxConcurrent.(float64); ok {
				options.MaxConcurrent = int(maxConcurrentFloat)
			}
		}
	}

	return options, nil
}

// Schema returns the schema for the structured chunking provider
func (s *Structured) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}

// === Semantic Chunking ===

// Make creates a semantic chunking provider
func (s *Semantic) Make(_ *kbtypes.ProviderOption) (types.Chunking, error) {
	return chunking.NewSemanticChunker(nil), nil
}

// Options returns the options for the semantic chunking provider
func (s *Semantic) Options(option *kbtypes.ProviderOption) (*types.ChunkingOptions, error) {
	if option == nil {
		// Return default semantic options
		return &types.ChunkingOptions{
			Size:           300,
			Overlap:        50,
			MaxDepth:       3,
			SizeMultiplier: 3,
			MaxConcurrent:  10,
			SemanticOptions: &types.SemanticOptions{
				ContextSize:   1800, // Default L1 Size (ChunkSize * 6)
				MaxRetry:      3,
				MaxConcurrent: 10,
				Toolcall:      false,
			},
		}, nil
	}

	// Start with default values
	options := &types.ChunkingOptions{
		Size:           300,
		Overlap:        50,
		MaxDepth:       3,
		SizeMultiplier: 3,
		MaxConcurrent:  10,
		SemanticOptions: &types.SemanticOptions{
			ContextSize:   1800, // Default L1 Size (ChunkSize * 6)
			MaxRetry:      3,
			MaxConcurrent: 10,
			Toolcall:      false,
		},
	}

	// Extract values from Properties map
	if option.Properties != nil {
		// Basic chunking options
		if size, ok := option.Properties["size"]; ok {
			if sizeInt, ok := size.(int); ok {
				options.Size = sizeInt
				// Update context size based on new size
				options.SemanticOptions.ContextSize = sizeInt * 6
			} else if sizeFloat, ok := size.(float64); ok {
				options.Size = int(sizeFloat)
				options.SemanticOptions.ContextSize = int(sizeFloat) * 6
			}
		}

		if overlap, ok := option.Properties["overlap"]; ok {
			if overlapInt, ok := overlap.(int); ok {
				options.Overlap = overlapInt
			} else if overlapFloat, ok := overlap.(float64); ok {
				options.Overlap = int(overlapFloat)
			}
		}

		if maxDepth, ok := option.Properties["max_depth"]; ok {
			if maxDepthInt, ok := maxDepth.(int); ok {
				options.MaxDepth = maxDepthInt
			} else if maxDepthFloat, ok := maxDepth.(float64); ok {
				options.MaxDepth = int(maxDepthFloat)
			}
		}

		if sizeMultiplier, ok := option.Properties["size_multiplier"]; ok {
			if sizeMultiplierInt, ok := sizeMultiplier.(int); ok {
				options.SizeMultiplier = sizeMultiplierInt
			} else if sizeMultiplierFloat, ok := sizeMultiplier.(float64); ok {
				options.SizeMultiplier = int(sizeMultiplierFloat)
			}
		}

		if maxConcurrent, ok := option.Properties["max_concurrent"]; ok {
			if maxConcurrentInt, ok := maxConcurrent.(int); ok {
				options.MaxConcurrent = maxConcurrentInt
			} else if maxConcurrentFloat, ok := maxConcurrent.(float64); ok {
				options.MaxConcurrent = int(maxConcurrentFloat)
			}
		}

		// Semantic-specific options
		if connector, ok := option.Properties["connector"]; ok {
			if connectorStr, ok := connector.(string); ok {
				options.SemanticOptions.Connector = connectorStr
			}
		}

		if toolcall, ok := option.Properties["toolcall"]; ok {
			if toolcallBool, ok := toolcall.(bool); ok {
				options.SemanticOptions.Toolcall = toolcallBool
			}
		}

		if contextSize, ok := option.Properties["context_size"]; ok {
			if contextSizeInt, ok := contextSize.(int); ok {
				options.SemanticOptions.ContextSize = contextSizeInt
			} else if contextSizeFloat, ok := contextSize.(float64); ok {
				options.SemanticOptions.ContextSize = int(contextSizeFloat)
			}
		}

		if optionsStr, ok := option.Properties["options"]; ok {
			if optionsString, ok := optionsStr.(string); ok {
				options.SemanticOptions.Options = optionsString
			}
		}

		if prompt, ok := option.Properties["prompt"]; ok {
			if promptStr, ok := prompt.(string); ok {
				options.SemanticOptions.Prompt = promptStr
			}
		}

		if maxRetry, ok := option.Properties["max_retry"]; ok {
			if maxRetryInt, ok := maxRetry.(int); ok {
				options.SemanticOptions.MaxRetry = maxRetryInt
			} else if maxRetryFloat, ok := maxRetry.(float64); ok {
				options.SemanticOptions.MaxRetry = int(maxRetryFloat)
			}
		}

		if semanticMaxConcurrent, ok := option.Properties["semantic_max_concurrent"]; ok {
			if semanticMaxConcurrentInt, ok := semanticMaxConcurrent.(int); ok {
				options.SemanticOptions.MaxConcurrent = semanticMaxConcurrentInt
			} else if semanticMaxConcurrentFloat, ok := semanticMaxConcurrent.(float64); ok {
				options.SemanticOptions.MaxConcurrent = int(semanticMaxConcurrentFloat)
			}
		}
	}

	return options, nil
}

// Schema returns the schema for the semantic chunking provider
func (s *Semantic) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}
