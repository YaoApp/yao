package kb

import (
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

/*
Usage Examples:

1. AddFile API (converter will be auto-detected based on file info):
{
  "collection_id": "my_collection",
  "locale": "en",
  "file_id": "uploaded_file_123",
  "chunking": {
    "provider_id": "__yao.structured",
    "option_id": "standard"
  },
  "embedding": {
    "provider_id": "__yao.openai",
    "option_id": "text-embedding-3-small"
  },
  "doc_id": "document_001",
  "metadata": {
    "source": "research_paper"
  }
}

2. AddText API with Chinese locale:
{
  "collection_id": "my_collection",
  "locale": "zh-cn",
  "text": "这是要处理的文本内容。",
  "chunking": {
    "provider_id": "__yao.structured"
  },
  "embedding": {
    "provider_id": "__yao.fastembed",
    "option_id": "fastembed-chinese"
  }
}

3. AddSegments API:
{
  "collection_id": "my_collection",
  "locale": "en",
  "doc_id": "document_001",
  "segment_texts": [
    {"text": "First segment", "metadata": {"page": 1}},
    {"text": "Second segment", "metadata": {"page": 2}}
  ],
  "embedding": {
    "provider_id": "__yao.openai",
    "option_id": "text-embedding-3-small"
  }
}

Note:
- If no locale is specified, defaults to "en"
- If no option_id is specified, the default option from provider configuration will be selected
- Providers are loaded based on locale with fallback to "en" if the specified locale is not available
- For AddFile API, converter will be auto-detected based on filename and content_type obtained from GetFileInfo(file_id)
- ToUpsertOptions() can be called without parameters, or with filename and contentType for converter auto-detection
*/

// ProviderConfig represents a provider configuration that can be specified in two ways:
// 1. ProviderID + OptionID (option will be looked up from provider)
// 2. ProviderID + Option (option is provided directly)
type ProviderConfig struct {
	ProviderID string                  `json:"provider_id" binding:"required"`
	OptionID   string                  `json:"option_id,omitempty"`
	Option     *kbtypes.ProviderOption `json:"option,omitempty"`
}

// BaseUpsertRequest contains common fields for all upsert operations
type BaseUpsertRequest struct {
	// Collection ID - this will be mapped to UpsertOptions.CollectionID
	CollectionID string `json:"collection_id" binding:"required"`

	// Language/locale for provider selection (defaults to "en")
	Locale string `json:"locale,omitempty"`

	// Provider configurations
	Chunking   *ProviderConfig `json:"chunking" binding:"required"`
	Embedding  *ProviderConfig `json:"embedding" binding:"required"`
	Extraction *ProviderConfig `json:"extraction,omitempty"`
	Fetcher    *ProviderConfig `json:"fetcher,omitempty"`
	Converter  *ProviderConfig `json:"converter,omitempty"`

	// Upsert options
	DocID    string                 `json:"doc_id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AddFileRequest represents the request for AddFile API
type AddFileRequest struct {
	BaseUpsertRequest
	FileID   string `json:"file_id" binding:"required"`
	Uploader string `json:"uploader,omitempty"` // The name of the uploader, e.g. "s3", "local", "webdav", etc.
}

// AddTextRequest represents the request for AddText API
type AddTextRequest struct {
	BaseUpsertRequest
	Text string `json:"text" binding:"required"`
}

// AddURLRequest represents the request for AddURL API
type AddURLRequest struct {
	BaseUpsertRequest
	URL string `json:"url" binding:"required"`
}

// AddSegmentsRequest represents the request for AddSegments API
type AddSegmentsRequest struct {
	BaseUpsertRequest
	SegmentTexts []types.SegmentText `json:"segment_texts" binding:"required"`
}

// UpdateSegmentsRequest represents the request for UpdateSegments API
type UpdateSegmentsRequest struct {
	BaseUpsertRequest
	SegmentTexts []types.SegmentText `json:"segment_texts" binding:"required"`
}

// resolveProviderOption resolves a ProviderConfig to a *kbtypes.ProviderOption
// If OptionID is provided, it looks up the option from the provider
// If Option is provided directly, it uses the Option field
// If neither is provided, it selects the default option from provider's Options
func resolveProviderOption(config *ProviderConfig, locale string) (*kbtypes.ProviderOption, error) {
	if config == nil {
		return nil, fmt.Errorf("provider config is required")
	}

	if config.ProviderID == "" {
		return nil, fmt.Errorf("provider_id is required")
	}

	// If Option is provided directly, use it
	if config.Option != nil {
		return config.Option, nil
	}

	// Get the provider from KB instance
	if kb.Instance == nil {
		return nil, fmt.Errorf("KB instance is not initialized")
	}

	// Default locale to "en" if not provided
	if locale == "" {
		locale = "en"
	}

	// Find the provider using the new multi-language system
	var provider *kbtypes.Provider
	kbInstance := kb.Instance.(*kb.KnowledgeBase)

	// Check all provider types to find the matching provider
	providerTypes := []string{"chunking", "embedding", "converter", "extractor", "fetcher"}

	for _, providerType := range providerTypes {
		providers := kbInstance.Providers.GetProviders(providerType, locale)
		for _, p := range providers {
			if p.ID == config.ProviderID {
				provider = p
				break
			}
		}
		if provider != nil {
			break
		}
	}

	if provider == nil {
		return nil, fmt.Errorf("provider %s not found for locale %s", config.ProviderID, locale)
	}

	// If OptionID is provided, look it up from the provider
	if config.OptionID != "" {
		option, exists := provider.GetOption(config.OptionID)
		if !exists {
			return nil, fmt.Errorf("option %s not found in provider %s", config.OptionID, config.ProviderID)
		}
		return option, nil
	}

	// If no option specified, try to find the default option
	if provider.Options != nil {
		for _, option := range provider.Options {
			if option.Default {
				return option, nil
			}
		}
		// If no default option found but options exist, return the first one
		if len(provider.Options) > 0 {
			return provider.Options[0], nil
		}
	}

	return nil, fmt.Errorf("no option specified and no default option found for provider %s", config.ProviderID)
}

// ToUpsertOptions converts BaseUpsertRequest to types.UpsertOptions
// Optional parameters: filename, contentType (for converter auto-detection)
func (r *BaseUpsertRequest) ToUpsertOptions(fileInfo ...string) (*types.UpsertOptions, error) {
	var filename, contentType string
	if len(fileInfo) >= 1 {
		filename = fileInfo[0]
	}
	if len(fileInfo) >= 2 {
		contentType = fileInfo[1]
	}

	// Default locale to "en" if not specified
	locale := r.Locale
	if locale == "" {
		locale = "en"
	}

	options := &types.UpsertOptions{
		CollectionID: r.CollectionID, // Collection ID maps to CollectionID
		DocID:        r.DocID,
		Metadata:     r.Metadata,
	}

	// Resolve and create chunking provider
	chunkingOption, err := resolveProviderOption(r.Chunking, locale)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve chunking provider: %w", err)
	}

	chunking, err := factory.MakeChunking(r.Chunking.ProviderID, chunkingOption)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunking provider: %w", err)
	}
	options.Chunking = chunking

	// Get chunking options
	chunkingOpts, err := factory.ChunkingOptions(r.Chunking.ProviderID, chunkingOption)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunking options: %w", err)
	}
	options.ChunkingOptions = chunkingOpts

	// Resolve and create embedding provider
	embeddingOption, err := resolveProviderOption(r.Embedding, locale)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve embedding provider: %w", err)
	}

	embedding, err := factory.MakeEmbedding(r.Embedding.ProviderID, embeddingOption)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding provider: %w", err)
	}
	options.Embedding = embedding

	// Optional providers
	if r.Extraction != nil {
		extractionOption, err := resolveProviderOption(r.Extraction, locale)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve extraction provider: %w", err)
		}

		extraction, err := factory.MakeExtractor(r.Extraction.ProviderID, extractionOption)
		if err != nil {
			return nil, fmt.Errorf("failed to create extraction provider: %w", err)
		}
		options.Extraction = extraction
	}

	if r.Fetcher != nil {
		fetcherOption, err := resolveProviderOption(r.Fetcher, locale)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve fetcher provider: %w", err)
		}

		fetcher, err := factory.MakeFetcher(r.Fetcher.ProviderID, fetcherOption)
		if err != nil {
			return nil, fmt.Errorf("failed to create fetcher provider: %w", err)
		}
		options.Fetcher = fetcher
	}

	// Handle converter - auto-detect if not specified
	if r.Converter != nil {
		// User specified converter
		converterOption, err := resolveProviderOption(r.Converter, locale)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve converter provider: %w", err)
		}

		converter, err := factory.MakeConverter(r.Converter.ProviderID, converterOption)
		if err != nil {
			return nil, fmt.Errorf("failed to create converter provider: %w", err)
		}
		options.Converter = converter
	} else if filename != "" || contentType != "" {
		// Auto-detect converter based on filename and content type
		matched, converterID, err := factory.AutoDetectConverter(filename, contentType)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-detect converter: %w", err)
		}

		if matched {
			// Find the provider to get default option
			converterConfig := &ProviderConfig{
				ProviderID: converterID,
			}

			converterOption, err := resolveProviderOption(converterConfig, locale)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve auto-detected converter provider: %w", err)
			}

			converter, err := factory.MakeConverter(converterID, converterOption)
			if err != nil {
				return nil, fmt.Errorf("failed to create auto-detected converter provider: %w", err)
			}
			options.Converter = converter
		}
	}

	return options, nil
}

// Validate validates the common fields
func (r *BaseUpsertRequest) Validate() error {
	if r.CollectionID == "" {
		return fmt.Errorf("collection_id is required")
	}
	if r.Chunking == nil {
		return fmt.Errorf("chunking provider is required")
	}
	if r.Embedding == nil {
		return fmt.Errorf("embedding provider is required")
	}
	return nil
}

// Validate validates the AddFileRequest fields
func (r *AddFileRequest) Validate() error {
	if err := r.BaseUpsertRequest.Validate(); err != nil {
		return err
	}
	if r.FileID == "" {
		return fmt.Errorf("file_id is required")
	}
	return nil
}

// Validate validates the AddTextRequest fields
func (r *AddTextRequest) Validate() error {
	if err := r.BaseUpsertRequest.Validate(); err != nil {
		return err
	}
	if r.Text == "" {
		return fmt.Errorf("text is required")
	}
	return nil
}

// Validate validates the AddURLRequest fields
func (r *AddURLRequest) Validate() error {
	if err := r.BaseUpsertRequest.Validate(); err != nil {
		return err
	}
	if r.URL == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

// Validate validates the AddSegmentsRequest fields
func (r *AddSegmentsRequest) Validate() error {
	if err := r.BaseUpsertRequest.Validate(); err != nil {
		return err
	}
	if len(r.SegmentTexts) == 0 {
		return fmt.Errorf("segment_texts is required")
	}
	if r.DocID == "" {
		return fmt.Errorf("doc_id is required for AddSegments operation")
	}
	return nil
}

// Validate validates the UpdateSegmentsRequest fields
func (r *UpdateSegmentsRequest) Validate() error {
	if err := r.BaseUpsertRequest.Validate(); err != nil {
		return err
	}
	if len(r.SegmentTexts) == 0 {
		return fmt.Errorf("segment_texts is required")
	}
	return nil
}
