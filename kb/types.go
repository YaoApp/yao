package kb

// Features represents the available features based on current configuration
type Features struct {
	// Core features
	GraphDatabase   bool // Graph database support (neo4j)
	PDFProcessing   bool // PDF text extraction
	VideoProcessing bool // Video/audio processing (ffmpeg)

	// File format support (based on converters)
	PlainText       bool // Plain text files (.txt, .md)
	OfficeDocuments bool // Office documents (.docx, .pptx)
	OCRProcessing   bool // Text recognition from images and PDFs
	AudioTranscript bool // Audio transcription
	ImageAnalysis   bool // Image content analysis

	// Advanced features
	EntityExtraction bool // Entity and relationship extraction
	WebFetching      bool // Web URL fetching
	CustomSearch     bool // Custom search providers
	ResultReranking  bool // Search result reranking
	SegmentVoting    bool // Segment voting system
	SegmentWeighting bool // Segment weighting system
	SegmentScoring   bool // Segment scoring system
}

// Config is the configuration for the Knowledge Base
type Config struct {
	// Vector Database configuration (Required)
	Vector VectorConfig `json:"vector" yaml:"vector"`

	// Graph Database configuration (Optional, if not set, graph feature will be disabled)
	Graph *GraphConfig `json:"graph,omitempty" yaml:"graph"`

	// KV store name (Optional with default value)
	Store string `json:"store,omitempty" yaml:"store"` // Default: "__yao.kb.store"

	// PDF parser configuration (Optional)
	PDF *PDFConfig `json:"pdf,omitempty" yaml:"pdf"`

	// FFmpeg configuration (Optional)
	FFmpeg *FFmpegConfig `json:"ffmpeg,omitempty" yaml:"ffmpeg"`

	// Concurrency limits for task processing (Optional)
	Limits *LimitsConfig `json:"limits,omitempty" yaml:"limits"`

	// Provider configurations
	Chunkings  []*Provider `json:"chunkings" yaml:"chunkings"`             // Text splitting providers (Required - at least one)
	Embeddings []*Provider `json:"embeddings" yaml:"embeddings"`           // Text vectorization providers (Required - at least one)
	Converters []*Provider `json:"converters,omitempty" yaml:"converters"` // File processing converters (Optional)
	Extractors []*Provider `json:"extractors,omitempty" yaml:"extractors"` // Entity and relationship extractors (Optional)
	Fetchers   []*Provider `json:"fetchers,omitempty" yaml:"fetchers"`     // File fetchers (Optional)
	Searchers  []*Provider `json:"searchers,omitempty" yaml:"searchers"`   // Search providers (Optional)
	Rerankers  []*Provider `json:"rerankers,omitempty" yaml:"rerankers"`   // Reranking providers (Optional)
	Votes      []*Provider `json:"votes,omitempty" yaml:"votes"`           // Voting providers (Optional)
	Weights    []*Provider `json:"weights,omitempty" yaml:"weights"`       // Weighting providers (Optional)
	Scores     []*Provider `json:"scores,omitempty" yaml:"scores"`         // Scoring providers (Optional)

	// Feature flags (computed during parsing, not serialized)
	Features Features `json:"-"`
}

// VectorConfig represents vector database configuration
type VectorConfig struct {
	Driver string                 `json:"driver" yaml:"driver"` // Required, currently only support "qdrant"
	Config map[string]interface{} `json:"config" yaml:"config"` // Driver-specific configuration
}

// GraphConfig represents graph database configuration
type GraphConfig struct {
	Driver           string                 `json:"driver" yaml:"driver"`                                 // Required, currently only support "neo4j"
	Config           map[string]interface{} `json:"config" yaml:"config"`                                 // Driver-specific configuration
	SeparateDatabase bool                   `json:"separate_database,omitempty" yaml:"separate_database"` // Optional, for neo4j enterprise edition only
}

