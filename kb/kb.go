package kb

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/graphrag"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"

	// Register the built-in providers
	_ "github.com/yaoapp/yao/kb/providers"

	// Import the kb types
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// Instance is the GraphRag instance
var Instance types.GraphRag = nil

// KnowledgeBase is the Knowledge Base instance
type KnowledgeBase struct {
	Config *kbtypes.Config // Knowledge Base configuration
	*graphrag.GraphRag
}

// Load loads the GraphRag instance
func Load(appConfig config.Config) (*KnowledgeBase, error) {

	configPath := filepath.Join("kb", "kb.yao")
	exists, err := application.App.Exists(configPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		log.Warn("[Knowledge Base] kb.yao file not found, skip loading knowledge base")
		return nil, nil
	}

	// Parse the configuration
	var config kbtypes.Config
	raw, err := application.App.Read(filepath.Join("kb", "kb.yao"))
	if err != nil {
		return nil, err
	}

	err = application.Parse("kb.yao", raw, &config)
	if err != nil {
		return nil, err
	}

	// Set global configurations for providers to use
	kbtypes.SetGlobalPDF(config.PDF)
	kbtypes.SetGlobalFFmpeg(config.FFmpeg)

	// Create the GraphRag config
	graphRagConfig, err := config.GraphRagConfig()
	if err != nil {
		return nil, err
	}

	// Create the GraphRag instance
	graphRag, err := graphrag.New(graphRagConfig)
	if err != nil {
		return nil, err
	}

	// Set the instance
	instance := &KnowledgeBase{Config: &config, GraphRag: graphRag}

	// Set the instance to the global variable
	Instance = instance
	return instance, nil
}

// GetProviders returns all providers
func GetProviders(typ string, ids []string, locale string) ([]kbtypes.Provider, error) {
	if Instance == nil {
		return nil, fmt.Errorf("knowledge base not initialized")
	}

	// Get the providers from the instance
	knowledgeBase, ok := Instance.(*KnowledgeBase)
	if !ok {
		return nil, fmt.Errorf("knowledge base not initialized")
	}

	// Get the configuration
	conf := knowledgeBase.Config
	if conf == nil {
		return nil, fmt.Errorf("configuration not found")
	}

	providers := []*kbtypes.Provider{}
	switch typ {
	case "chunking":
		providers = conf.Chunkings

	case "converter":
		providers = conf.Converters

	case "embedding":
		providers = conf.Embeddings

	case "extractor":
		providers = conf.Extractors

	case "fetcher":
		providers = conf.Fetchers

	case "searcher":
		providers = conf.Searchers

	case "reranker":
		providers = conf.Rerankers

	case "vote":
		providers = conf.Votes

	case "weight":
		providers = conf.Weights

	case "score":
		providers = conf.Scores

	default:
		return nil, fmt.Errorf("invalid provider type: %s", typ)

	}

	// Filter empty ids
	filteredIds := []string{}
	for _, id := range ids {
		if id != "" {
			filteredIds = append(filteredIds, id)
		}
	}

	// Filter the providers by ids
	filteredProviders := []kbtypes.Provider{}
	for _, provider := range providers {
		if len(filteredIds) == 0 || slices.Contains(ids, provider.ID) {
			filteredProviders = append(filteredProviders, *provider)
		}
	}
	return filteredProviders, nil
}

// GetProvider returns a provider by id
func GetProvider(typ string, id string) (*kbtypes.Provider, error) {
	if Instance == nil {
		return nil, fmt.Errorf("knowledge base not initialized")
	}

	knowledgeBase, ok := Instance.(*KnowledgeBase)
	if !ok {
		return nil, fmt.Errorf("knowledge base not initialized")
	}

	conf := knowledgeBase.Config
	if conf == nil {
		return nil, fmt.Errorf("configuration not found")
	}

	providers := []*kbtypes.Provider{}
	switch typ {
	case "chunking":
		providers = conf.Chunkings

	case "converter":
		providers = conf.Converters

	case "embedding":
		providers = conf.Embeddings

	case "extractor":
		providers = conf.Extractors

	case "fetcher":
		providers = conf.Fetchers

	case "searcher":
		providers = conf.Searchers

	case "reranker":
		providers = conf.Rerankers

	case "vote":
		providers = conf.Votes

	case "weight":
		providers = conf.Weights

	case "score":
		providers = conf.Scores

	default:
		return nil, fmt.Errorf("invalid provider type: %s", typ)
	}

	// Find the provider by id
	for _, provider := range providers {
		if provider.ID == id {
			return provider, nil
		}
	}

	return nil, fmt.Errorf("provider %s not found", id)
}
