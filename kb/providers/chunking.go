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
	return nil, nil
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
	return nil, nil
}

// Schema returns the schema for the semantic chunking provider
func (s *Semantic) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}