// PDFConfig represents PDF parser configuration
type PDFConfig struct {
	ConvertTool string `json:"convert_tool" yaml:"convert_tool"` // Required, pdftoppm/pdf2image/convert(imagemagick)
	ToolPath    string `json:"tool_path" yaml:"tool_path"`       // Required, path to the tool
}

// FFmpegConfig represents FFmpeg configuration
type FFmpegConfig struct {
	FFmpegPath   string `json:"ffmpeg_path" yaml:"ffmpeg_path"`               // Required, path to ffmpeg
	FFprobePath  string `json:"ffprobe_path" yaml:"ffprobe_path"`             // Required, path to ffprobe
	EnableGPU    bool   `json:"enable_gpu,omitempty" yaml:"enable_gpu"`       // Optional, default false
	GPUIndex     int    `json:"gpu_index,omitempty" yaml:"gpu_index"`         // GPU index (-1 means auto detect)
	MaxProcesses int    `json:"max_processes,omitempty" yaml:"max_processes"` // Optional, -1 means max cpu cores
	MaxThreads   int    `json:"max_threads,omitempty" yaml:"max_threads"`     // Optional, -1 means max cpu threads
}

// LimitsConfig represents concurrency limits configuration
type LimitsConfig struct {
	Job       *QueueLimit `json:"job,omitempty" yaml:"job"`             // Job queue limits
	Chunking  *QueueLimit `json:"chunking,omitempty" yaml:"chunking"`   // Chunking limits
	Embedding *QueueLimit `json:"embedding,omitempty" yaml:"embedding"` // Embedding limits
	Converter *QueueLimit `json:"converter,omitempty" yaml:"converter"` // Converter limits
	Extractor *QueueLimit `json:"extractor,omitempty" yaml:"extractor"` // Extractor limits
	Fetcher   *QueueLimit `json:"fetcher,omitempty" yaml:"fetcher"`     // Fetcher limits
	Searcher  *QueueLimit `json:"searcher,omitempty" yaml:"searcher"`   // Searcher limits
	Reranker  *QueueLimit `json:"reranker,omitempty" yaml:"reranker"`   // Reranker limits
	Vote      *QueueLimit `json:"vote,omitempty" yaml:"vote"`           // Vote limits
	Weight    *QueueLimit `json:"weight,omitempty" yaml:"weight"`       // Weight limits
	Score     *QueueLimit `json:"score,omitempty" yaml:"score"`         // Score limits
}

// QueueLimit represents queue and concurrency limits
type QueueLimit struct {
	MaxConcurrent int `json:"max_concurrent,omitempty" yaml:"max_concurrent"` // Maximum concurrent operations
	QueueSize     int `json:"queue_size,omitempty" yaml:"queue_size"`         // Queue size (0 means unlimited)
}

// Provider represents a service provider configuration (chunking, embedding, converter, extractor, fetcher, searcher, etc.)
type Provider struct {
	ID          string            `json:"id" yaml:"id"`                     // Required, unique id for the provider
	Label       string            `json:"label" yaml:"label"`               // Required, label for the provider, for display
	Description string            `json:"description" yaml:"description"`   // Required, description for the provider, for display
	Default     bool              `json:"default,omitempty" yaml:"default"` // Optional, default is false, if true, will be used as the default provider
	Options     []*ProviderOption `json:"options" yaml:"options"`           // Available preset provider options
}

// ProviderOption represents an option for a provider
type ProviderOption struct {
	Label       string                 `json:"label" yaml:"label"`               // Required, label for the option, for display
	Value       string                 `json:"value" yaml:"value"`               // Required, unique value for the option
	Description string                 `json:"description" yaml:"description"`   // Required, description for the option, for display
	Default     bool                   `json:"default,omitempty" yaml:"default"` // Optional, default is false, if true, will be used as the default option
	Properties  map[string]interface{} `json:"properties" yaml:"properties"`     // Required, properties for the option
}

