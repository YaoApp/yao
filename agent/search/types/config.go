package types

// Config represents the complete search configuration
type Config struct {
	Web      *WebConfig      `json:"web,omitempty" yaml:"web,omitempty"`
	KB       *KBConfig       `json:"kb,omitempty" yaml:"kb,omitempty"`
	DB       *DBConfig       `json:"db,omitempty" yaml:"db,omitempty"`
	Keyword  *KeywordConfig  `json:"keyword,omitempty" yaml:"keyword,omitempty"`
	QueryDSL *QueryDSLConfig `json:"querydsl,omitempty" yaml:"querydsl,omitempty"`
	Rerank   *RerankConfig   `json:"rerank,omitempty" yaml:"rerank,omitempty"`
	Citation *CitationConfig `json:"citation,omitempty" yaml:"citation,omitempty"`
	Weights  *WeightsConfig  `json:"weights,omitempty" yaml:"weights,omitempty"`
	Options  *OptionsConfig  `json:"options,omitempty" yaml:"options,omitempty"`
}

// WebConfig for web search settings
// Note: uses.web determines the mode (builtin/agent/mcp)
// Provider is only used when uses.web = "builtin"
type WebConfig struct {
	Provider   string `json:"provider,omitempty" yaml:"provider,omitempty"`       // "tavily", "serper", or "serpapi" (for builtin mode)
	APIKeyEnv  string `json:"api_key_env,omitempty" yaml:"api_key_env,omitempty"` // Environment variable for API key
	MaxResults int    `json:"max_results,omitempty" yaml:"max_results,omitempty"` // Max results (default: 10)
	Engine     string `json:"engine,omitempty" yaml:"engine,omitempty"`           // Search engine for SerpAPI: "google", "bing", "baidu", "yandex", etc. (default: "google")
}

// KBConfig for knowledge base search settings
type KBConfig struct {
	Collections []string `json:"collections,omitempty" yaml:"collections,omitempty"` // Default collections
	Threshold   float64  `json:"threshold,omitempty" yaml:"threshold,omitempty"`     // Similarity threshold (default: 0.7)
	Graph       bool     `json:"graph,omitempty" yaml:"graph,omitempty"`             // Enable GraphRAG (default: false)
}

// DBConfig for database search settings
type DBConfig struct {
	Models     []string `json:"models,omitempty" yaml:"models,omitempty"`           // Default models
	MaxResults int      `json:"max_results,omitempty" yaml:"max_results,omitempty"` // Max results (default: 20)
}

// KeywordConfig for keyword extraction
type KeywordConfig struct {
	MaxKeywords int    `json:"max_keywords,omitempty" yaml:"max_keywords,omitempty"` // Max keywords (default: 10)
	Language    string `json:"language,omitempty" yaml:"language,omitempty"`         // "auto", "en", "zh", etc.
}

// KeywordOptions for keyword extraction (runtime options)
type KeywordOptions struct {
	MaxKeywords int    `json:"max_keywords,omitempty"`
	Language    string `json:"language,omitempty"`
}

// QueryDSLConfig for QueryDSL generation from natural language
type QueryDSLConfig struct {
	Strict bool `json:"strict,omitempty" yaml:"strict,omitempty"` // Fail if generation fails (default: false)
}

// RerankConfig for reranking
type RerankConfig struct {
	TopN int `json:"top_n,omitempty" yaml:"top_n,omitempty"` // Return top N (default: 10)
}

// CitationConfig for citation format
type CitationConfig struct {
	Format           string `json:"format,omitempty" yaml:"format,omitempty"`                         // Default: "#ref:{id}"
	AutoInjectPrompt bool   `json:"auto_inject_prompt,omitempty" yaml:"auto_inject_prompt,omitempty"` // Auto-inject prompt (default: true)
	CustomPrompt     string `json:"custom_prompt,omitempty" yaml:"custom_prompt,omitempty"`           // Custom prompt template
}

// WeightsConfig for source weighting
type WeightsConfig struct {
	User float64 `json:"user,omitempty" yaml:"user,omitempty"` // User-provided (default: 1.0)
	Hook float64 `json:"hook,omitempty" yaml:"hook,omitempty"` // Hook results (default: 0.8)
	Auto float64 `json:"auto,omitempty" yaml:"auto,omitempty"` // Auto search (default: 0.6)
}

// OptionsConfig for search behavior
type OptionsConfig struct {
	SkipThreshold int `json:"skip_threshold,omitempty" yaml:"skip_threshold,omitempty"` // Skip auto search if user provides >= N results
}

// GetWeight returns the weight for a source type
func (c *Config) GetWeight(source SourceType) float64 {
	if c == nil || c.Weights == nil {
		return getDefaultWeight(source)
	}
	switch source {
	case SourceUser:
		if c.Weights.User > 0 {
			return c.Weights.User
		}
		return 1.0
	case SourceHook:
		if c.Weights.Hook > 0 {
			return c.Weights.Hook
		}
		return 0.8
	case SourceAuto:
		if c.Weights.Auto > 0 {
			return c.Weights.Auto
		}
		return 0.6
	default:
		return 0.6
	}
}

// getDefaultWeight returns default weight for a source type
func getDefaultWeight(source SourceType) float64 {
	switch source {
	case SourceUser:
		return 1.0
	case SourceHook:
		return 0.8
	case SourceAuto:
		return 0.6
	default:
		return 0.6
	}
}
