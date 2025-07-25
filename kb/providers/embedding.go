package providers

import (
	"github.com/yaoapp/gou/graphrag/embedding"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// OpenAI is an embedding provider
type OpenAI struct{}

// Fastembed is an embedding provider
type Fastembed struct{}

func init() {
	factory.Embeddings["__yao.openai"] = &OpenAI{}
	factory.Embeddings["__yao.fastembed"] = &Fastembed{}
}

// === OpenAI ===

// Make creates an OpenAI embedding provider
func (o *OpenAI) Make(option *kbtypes.ProviderOption) (types.Embedding, error) {
	return embedding.NewOpenai(embedding.OpenaiOptions{})
}

// Schema returns the schema for the OpenAI embedding provider
func (o *OpenAI) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}

// === Fastembed ===

// Make creates a Fastembed embedding provider
func (f *Fastembed) Make(option *kbtypes.ProviderOption) (types.Embedding, error) {
	return embedding.NewFastEmbed(embedding.FastEmbedOptions{})
}

// Schema returns the schema for the Fastembed embedding provider
func (f *Fastembed) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}
