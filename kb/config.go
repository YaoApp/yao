package kb

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
	// Parse environment variables in config
	resolvedConfig, err := c.resolveEnvVars(c.Vector.Config)
	if err != nil {
		return types.VectorStoreConfig{}, err
	}

	// Convert resolved config to types.VectorStoreConfig via JSON
	jsonData, err := json.Marshal(resolvedConfig)
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
	// Parse environment variables in config
	resolvedConfig, err := c.resolveEnvVars(c.Graph.Config)
	if err != nil {
		return types.GraphStoreConfig{}, err
	}

	// Convert resolved config to types.GraphStoreConfig via JSON
	jsonData, err := json.Marshal(resolvedConfig)
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

// UnmarshalJSON implements json.Unmarshaler interface
func (c *Config) UnmarshalJSON(data []byte) error {
	// Use alias type to avoid infinite recursion
	raw := (*RawConfig)(c)
	if err := json.Unmarshal(data, raw); err != nil {
		return err
	}

	// Compute features after parsing
	c.Features = c.ComputeFeatures()

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
	for _, provider := range c.Converters {
		converterMap[provider.ID] = true
	}

	features.PlainText = true // Plain text is always supported as a basic feature
	features.OfficeDocuments = converterMap["__yao.office"]
	features.OCRProcessing = converterMap["__yao.ocr"]
	features.AudioTranscript = converterMap["__yao.whisper"]
	features.ImageAnalysis = converterMap["__yao.vision"]

	// Advanced features
	features.EntityExtraction = len(c.Extractors) > 0
	features.WebFetching = len(c.Fetchers) > 0
	features.CustomSearch = len(c.Searchers) > 0
	features.ResultReranking = len(c.Rerankers) > 0
	features.SegmentVoting = len(c.Votes) > 0
	features.SegmentWeighting = len(c.Weights) > 0
	features.SegmentScoring = len(c.Scores) > 0

	return features
}
