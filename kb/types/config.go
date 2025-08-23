package types

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"github.com/yaoapp/gou/graphrag"
	"github.com/yaoapp/gou/graphrag/graph/neo4j"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/vector/qdrant"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/log"
)

// Config parses the Knowledge Base configuration

// ParseConfigFromJSON parses config from JSON bytes
func ParseConfigFromJSON(data []byte) (*Config, error) {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ParseConfigFromFile parses config from JSON file
func ParseConfigFromFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseConfigFromJSON(data)
}

// ToJSON converts config to JSON bytes
func (c *Config) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// GraphRagConfig converts KB config to GraphRag config
func (c *Config) GraphRagConfig() (*graphrag.Config, error) {

	config := &graphrag.Config{
		Logger: log.StandardLogger(),
		System: "__yao_kb_system", // Default system collection name
		Vector: nil,
		Graph:  nil,
		Store:  nil,
	}

	// Configure Vector Store (required)
	vectorStore, err := c.createVectorStore()
	if err != nil {
		return nil, err
	}
	config.Vector = vectorStore

	// Configure Graph Store (optional)
	if c.Graph != nil {
		graphStore, err := c.createGraphStore()
		if err != nil {
			return nil, err
		}
		config.Graph = graphStore
	}

	// Configure Store
	storeName := c.getStoreName()
	kvStore, err := store.Get(storeName)
	if err != nil {
		return nil, err
	}
	config.Store = kvStore
	return config, nil
}

// getStoreName returns the store name, using default if not configured
func (c *Config) getStoreName() string {
	if c.Store != "" {
		return c.Store
	}
	return "__yao.kb.store"
}

// createVectorStore creates a vector store from config
func (c *Config) createVectorStore() (types.VectorStore, error) {
	switch c.Vector.Driver {
	case "qdrant":
		// Convert config to VectorStoreConfig
		vectorConfig, err := c.toVectorStoreConfig()
		if err != nil {
			return nil, err
		}
		return qdrant.NewStoreWithConfig(vectorConfig), nil
	default:
		return nil, nil
	}
}

// createGraphStore creates a graph store from config
func (c *Config) createGraphStore() (types.GraphStore, error) {
	switch c.Graph.Driver {
	case "neo4j":
		// Convert config to GraphStoreConfig
		graphConfig, err := c.toGraphStoreConfig()
		if err != nil {
			return nil, err
		}
		return neo4j.NewStoreWithConfig(graphConfig), nil
	default:
		return nil, nil
	}
}

// toVectorStoreConfig converts the vector config to VectorStoreConfig
func (c *Config) toVectorStoreConfig() (types.VectorStoreConfig, error) {
	// Environment variables are already resolved during parsing
	configCopy := make(map[string]interface{})
	for k, v := range c.Vector.Config {
		configCopy[k] = v
	}

	// Ensure host and port are in ExtraParams for Qdrant
	if _, exists := configCopy["extra_params"]; !exists {
		configCopy["extra_params"] = make(map[string]interface{})
	}

	extraParams := configCopy["extra_params"].(map[string]interface{})

	// Map host field
	if host, exists := configCopy["host"]; exists {
		extraParams["host"] = host
	}

	// Map port field
	if port, exists := configCopy["port"]; exists {
		extraParams["port"] = port
	}

	// Convert config to types.VectorStoreConfig via JSON
	jsonData, err := json.Marshal(configCopy)
	if err != nil {
		return types.VectorStoreConfig{}, err
	}

	var config types.VectorStoreConfig
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return types.VectorStoreConfig{}, err
	}

	return config, nil
}

// toGraphStoreConfig converts the graph config to GraphStoreConfig
func (c *Config) toGraphStoreConfig() (types.GraphStoreConfig, error) {
	// Environment variables are already resolved during parsing
	configCopy := make(map[string]interface{})
	for k, v := range c.Graph.Config {
		configCopy[k] = v
	}

	// Map field names to match GraphStoreConfig structure
	if url, exists := configCopy["url"]; exists {
		configCopy["database_url"] = url
		delete(configCopy, "url") // Remove the original field
	}

	// Ensure DriverConfig exists and map username/password into it
	if _, exists := configCopy["driver_config"]; !exists {
		configCopy["driver_config"] = make(map[string]interface{})
	}

	driverConfig := configCopy["driver_config"].(map[string]interface{})

	// Map username field to DriverConfig
	if username, exists := configCopy["username"]; exists {
		driverConfig["username"] = username
		delete(configCopy, "username") // Remove from top level
	}

	// Map password field to DriverConfig
	if password, exists := configCopy["password"]; exists {
		driverConfig["password"] = password
		delete(configCopy, "password") // Remove from top level
	}

	// Convert config to types.GraphStoreConfig via JSON
	jsonData, err := json.Marshal(configCopy)
	if err != nil {
		return types.GraphStoreConfig{}, err
	}

	var config types.GraphStoreConfig
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return types.GraphStoreConfig{}, err
	}

	return config, nil
}

// resolveEnvVars resolves environment variables in configuration values
func (c *Config) resolveEnvVars(config map[string]interface{}) (map[string]interface{}, error) {
	resolved := make(map[string]interface{})

	for key, value := range config {
		switch v := value.(type) {
		case string:
			resolved[key] = c.parseEnvVar(v)
		case map[string]interface{}:
			// Recursively resolve nested maps
			nestedResolved, err := c.resolveEnvVars(v)
			if err != nil {
				return nil, err
			}
			resolved[key] = nestedResolved
		default:
			resolved[key] = value
		}
	}

	return resolved, nil
}

