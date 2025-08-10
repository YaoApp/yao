package providers

import (
	"github.com/yaoapp/gou/graphrag/embedding"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// OpenAI is an OpenAI embedding provider
type OpenAI struct{}

// Fastembed is a Fastembed embedding provider
type Fastembed struct{}

// AutoRegister registers the embedding providers
func init() {
	factory.Embeddings["__yao.openai"] = &OpenAI{}
	factory.Embeddings["__yao.fastembed"] = &Fastembed{}
}

// === OpenAI ===

// Make creates an OpenAI embedding provider
func (o *OpenAI) Make(option *kbtypes.ProviderOption) (types.Embedding, error) {
	// Start with default values
	options := embedding.OpenaiOptions{
		ConnectorName: "",   // Will be set from option
		Concurrent:    10,   // Default concurrent requests
		Dimension:     1536, // Default dimension for text-embedding-3-small
		Model:         "",   // Will be determined by connector or use default
	}

	// Extract values from Properties map
	if option != nil && option.Properties != nil {
		// Set connector name
		if connector, ok := option.Properties["connector"]; ok {
			if connectorStr, ok := connector.(string); ok {
				options.ConnectorName = connectorStr
			}
		}

		// Set dimensions
		if dimensions, ok := option.Properties["dimensions"]; ok {
			if dimensionsInt, ok := dimensions.(int); ok {
				options.Dimension = dimensionsInt
			} else if dimensionsFloat, ok := dimensions.(float64); ok {
				options.Dimension = int(dimensionsFloat)
			}
		}

		// Set concurrent requests
		if concurrent, ok := option.Properties["concurrent"]; ok {
			if concurrentInt, ok := concurrent.(int); ok {
				options.Concurrent = concurrentInt
			} else if concurrentFloat, ok := concurrent.(float64); ok {
				options.Concurrent = int(concurrentFloat)
			}
		}

		// Set model
		if model, ok := option.Properties["model"]; ok {
			if modelStr, ok := model.(string); ok {
				options.Model = modelStr
			}
		}
	}

	return embedding.NewOpenai(options)
}

// Schema returns the schema for the OpenAI embedding provider
func (o *OpenAI) Schema(provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error) {
	return factory.GetSchemaFromBindata(factory.ProviderTypeEmbedding, "openai", locale)
}

// === Fastembed ===

// Make creates a Fastembed embedding provider
func (f *Fastembed) Make(option *kbtypes.ProviderOption) (types.Embedding, error) {
	// Start with default values
	options := embedding.FastEmbedOptions{
		ConnectorName: "",  // Will be set from option
		Concurrent:    10,  // Default concurrent requests
		Dimension:     384, // Default dimension for BAAI/bge-small-en-v1.5
		Model:         "",  // Will be determined by connector or use default
		Host:          "",  // Will be determined by connector
		Key:           "",  // Will be determined by connector
	}

	// Extract values from Properties map
	if option != nil && option.Properties != nil {
		// Set connector name
		if connector, ok := option.Properties["connector"]; ok {
			if connectorStr, ok := connector.(string); ok {
				options.ConnectorName = connectorStr
			}
		}

		// Set dimensions
		if dimensions, ok := option.Properties["dimensions"]; ok {
			if dimensionsInt, ok := dimensions.(int); ok {
				options.Dimension = dimensionsInt
			} else if dimensionsFloat, ok := dimensions.(float64); ok {
				options.Dimension = int(dimensionsFloat)
			}
		}

		// Set concurrent requests
		if concurrent, ok := option.Properties["concurrent"]; ok {
			if concurrentInt, ok := concurrent.(int); ok {
				options.Concurrent = concurrentInt
			} else if concurrentFloat, ok := concurrent.(float64); ok {
				options.Concurrent = int(concurrentFloat)
			}
		}

		// Set model
		if model, ok := option.Properties["model"]; ok {
			if modelStr, ok := model.(string); ok {
				options.Model = modelStr
			}
		}

		// Set host
		if host, ok := option.Properties["host"]; ok {
			if hostStr, ok := host.(string); ok {
				options.Host = hostStr
			}
		}

		// Set key
		if key, ok := option.Properties["key"]; ok {
			if keyStr, ok := key.(string); ok {
				options.Key = keyStr
			}
		}
	}

	return embedding.NewFastEmbed(options)
}

// Schema returns the schema for the Fastembed embedding provider
func (f *Fastembed) Schema(provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error) {
	return factory.GetSchemaFromBindata(factory.ProviderTypeEmbedding, "fastembed", locale)
}