// ProviderSchema defines the unified schema for a provider's properties (data + UI in one)
type ProviderSchema struct {
	ID          string                     `json:"id" yaml:"id"`                             // Provider ID this schema applies to
	Title       string                     `json:"title,omitempty" yaml:"title"`             // Optional title for the schema
	Description string                     `json:"description,omitempty" yaml:"description"` // Optional description
	Properties  map[string]*PropertySchema `json:"properties" yaml:"properties"`             // Property definitions
	Required    []string                   `json:"required,omitempty" yaml:"required"`       // Required property names
}

// PropertySchema defines both data structure and UI configuration for a single property
type PropertySchema struct {
	// Data Structure (similar to JSON Schema)
	Type        PropertyType `json:"type" yaml:"type"`                         // Property type
	Title       string       `json:"title,omitempty" yaml:"title"`             // Display title
	Description string       `json:"description,omitempty" yaml:"description"` // Property description
	Default     interface{}  `json:"default,omitempty" yaml:"default"`         // Default value
	Enum        []EnumOption `json:"enum,omitempty" yaml:"enum"`               // Enumeration options (with labels)

	// Validation Constraints
	Required  bool     `json:"required,omitempty" yaml:"required"`   // Is this property required
	MinLength *int     `json:"minLength,omitempty" yaml:"minLength"` // String min length
	MaxLength *int     `json:"maxLength,omitempty" yaml:"maxLength"` // String max length
	Pattern   *string  `json:"pattern,omitempty" yaml:"pattern"`     // Regex pattern
	Minimum   *float64 `json:"minimum,omitempty" yaml:"minimum"`     // Number minimum
	Maximum   *float64 `json:"maximum,omitempty" yaml:"maximum"`     // Number maximum

	// UI Configuration
	Component   string `json:"component,omitempty" yaml:"component"`     // UI component type (Input, Select, Textarea, etc.)
	Placeholder string `json:"placeholder,omitempty" yaml:"placeholder"` // Input placeholder
	Help        string `json:"help,omitempty" yaml:"help"`               // Help text
	Order       int    `json:"order,omitempty" yaml:"order"`             // Display order
	Hidden      bool   `json:"hidden,omitempty" yaml:"hidden"`           // Hide field
	Disabled    bool   `json:"disabled,omitempty" yaml:"disabled"`       // Disable field
	ReadOnly    bool   `json:"readOnly,omitempty" yaml:"readOnly"`       // Read-only field
	Width       string `json:"width,omitempty" yaml:"width"`             // Field width (full, half, quarter)
	Group       string `json:"group,omitempty" yaml:"group"`             // Group name for organizing fields

	// Nested Properties (for object types)
	Properties map[string]*PropertySchema `json:"properties,omitempty" yaml:"properties"` // Object properties

	// Array Items (for array types)
	Items *PropertySchema `json:"items,omitempty" yaml:"items"` // Array items schema
}

// PropertyType represents the type of a property
type PropertyType string

// Property type constants
const (
	PropertyTypeString  PropertyType = "string"  // PropertyTypeString represents a string property
	PropertyTypeNumber  PropertyType = "number"  // PropertyTypeNumber represents a number property
	PropertyTypeInteger PropertyType = "integer" // PropertyTypeInteger represents an integer property
	PropertyTypeBoolean PropertyType = "boolean" // PropertyTypeBoolean represents a boolean property
	PropertyTypeObject  PropertyType = "object"  // PropertyTypeObject represents an object property
	PropertyTypeArray   PropertyType = "array"   // PropertyTypeArray represents an array property
)

// EnumOption represents an enumeration option with both value and display label
type EnumOption struct {
	Value string `json:"value" yaml:"value"`         // Actual value
	Label string `json:"label" yaml:"label"`         // Display label
	Desc  string `json:"desc,omitempty" yaml:"desc"` // Optional description
	Icon  string `json:"icon,omitempty" yaml:"icon"` // Optional icon
}

// RawConfig is an alias for Config to enable custom JSON marshaling/unmarshaling
type RawConfig Config
