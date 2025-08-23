package providers

import (
	"time"

	"github.com/yaoapp/gou/graphrag/extraction/openai"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// ExtractionOpenAI is an OpenAI extraction provider
type ExtractionOpenAI struct{}

// AutoRegister registers the extraction providers
func init() {
	factory.Extractions["__yao.openai"] = &ExtractionOpenAI{}
}

// Make creates a new OpenAI extraction
func (e *ExtractionOpenAI) Make(option *kbtypes.ProviderOption) (types.Extraction, error) {
	// Start with default values
	options := openai.Options{
		ConnectorName: "",          // Will be set from option
		Concurrent:    5,           // Default concurrent requests for extraction
		Model:         "",          // Will be determined by connector or use default
		Temperature:   0.1,         // Low temperature for consistent extraction
		MaxTokens:     4000,        // Default max tokens
		Prompt:        "",          // Custom prompt (optional)
		Toolcall:      nil,         // Will be set from option (nil = default true)
		Tools:         nil,         // Will use default extraction tools
		RetryAttempts: 3,           // Default retry attempts
		RetryDelay:    time.Second, // Default retry delay
	}

	// Extract values from Properties map
	if option != nil && option.Properties != nil {
		// Set connector name
		if connector, ok := option.Properties["connector"]; ok {
			if connectorStr, ok := connector.(string); ok {
				options.ConnectorName = connectorStr
			}
		}

		// Set toolcall (explicit bool pointer)
		if toolcall, ok := option.Properties["toolcall"]; ok {
			if toolcallBool, ok := toolcall.(bool); ok {
				options.Toolcall = &toolcallBool
			}
		}

		// Set temperature
		if temperature, ok := option.Properties["temperature"]; ok {
			if temperatureFloat, ok := temperature.(float64); ok {
				options.Temperature = temperatureFloat
			} else if temperatureInt, ok := temperature.(int); ok {
				options.Temperature = float64(temperatureInt)
			}
		}

		// Set max tokens
		if maxTokens, ok := option.Properties["max_tokens"]; ok {
			if maxTokensInt, ok := maxTokens.(int); ok {
				options.MaxTokens = maxTokensInt
			} else if maxTokensFloat, ok := maxTokens.(float64); ok {
				options.MaxTokens = int(maxTokensFloat)
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

		// Set custom prompt
		if prompt, ok := option.Properties["prompt"]; ok {
			if promptStr, ok := prompt.(string); ok {
				options.Prompt = promptStr
			}
		}

		// Set retry attempts
		if retryAttempts, ok := option.Properties["retry_attempts"]; ok {
			if retryAttemptsInt, ok := retryAttempts.(int); ok {
				options.RetryAttempts = retryAttemptsInt
			} else if retryAttemptsFloat, ok := retryAttempts.(float64); ok {
				options.RetryAttempts = int(retryAttemptsFloat)
			}
		}

		// Set retry delay (in seconds)
		if retryDelay, ok := option.Properties["retry_delay"]; ok {
			if retryDelayFloat, ok := retryDelay.(float64); ok {
				options.RetryDelay = time.Duration(retryDelayFloat * float64(time.Second))
			} else if retryDelayInt, ok := retryDelay.(int); ok {
				options.RetryDelay = time.Duration(retryDelayInt) * time.Second
			}
		}

		// Set custom tools (advanced usage)
		if tools, ok := option.Properties["tools"]; ok {
			if toolsSlice, ok := tools.([]interface{}); ok {
				customTools := make([]map[string]interface{}, 0, len(toolsSlice))
				for _, tool := range toolsSlice {
					if toolMap, ok := tool.(map[string]interface{}); ok {
						customTools = append(customTools, toolMap)
					}
				}
				if len(customTools) > 0 {
					options.Tools = customTools
				}
			}
		}
	}

	return openai.NewOpenai(options)
}

// Schema returns the schema for the OpenAI extraction provider
func (e *ExtractionOpenAI) Schema(provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error) {
	return factory.GetSchemaFromBindata(factory.ProviderTypeExtraction, "openai", locale)
}
