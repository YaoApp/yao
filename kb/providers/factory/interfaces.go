package factory

import (
	"github.com/yaoapp/gou/graphrag/types"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// Chunking is a factory for chunking providers
type Chunking interface {
	Make(option *kbtypes.ProviderOption) (types.Chunking, error)
	Options(option *kbtypes.ProviderOption) (*types.ChunkingOptions, error)
	Schema
}

// Converter is a factory for converter providers
type Converter interface {
	Make(option *kbtypes.ProviderOption) (types.Converter, error)
	AutoDetect(filename, contentTypes string) (bool, int, error)
	Schema
}

// Embedding is a factory for embedding providers
type Embedding interface {
	Make(options *kbtypes.ProviderOption) (types.Embedding, error)
	Schema
}

// Extraction is a factory for extraction providers
type Extraction interface {
	Make(option *kbtypes.ProviderOption) (types.Extraction, error)
	Schema
}

// Fetcher is a factory for fetcher providers
type Fetcher interface {
	Make(option *kbtypes.ProviderOption) (types.Fetcher, error)
	Schema
}

// Schema interface for providers
type Schema interface {
	Schema(provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error)
}
