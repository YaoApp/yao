package kb

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/yao/attachment"
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
func resolveProviderOption(config *ProviderConfig, providerType, locale string) (*kbtypes.ProviderOption, error) {
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
	chunkingOption, err := resolveProviderOption(r.Chunking, "chunking", locale)
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
	embeddingOption, err := resolveProviderOption(r.Embedding, "embedding", locale)
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
		extractionOption, err := resolveProviderOption(r.Extraction, "extraction", locale)
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
		fetcherOption, err := resolveProviderOption(r.Fetcher, "fetcher", locale)
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
		converterOption, err := resolveProviderOption(r.Converter, "converter", locale)
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

			converterOption, err := resolveProviderOption(converterConfig, "converter", locale)
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

// PrepareCreateCollection prepares CreateCollection request and database data
func PrepareCreateCollection(c *gin.Context) (*CreateCollectionRequest, map[string]interface{}, error) {
	var req CreateCollectionRequest

	// Parse and bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, nil, fmt.Errorf("invalid request format: %w", err)
	}

	// Get provider settings first to resolve dimension
	providerSettings, err := getProviderSettings(req.Config.EmbeddingProvider, req.Config.EmbeddingOption, req.Config.Locale)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve provider settings: %w", err)
	}

	// Set dimension from provider settings
	req.Config.Dimension = providerSettings.Dimension

	// Add metadata with provider information
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}
	req.Metadata["__embedding_provider"] = req.Config.EmbeddingProvider
	req.Metadata["__embedding_option"] = req.Config.EmbeddingOption
	if req.Config.Locale != "" {
		req.Metadata["__locale"] = req.Config.Locale
	}

	// Now validate request parameters (after dimension and metadata are set)
	if err := validateCreateCollectionRequest(&req); err != nil {
		return nil, nil, err
	}

	// Prepare collection data for database
	data := map[string]interface{}{
		"collection_id":      req.ID,
		"name":               req.Metadata["name"],
		"description":        req.Metadata["description"],
		"status":             "creating",
		"embedding_provider": req.Config.EmbeddingProvider,
		"embedding_option":   req.Config.EmbeddingOption,
		"locale":             req.Config.Locale,
		"distance":           req.Config.Distance,
		"index_type":         req.Config.IndexType,
	}

	// Add optional HNSW parameters
	if req.Config.M > 0 {
		data["m"] = req.Config.M
	}
	if req.Config.EfConstruction > 0 {
		data["ef_construction"] = req.Config.EfConstruction
	}
	if req.Config.EfSearch > 0 {
		data["ef_search"] = req.Config.EfSearch
	}

	// Add optional IVF parameters
	if req.Config.NumLists > 0 {
		data["num_lists"] = req.Config.NumLists
	}
	if req.Config.NumProbes > 0 {
		data["num_probes"] = req.Config.NumProbes
	}

	// Add context fields (permissions, user info, etc.)
	addContextFields(c, data)

	return &req, data, nil
}

// PrepareAddFile prepares AddFile request and database data
func PrepareAddFile(c *gin.Context) (*AddFileRequest, map[string]interface{}, error) {
	var req AddFileRequest

	// Parse and validate request
	if err := validateRequest(c, &req); err != nil {
		return nil, nil, err
	}

	// Validate file and get path
	path, contentType, err := validateFileAndGetPath(c, &req)
	if err != nil {
		return nil, nil, err
	}

	// Get file info
	m, _ := attachment.Managers[req.Uploader]
	fileInfo, _ := m.Info(c.Request.Context(), req.FileID)

	// Generate document ID if not provided
	if req.DocID == "" {
		req.DocID = utils.GenDocIDWithCollectionID(req.CollectionID)
	}

	// Prepare document data for database
	data := map[string]interface{}{
		"document_id":    req.DocID,
		"collection_id":  req.CollectionID,
		"name":           fileInfo.Filename,
		"type":           "file",
		"status":         "pending",
		"uploader_id":    req.Uploader,
		"file_name":      fileInfo.Filename,
		"file_path":      path,
		"file_mime_type": contentType,
		"size":           int64(fileInfo.Bytes),
	}

	addBaseRequestFields(data, &req.BaseUpsertRequest)
	addContextFields(c, data)

	return &req, data, nil
}

