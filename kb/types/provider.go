package types

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
)

// GetOption returns the option for a provider
func (p *Provider) GetOption(id string) (*ProviderOption, bool) {
	if p.Options == nil {
		return nil, false
	}

	for _, option := range p.Options {
		if option.Value == id {
			return option, true
		}
	}

	return nil, false
}

// GetOptionByIndex returns the option by index
func (p *Provider) GetOptionByIndex(index int) (*ProviderOption, bool) {
	if len(p.Options) <= index {
		return nil, false
	}
	return p.Options[index], true
}

// Parse parses the provider option
func (p *ProviderOption) Parse(v interface{}) error {

	raw, err := jsoniter.Marshal(p)
	if err != nil {
		return err
	}

	err = jsoniter.Unmarshal(raw, v)
	if err != nil {
		return err
	}

	return nil
}

// LoadProviders loads providers from directories with language support
func LoadProviders(basePath string) (*ProviderConfig, error) {
	config := &ProviderConfig{
		Chunkings:   make(map[string][]*Provider),
		Embeddings:  make(map[string][]*Provider),
		Converters:  make(map[string][]*Provider),
		Extractions: make(map[string][]*Provider),
		Fetchers:    make(map[string][]*Provider),
		Searchers:   make(map[string][]*Provider),
		Rerankers:   make(map[string][]*Provider),
		Votes:       make(map[string][]*Provider),
		Weights:     make(map[string][]*Provider),
		Scores:      make(map[string][]*Provider),
	}

	// Provider type directories to load
	providerTypes := []string{
		"chunkings", "embeddings", "converters", "extractions",
		"fetchers", "searchers", "rerankers", "votes", "weights", "scores",
	}

	for _, providerType := range providerTypes {
		err := loadProviderType(basePath, providerType, config)
		if err != nil {
			log.Warn("[Knowledge Base] Failed to load %s providers: %v", providerType, err)
		}
	}

	return config, nil
}

// loadProviderType loads providers for a specific type from language files
func loadProviderType(basePath, providerType string, config *ProviderConfig) error {
	providerDir := filepath.Join(basePath, providerType)

	// Check if directory exists
	exists, err := application.App.Exists(providerDir)
	if err != nil {
		return err
	}
	if !exists {
		log.Debug("[Knowledge Base] Provider directory %s not found, skipping", providerDir)
		return nil
	}

	// Use Walk to find all provider files in the provider directory
	err = application.App.Walk(providerDir, func(root, filename string, isdir bool) error {
		if isdir {
			return nil
		}

		// Skip non-yao files
		if !strings.HasSuffix(filename, ".yao") {
			return nil
		}

		// Extract language from filename (e.g., "en.yao" -> "en")
		baseName := filepath.Base(filename)
		language := strings.TrimSuffix(baseName, ".yao")

		// Load providers for this language
		providers, err := loadProvidersForLanguage(providerDir, baseName)
		if err != nil {
			log.Warn("[Knowledge Base] Failed to load %s providers for language %s: %v", providerType, language, err)
			return nil // Continue processing other files
		}

		// Store providers in the appropriate map
		switch providerType {
		case "chunkings":
			config.Chunkings[language] = providers
		case "embeddings":
			config.Embeddings[language] = providers
		case "converters":
			config.Converters[language] = providers
		case "extractions":
			config.Extractions[language] = providers
		case "fetchers":
			config.Fetchers[language] = providers
		case "searchers":
			config.Searchers[language] = providers
		case "rerankers":
			config.Rerankers[language] = providers
		case "votes":
			config.Votes[language] = providers
		case "weights":
			config.Weights[language] = providers
		case "scores":
			config.Scores[language] = providers
		}

		log.Debug("[Knowledge Base] Loaded %d %s providers for language %s", len(providers), providerType, language)
		return nil
	}, "*.yao")
	if err != nil {
		return err
	}

	return nil
}

// loadProvidersForLanguage loads providers from a specific language file
func loadProvidersForLanguage(providerDir, filename string) ([]*Provider, error) {
	filePath := filepath.Join(providerDir, filename)

	// Read the file
	data, err := application.App.Read(filePath)
	if err != nil {
		return nil, err
	}

	// Parse as array of providers
	var providers []*Provider
	err = application.Parse(filename, data, &providers)
	if err != nil {
		return nil, err
	}

	return providers, nil
}

// GetProviders returns providers for a specific type and language with fallback to "en"
func (pc *ProviderConfig) GetProviders(providerType, language string) []*Provider {
	if pc == nil {
		return []*Provider{}
	}

	var providerMap map[string][]*Provider
	switch providerType {
	case "chunking":
		providerMap = pc.Chunkings
	case "embedding":
		providerMap = pc.Embeddings
	case "converter":
		providerMap = pc.Converters
	case "extraction":
		providerMap = pc.Extractions
	case "fetcher":
		providerMap = pc.Fetchers
	case "searcher":
		providerMap = pc.Searchers
	case "reranker":
		providerMap = pc.Rerankers
	case "vote":
		providerMap = pc.Votes
	case "weight":
		providerMap = pc.Weights
	case "score":
		providerMap = pc.Scores
	default:
		return []*Provider{}
	}

	// Try to get providers for the requested language
	if providers, exists := providerMap[language]; exists && len(providers) > 0 {
		return providers
	}

	// Fallback to "en" if requested language not found
	if language != "en" {
		if providers, exists := providerMap["en"]; exists && len(providers) > 0 {
			return providers
		}
	}

	return []*Provider{}
}

// GetProvider returns a specific provider by ID, type, and language with fallback to "en"
func (pc *ProviderConfig) GetProvider(providerType, id, language string) (*Provider, error) {
	providers := pc.GetProviders(providerType, language)

	for _, provider := range providers {
		if provider.ID == id {
			return provider, nil
		}
	}

	return nil, fmt.Errorf("provider %s not found for type %s and language %s", id, providerType, language)
}
