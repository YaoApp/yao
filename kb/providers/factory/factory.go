package factory

import (
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// ProviderType is a type for provider types
type ProviderType string

const (
	// ProviderTypeChunking is a type for chunking providers
	ProviderTypeChunking ProviderType = "chunking"
	// ProviderTypeConverter is a type for converter providers
	ProviderTypeConverter ProviderType = "converter"
	// ProviderTypeEmbedding is a type for embedding providers
	ProviderTypeEmbedding ProviderType = "embedding"
	// ProviderTypeExtraction is a type for extraction providers
	ProviderTypeExtraction ProviderType = "extraction"
	// ProviderTypeFetcher is a type for fetcher providers
	ProviderTypeFetcher ProviderType = "fetcher"
)

// DetectMatch is a match for auto detect
type DetectMatch struct {
	ID       string
	Priority int
}

// Chunkings is a map of chunking providers
var Chunkings = map[string]Chunking{}

// Converters is a map of converter providers
var Converters = map[string]Converter{}

// Embeddings is a map of embedding providers
var Embeddings = map[string]Embedding{}

// Extractions is a map of extraction providers
var Extractions = map[string]Extraction{}

// Fetchers is a map of fetcher providers
var Fetchers = map[string]Fetcher{}

// === Chunking API ===

// MakeChunking creates a new chunking provider
func MakeChunking(id string, option *kbtypes.ProviderOption) (types.Chunking, error) {
	chunking, ok := Chunkings[id]
	if !ok {
		return nil, fmt.Errorf("chunking provider %s not found", id)
	}
	return chunking.Make(option)
}

// ChunkingOptions returns the options for a chunking provider
func ChunkingOptions(id string, option *kbtypes.ProviderOption) (*types.ChunkingOptions, error) {
	chunking, ok := Chunkings[id]
	if !ok {
		return nil, fmt.Errorf("chunking provider %s not found", id)
	}
	return chunking.Options(option)
}

// === Converter API ===

// MakeConverter creates a new converter provider
func MakeConverter(id string, option *kbtypes.ProviderOption) (types.Converter, error) {
	converter, ok := Converters[id]
	if !ok {
		return nil, fmt.Errorf("converter provider %s not found", id)
	}
	return converter.Make(option)
}

// AutoDetectConverter detects the converter based on the filename and content types
// return matched, id, error
func AutoDetectConverter(filename, contentType string) (bool, string, error) {
	var highestPriority int = 0
	var highestID string = ""
	for id, converter := range Converters {
		ok, priority, err := converter.AutoDetect(filename, contentType)
		if err != nil {
			continue
		}
		if ok && priority > highestPriority {
			highestPriority = priority
			highestID = id
		}
	}
	return highestID != "", highestID, nil
}

// === Embedding API ===

// MakeEmbedding creates a new embedding provider
func MakeEmbedding(id string, option *kbtypes.ProviderOption) (types.Embedding, error) {
	embedding, ok := Embeddings[id]
	if !ok {
		return nil, fmt.Errorf("embedding provider %s not found", id)
	}
	return embedding.Make(option)
}

// === Extraction API ===

// MakeExtraction creates a new extraction provider
func MakeExtraction(id string, option *kbtypes.ProviderOption) (types.Extraction, error) {
	extraction, ok := Extractions[id]
	if !ok {
		return nil, fmt.Errorf("extraction provider %s not found", id)
	}
	return extraction.Make(option)
}

// === Fetcher API ===

// MakeFetcher creates a new fetcher provider
func MakeFetcher(id string, option *kbtypes.ProviderOption) (types.Fetcher, error) {
	fetcher, ok := Fetchers[id]
	if !ok {
		return nil, fmt.Errorf("fetcher provider %s not found", id)
	}
	return fetcher.Make(option)
}

// === Schema API ===

// GetSchema returns the schema for a provider
func GetSchema(typ ProviderType, provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error) {
	var schema Schema = nil
	var exists bool = false
	switch typ {
	case ProviderTypeChunking:
		schema, exists = Chunkings[provider.ID]
	case ProviderTypeConverter:
		schema, exists = Converters[provider.ID]
	case ProviderTypeEmbedding:
		schema, exists = Embeddings[provider.ID]
	case ProviderTypeExtraction:
		schema, exists = Extractions[provider.ID]
	case ProviderTypeFetcher:
		schema, exists = Fetchers[provider.ID]
	}
	if !exists {
		return nil, fmt.Errorf("%s provider %s not found", typ, provider.ID)
	}
	return schema.Schema(provider, locale)
}