// parseEnvVar parses environment variable pattern $ENV.VAR_NAME
func (c *Config) parseEnvVar(value string) string {
	// Simple pattern to match $ENV.VAR_NAME
	envPattern := regexp.MustCompile(`\$ENV\.([A-Za-z_][A-Za-z0-9_]*)`)

	return envPattern.ReplaceAllStringFunc(value, func(match string) string {
		// Extract variable name (remove $ENV. prefix)
		varName := strings.TrimPrefix(match, "$ENV.")

		// Get environment variable value
		if envValue := os.Getenv(varName); envValue != "" {
			return envValue
		}

		// Return original if environment variable is not set
		return match
	})
}

// resolveAllEnvVars resolves environment variables in all configuration sections
func (c *Config) resolveAllEnvVars() error {
	// Resolve Vector config
	if c.Vector.Config != nil {
		resolved, err := c.resolveEnvVars(c.Vector.Config)
		if err != nil {
			return err
		}
		c.Vector.Config = resolved
	}

	// Resolve Graph config
	if c.Graph != nil && c.Graph.Config != nil {
		resolved, err := c.resolveEnvVars(c.Graph.Config)
		if err != nil {
			return err
		}
		c.Graph.Config = resolved
	}

	// Resolve Provider options (if they contain env vars)
	if err := c.resolveProviderEnvVars(); err != nil {
		return err
	}

	return nil
}

// resolveProviderEnvVars resolves environment variables in provider configurations
func (c *Config) resolveProviderEnvVars() error {
	if c.Providers == nil {
		return nil
	}

	// Resolve env vars for all provider types and languages
	providerMaps := []map[string][]*Provider{
		c.Providers.Chunkings, c.Providers.Embeddings, c.Providers.Converters, c.Providers.Extractions,
		c.Providers.Fetchers, c.Providers.Searchers, c.Providers.Rerankers, c.Providers.Votes,
		c.Providers.Weights, c.Providers.Scores,
	}

	for _, providerMap := range providerMaps {
		if providerMap == nil {
			continue
		}
		for _, providers := range providerMap {
			for _, provider := range providers {
				for _, option := range provider.Options {
					if option.Properties != nil {
						resolved, err := c.resolveEnvVars(option.Properties)
						if err != nil {
							return err
						}
						option.Properties = resolved
					}
				}
			}
		}
	}

	return nil
}

// UnmarshalJSON implements json.Unmarshaler interface
func (c *Config) UnmarshalJSON(data []byte) error {
	// Use alias type to avoid infinite recursion
	raw := (*RawConfig)(c)
	if err := json.Unmarshal(data, raw); err != nil {
		return err
	}

	// Resolve environment variables immediately after parsing
	if err := c.resolveAllEnvVars(); err != nil {
		return err
	}

	// Set default uploader if not configured
	if c.Uploader == "" {
		c.Uploader = "__yao.attachment"
	}

	// Set default collection model if not configured
	if c.CollectionModel == "" {
		c.CollectionModel = "__yao.kb.collection"
	}

	// Set default document model if not configured
	if c.DocumentModel == "" {
		c.DocumentModel = "__yao.kb.document"
	}

	// Note: Features will be computed later after providers are loaded
	return nil
}

// MarshalJSON implements json.Marshaler interface
func (c *Config) MarshalJSON() ([]byte, error) {
	// Use alias type for standard JSON marshaling (Features field is ignored)
	raw := (*RawConfig)(c)
	return json.Marshal(raw)
}

// ComputeFeatures calculates available features based on current configuration
func (c *Config) ComputeFeatures() Features {
	features := Features{}

	// Core features
	features.GraphDatabase = c.Graph != nil
	features.PDFProcessing = c.PDF != nil
	features.VideoProcessing = c.FFmpeg != nil

	// File format support (based on converters)
	converterMap := make(map[string]bool)
	if c.Providers != nil && c.Providers.Converters != nil {
		// Check all languages for converter availability
		for _, providers := range c.Providers.Converters {
			for _, provider := range providers {
				converterMap[provider.ID] = true
			}
		}
	}

	features.PlainText = true // Plain text is always supported as a basic feature
	features.OfficeDocuments = converterMap["__yao.office"]
	features.OCRProcessing = converterMap["__yao.ocr"]
	features.AudioTranscript = converterMap["__yao.whisper"]
	features.ImageAnalysis = converterMap["__yao.vision"]

	// Advanced features
	if c.Providers != nil {
		features.EntityExtraction = c.hasProvidersInAnyLanguage(c.Providers.Extractions)
		features.WebFetching = c.hasProvidersInAnyLanguage(c.Providers.Fetchers)
		features.CustomSearch = c.hasProvidersInAnyLanguage(c.Providers.Searchers)
		features.ResultReranking = c.hasProvidersInAnyLanguage(c.Providers.Rerankers)
		features.SegmentVoting = c.hasProvidersInAnyLanguage(c.Providers.Votes)
		features.SegmentWeighting = c.hasProvidersInAnyLanguage(c.Providers.Weights)
		features.SegmentScoring = c.hasProvidersInAnyLanguage(c.Providers.Scores)
	}

	return features
}

// hasProvidersInAnyLanguage checks if there are providers available in any language
func (c *Config) hasProvidersInAnyLanguage(providerMap map[string][]*Provider) bool {
	if providerMap == nil {
		return false
	}
	for _, providers := range providerMap {
		if len(providers) > 0 {
			return true
		}
	}
	return false
}
