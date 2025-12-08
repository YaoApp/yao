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
	"github.com/yaoapp/yao/kb/api"

	// Register the built-in providers
	_ "github.com/yaoapp/yao/kb/providers"

	// Import the kb types
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// Instance is the GraphRag instance
var Instance types.GraphRag = nil

// API is the Knowledge Base API instance
var API api.API = nil

// KnowledgeBase is the Knowledge Base instance
type KnowledgeBase struct {
	Config    *kbtypes.Config         // Knowledge Base configuration
	Providers *kbtypes.ProviderConfig // Multi-language provider configurations
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

	// Load providers from directories first
	providers, err := kbtypes.LoadProviders("kb")
	if err != nil {
		return nil, err
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

	// Assign providers to config
	config.Providers = providers

	// Compute features after both config and providers are loaded
	config.Features = config.ComputeFeatures()

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
	instance := &KnowledgeBase{Config: &config, Providers: providers, GraphRag: graphRag}

	// Set the instance to the global variable
	Instance = instance

	// Create and set the API instance
	API = api.NewAPI(graphRag, &config, providers)

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

	// Default locale to "en" if empty
	if locale == "" {
		locale = "en"
	}

	// Get providers for the requested type and language
	providers := knowledgeBase.Providers.GetProviders(typ, locale)

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

// GetProvider returns a provider by id with default language "en"
func GetProvider(typ string, id string) (*kbtypes.Provider, error) {
	return GetProviderWithLanguage(typ, id, "en")
}

// GetProviderWithLanguage returns a provider by id, type, and language
func GetProviderWithLanguage(typ string, id string, locale string) (*kbtypes.Provider, error) {
	if Instance == nil {
		return nil, fmt.Errorf("knowledge base not initialized")
	}

	knowledgeBase, ok := Instance.(*KnowledgeBase)
	if !ok {
		return nil, fmt.Errorf("knowledge base not initialized")
	}

	// Default locale to "en" if empty
	if locale == "" {
		locale = "en"
	}

	return knowledgeBase.Providers.GetProvider(typ, id, locale)
}

// GetConfig returns the knowledge base configuration
func GetConfig() (*kbtypes.Config, error) {
	if Instance == nil {
		return nil, fmt.Errorf("knowledge base not initialized")
	}

	knowledgeBase, ok := Instance.(*KnowledgeBase)
	if !ok {
		return nil, fmt.Errorf("knowledge base not initialized")
	}

	return knowledgeBase.Config, nil
}
