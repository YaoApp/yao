package providers

import (
	"github.com/yaoapp/gou/graphrag/extraction/openai"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// ExtractorOpenAI is an extractor provider for entity and relationship extraction
type ExtractorOpenAI struct{}

// AutoRegister registers the extractor providers
func init() {
	factory.Extractors["__yao.openai"] = &ExtractorOpenAI{}
}

// === ExtractorOpenAI ===

// Make creates a new OpenAI extractor
func (e *ExtractorOpenAI) Make(option *kbtypes.ProviderOption) (types.Extraction, error) {
	// TODO: Map kbtypes.ProviderOption to openai.Options
	openaiOptions := openai.Options{
		// ConnectorName: "", // TODO: Get connector name from option
		// Concurrent:    0,  // Will use default
		// Model:         "", // Will use default
		// Temperature:   0,  // Will use default
		// MaxTokens:     0,  // Will use default
		// Prompt:        "", // Will use default
		// Toolcall:      nil, // Will use default
		// Tools:         nil, // Will use default
		// RetryAttempts: 0,  // Will use default
		// RetryDelay:    0,  // Will use default
	}
	return openai.NewOpenai(openaiOptions)
}

// Schema returns the schema for the OpenAI extractor
func (e *ExtractorOpenAI) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}