// PrepareAddText prepares AddText request and database data
func PrepareAddText(c *gin.Context) (*AddTextRequest, map[string]interface{}, error) {
	var req AddTextRequest

	// Parse and validate request
	if err := validateRequest(c, &req); err != nil {
		return nil, nil, err
	}

	// Generate document ID if not provided
	if req.DocID == "" {
		req.DocID = utils.GenDocIDWithCollectionID(req.CollectionID)
	}

	// Prepare document data for database
	data := map[string]interface{}{
		"document_id":   req.DocID,
		"collection_id": req.CollectionID,
		"name":          "Text Document",
		"type":          "text",
		"status":        "pending",
		"text_content":  req.Text,
		"size":          int64(len(req.Text)),
	}

	// Use title from metadata if available
	if req.Metadata != nil {
		if title, ok := req.Metadata["title"].(string); ok && title != "" {
			data["name"] = title
		}
	}

	addBaseRequestFields(data, &req.BaseUpsertRequest)
	addContextFields(c, data)

	return &req, data, nil
}

// PrepareAddURL prepares AddURL request and database data
func PrepareAddURL(c *gin.Context) (*AddURLRequest, map[string]interface{}, error) {
	var req AddURLRequest

	// Parse and validate request
	if err := validateRequest(c, &req); err != nil {
		return nil, nil, err
	}

	// Generate document ID if not provided
	if req.DocID == "" {
		req.DocID = utils.GenDocIDWithCollectionID(req.CollectionID)
	}

	// Prepare document data for database
	data := map[string]interface{}{
		"document_id":   req.DocID,
		"collection_id": req.CollectionID,
		"name":          req.URL,
		"type":          "url",
		"status":        "pending",
		"url":           req.URL,
	}

	// Use title from metadata if available
	if req.Metadata != nil {
		if title, ok := req.Metadata["title"].(string); ok && title != "" {
			data["name"] = title
			data["url_title"] = title
		}
	}

	addBaseRequestFields(data, &req.BaseUpsertRequest)
	addContextFields(c, data)

	return &req, data, nil
}

// addBaseRequestFields adds common fields from BaseUpsertRequest
func addBaseRequestFields(data map[string]interface{}, req *BaseUpsertRequest) {
	if req.Locale != "" {
		data["locale"] = req.Locale
	}
	if req.DocID != "" {
		data["document_id"] = req.DocID
	}
	if req.Metadata != nil {
		data["tags"] = req.Metadata
	}

	// Add provider configurations
	if req.Converter != nil {
		data["converter_provider_id"] = req.Converter.ProviderID
		if req.Converter.Option != nil {
			data["converter_properties"] = req.Converter.Option.Properties
		}
	}
	if req.Fetcher != nil {
		data["fetcher_provider_id"] = req.Fetcher.ProviderID
		if req.Fetcher.Option != nil {
			data["fetcher_properties"] = req.Fetcher.Option.Properties
		}
	}
	if req.Chunking != nil {
		data["chunking_provider_id"] = req.Chunking.ProviderID
		if req.Chunking.Option != nil {
			data["chunking_properties"] = req.Chunking.Option.Properties
		}
	}
	if req.Extraction != nil {
		data["extraction_provider_id"] = req.Extraction.ProviderID
		if req.Extraction.Option != nil {
			data["extraction_properties"] = req.Extraction.Option.Properties
		}
	}
}

// addContextFields adds context-specific fields like permissions, user info
func addContextFields(c *gin.Context, data map[string]interface{}) {
	// TODO: Add permission-related fields from Guard
	// Example: data["user_id"] = c.GetString("user_id")
	// Example: data["permissions"] = c.Get("permissions")
	// Example: data["tenant_id"] = c.GetString("tenant_id")
}
