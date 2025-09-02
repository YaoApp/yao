package kb

import (
	"fmt"
	"strings"

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

// JobOptions contains job options for async operations
type JobOptions struct {
	Name        string `json:"name,omitempty"`        // Job name (optional, defaults will be used)
	Description string `json:"description,omitempty"` // Job description (optional, defaults will be used)
	Icon        string `json:"icon,omitempty"`        // Job icon (optional, Material Icon name)
	Category    string `json:"category,omitempty"`    // Job category (optional, defaults will be used)
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

	// Job options for async operations
	Job *JobOptions `json:"job,omitempty"`
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
	// Segment texts to update
	SegmentTexts []types.SegmentText `json:"segment_texts" binding:"required"`
}

// UpdateVoteRequest represents the request for UpdateVote API
type UpdateVoteRequest struct {
	Segments        []types.SegmentVote    `json:"segments" binding:"required"`
	DefaultReaction *types.SegmentReaction `json:"default_reaction,omitempty"` // Optional default context for segments that don't have reaction
}

// UpdateHitRequest represents the request for UpdateHit API
type UpdateHitRequest struct {
	Segments        []types.SegmentHit     `json:"segments" binding:"required"`
	DefaultReaction *types.SegmentReaction `json:"default_reaction,omitempty"` // Optional default context for segments that don't have reaction
}

// UpdateScoreRequest represents the request for UpdateScore API
type UpdateScoreRequest struct {
	Segments []types.SegmentScore `json:"segments" binding:"required"`
}

// UpdateWeightRequest represents the request for UpdateWeight API
type UpdateWeightRequest struct {
	Segments []types.SegmentWeight `json:"segments" binding:"required"`
}

// UpdateScoresRequest represents the request for batch score updates
type UpdateScoresRequest struct {
	Scores []types.SegmentScore `json:"scores" binding:"required"`
}

// UpdateWeightsRequest represents the request for batch weight updates
type UpdateWeightsRequest struct {
	Weights []types.SegmentWeight `json:"weights" binding:"required"`
}

// ProviderOption resolves a ProviderConfig to a *kbtypes.ProviderOption
// If OptionID is provided, it looks up the option from the provider
// If Option is provided directly, it uses the Option field
// If neither is provided, it selects the default option from provider's Options
func (config *ProviderConfig) ProviderOption(providerType, locale string) (*kbtypes.ProviderOption, error) {
	if config == nil {
		return nil, fmt.Errorf("provider config is required")
	}

	if config.ProviderID == "" {
		return nil, fmt.Errorf("provider_id is required")
	}

	if providerType == "" {
		return nil, fmt.Errorf("provider_type is required")
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

	// Find the provider using the specified provider type
	var provider *kbtypes.Provider
	kbInstance := kb.Instance.(*kb.KnowledgeBase)

	// Get providers of the specific type
	providers := kbInstance.Providers.GetProviders(providerType, locale)
	for _, p := range providers {
		if p.ID == config.ProviderID {
			provider = p
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
	chunkingOption, err := r.Chunking.ProviderOption("chunking", locale)
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
	embeddingOption, err := r.Embedding.ProviderOption("embedding", locale)
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
		extractionOption, err := r.Extraction.ProviderOption("extraction", locale)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve extraction provider: %w", err)
		}

		extraction, err := factory.MakeExtraction(r.Extraction.ProviderID, extractionOption)
		if err != nil {
			return nil, fmt.Errorf("failed to create extraction provider: %w", err)
		}
		options.Extraction = extraction
	}

	if r.Fetcher != nil {
		fetcherOption, err := r.Fetcher.ProviderOption("fetcher", locale)
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
		converterOption, err := r.Converter.ProviderOption("converter", locale)
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

			converterOption, err := converterConfig.ProviderOption("converter", locale)
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

// Validate validates the UpdateWeightRequest fields
func (r *UpdateWeightRequest) Validate() error {
	if len(r.Segments) == 0 {
		return fmt.Errorf("segments is required")
	}
	for i, segment := range r.Segments {
		if strings.TrimSpace(segment.ID) == "" {
			return fmt.Errorf("segments[%d].id cannot be empty", i)
		}
		if segment.Weight < 0 {
			return fmt.Errorf("segments[%d].weight cannot be negative", i)
		}
	}
	return nil
}

// Validate validates the UpdateWeightsRequest fields
func (r *UpdateWeightsRequest) Validate() error {
	if len(r.Weights) == 0 {
		return fmt.Errorf("weights is required")
	}
	for i, weight := range r.Weights {
		if strings.TrimSpace(weight.ID) == "" {
			return fmt.Errorf("weights[%d].id cannot be empty", i)
		}
		if weight.Weight < 0 {
			return fmt.Errorf("weights[%d].weight cannot be negative", i)
		}
	}
	return nil
}

// GetJobOptions returns job options with defaults
func (r *BaseUpsertRequest) GetJobOptions(defaultName, defaultDescription, defaultIcon, defaultCategory string) (string, string, string, string) {
	name := defaultName
	description := defaultDescription
	icon := defaultIcon
	category := defaultCategory

	if r.Job != nil {
		if r.Job.Name != "" {
			name = r.Job.Name
		}
		if r.Job.Description != "" {
			description = r.Job.Description
		}
		if r.Job.Icon != "" {
			icon = r.Job.Icon
		}
		if r.Job.Category != "" {
			category = r.Job.Category
		}
	}

	return name, description, icon, category
}

// AddBaseFields adds common fields from BaseUpsertRequest to data map
func (r *BaseUpsertRequest) AddBaseFields(data map[string]interface{}) {
	if r.Locale != "" {
		data["locale"] = r.Locale
	}
	if r.DocID != "" {
		data["document_id"] = r.DocID
	}

	// Extract fields from metadata
	if r.Metadata != nil {
		// Extract specific fields from metadata
		if description, ok := r.Metadata["description"]; ok && description != nil {
			data["description"] = description
		}
		if cover, ok := r.Metadata["cover"]; ok && cover != nil {
			data["cover"] = cover
		}
		if tags, ok := r.Metadata["tags"]; ok && tags != nil {
			data["tags"] = tags
		}
		if name, ok := r.Metadata["name"]; ok && name != nil {
			data["name"] = name
		}
		// Note: Other metadata fields like size, type, etc. are handled by specific request types
	}

	// Add provider configurations
	if r.Converter != nil {
		data["converter_provider_id"] = r.Converter.ProviderID
		if r.Converter.OptionID != "" {
			data["converter_option_id"] = r.Converter.OptionID
		}
		if r.Converter.Option != nil {
			data["converter_properties"] = r.Converter.Option.Properties
		}
	}
	if r.Fetcher != nil {
		data["fetcher_provider_id"] = r.Fetcher.ProviderID
		if r.Fetcher.OptionID != "" {
			data["fetcher_option_id"] = r.Fetcher.OptionID
		}
		if r.Fetcher.Option != nil {
			data["fetcher_properties"] = r.Fetcher.Option.Properties
		}
	}
	if r.Chunking != nil {
		data["chunking_provider_id"] = r.Chunking.ProviderID
		if r.Chunking.OptionID != "" {
			data["chunking_option_id"] = r.Chunking.OptionID
		}
		if r.Chunking.Option != nil {
			data["chunking_properties"] = r.Chunking.Option.Properties
		}
	}
	if r.Embedding != nil {
		data["embedding_provider_id"] = r.Embedding.ProviderID
		if r.Embedding.OptionID != "" {
			data["embedding_option_id"] = r.Embedding.OptionID
		}
		if r.Embedding.Option != nil {
			data["embedding_properties"] = r.Embedding.Option.Properties
		}
	}
	if r.Extraction != nil {
		data["extraction_provider_id"] = r.Extraction.ProviderID
		if r.Extraction.OptionID != "" {
			data["extraction_option_id"] = r.Extraction.OptionID
		}
		if r.Extraction.Option != nil {
			data["extraction_properties"] = r.Extraction.Option.Properties
		}
	}
}
