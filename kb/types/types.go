package types

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

// Global shared configuration variables
var (
	// GlobalPDF holds the global PDF configuration
	GlobalPDF *PDFConfig
	// GlobalFFmpeg holds the global FFmpeg configuration
	GlobalFFmpeg *FFmpegConfig
)

// SetGlobalPDF sets the global PDF configuration
func SetGlobalPDF(config *PDFConfig) {
	GlobalPDF = config
}

// SetGlobalFFmpeg sets the global FFmpeg configuration
func SetGlobalFFmpeg(config *FFmpegConfig) {
	GlobalFFmpeg = config
}

// GetGlobalPDF returns the global PDF configuration
func GetGlobalPDF() *PDFConfig {
	return GlobalPDF
}

// GetGlobalFFmpeg returns the global FFmpeg configuration
func GetGlobalFFmpeg() *FFmpegConfig {
	return GlobalFFmpeg
}

// Config is the configuration for the Knowledge Base
type Config struct {
	// Vector Database configuration (Required)
	Vector VectorConfig `json:"vector" yaml:"vector"`

	// Graph Database configuration (Optional, if not set, graph feature will be disabled)
	Graph *GraphConfig `json:"graph,omitempty" yaml:"graph,omitempty"`

	// KV store name (Optional with default value)
	Store string `json:"store,omitempty" yaml:"store,omitempty"` // Default: "__yao.kb.store"

	// Bind Collection Model
	CollectionModel string `json:"collection_model,omitempty" yaml:"collection_model,omitempty"` // Default: "__yao.kb.collection"

	// Bind Document Model
	DocumentModel string `json:"document_model,omitempty" yaml:"document_model,omitempty"` // Default: "__yao.kb.document"

	// PDF parser configuration (Optional)
	PDF *PDFConfig `json:"pdf,omitempty" yaml:"pdf,omitempty"`

	// FFmpeg configuration (Optional)
	FFmpeg *FFmpegConfig `json:"ffmpeg,omitempty" yaml:"ffmpeg,omitempty"`

	// File uploader configuration (Optional)
	Uploader string `json:"uploader,omitempty" yaml:"uploader,omitempty"` // Default: "__yao.attachment"

	// Concurrency limits for task processing (Optional)
	Limits *LimitsConfig `json:"limits,omitempty" yaml:"limits,omitempty"`

	// Multi-language provider configurations (loaded from directories)
	Providers *ProviderConfig `json:"-"` // Loaded from provider directories, not serialized

	// Feature flags (computed during parsing, not serialized)
	Features Features `json:"-"`
}

// ProviderConfig holds providers organized by language
type ProviderConfig struct {
	// Provider configurations by language (e.g., "en", "zh-cn")
	Chunkings   map[string][]*Provider `json:"-"` // Text splitting providers by language
	Embeddings  map[string][]*Provider `json:"-"` // Text vectorization providers by language
	Converters  map[string][]*Provider `json:"-"` // File processing converters by language
	Extractions map[string][]*Provider `json:"-"` // Entity and relationship extractions by language
	Fetchers    map[string][]*Provider `json:"-"` // File fetchers by language
	Searchers   map[string][]*Provider `json:"-"` // Search providers by language
	Rerankers   map[string][]*Provider `json:"-"` // Reranking providers by language
	Votes       map[string][]*Provider `json:"-"` // Voting providers by language
	Weights     map[string][]*Provider `json:"-"` // Weighting providers by language
	Scores      map[string][]*Provider `json:"-"` // Scoring providers by language
}

// VectorConfig represents vector database configuration
type VectorConfig struct {
	Driver string                 `json:"driver" yaml:"driver"` // Required, currently only support "qdrant"
	Config map[string]interface{} `json:"config" yaml:"config"` // Driver-specific configuration
}

// GraphConfig represents graph database configuration
type GraphConfig struct {
	Driver           string                 `json:"driver" yaml:"driver"`                                           // Required, currently only support "neo4j"
	Config           map[string]interface{} `json:"config" yaml:"config"`                                           // Driver-specific configuration
	SeparateDatabase bool                   `json:"separate_database,omitempty" yaml:"separate_database,omitempty"` // Optional, for neo4j enterprise edition only
}

// PDFConfig represents PDF parser configuration
type PDFConfig struct {
	ConvertTool string `json:"convert_tool" yaml:"convert_tool"` // Required, pdftoppm/pdf2image/convert(imagemagick)
	ToolPath    string `json:"tool_path" yaml:"tool_path"`       // Required, path to the tool
}

// FFmpegConfig represents FFmpeg configuration
type FFmpegConfig struct {
	FFmpegPath   string `json:"ffmpeg_path" yaml:"ffmpeg_path"`                         // Required, path to ffmpeg
	FFprobePath  string `json:"ffprobe_path" yaml:"ffprobe_path"`                       // Required, path to ffprobe
	EnableGPU    bool   `json:"enable_gpu,omitempty" yaml:"enable_gpu,omitempty"`       // Optional, default false
	GPUIndex     int    `json:"gpu_index,omitempty" yaml:"gpu_index,omitempty"`         // GPU index (-1 means auto detect)
	MaxProcesses int    `json:"max_processes,omitempty" yaml:"max_processes,omitempty"` // Optional, -1 means max cpu cores
	MaxThreads   int    `json:"max_threads,omitempty" yaml:"max_threads,omitempty"`     // Optional, -1 means max cpu threads
}

// LimitsConfig represents concurrency limits configuration
type LimitsConfig struct {
	Job        *QueueLimit `json:"job,omitempty" yaml:"job,omitempty"`               // Job queue limits
	Chunking   *QueueLimit `json:"chunking,omitempty" yaml:"chunking,omitempty"`     // Chunking limits
	Embedding  *QueueLimit `json:"embedding,omitempty" yaml:"embedding,omitempty"`   // Embedding limits
	Converter  *QueueLimit `json:"converter,omitempty" yaml:"converter,omitempty"`   // Converter limits
	Extraction *QueueLimit `json:"extraction,omitempty" yaml:"extraction,omitempty"` // Extraction limits
	Fetcher    *QueueLimit `json:"fetcher,omitempty" yaml:"fetcher,omitempty"`       // Fetcher limits
	Searcher   *QueueLimit `json:"searcher,omitempty" yaml:"searcher,omitempty"`     // Searcher limits
	Reranker   *QueueLimit `json:"reranker,omitempty" yaml:"reranker,omitempty"`     // Reranker limits
	Vote       *QueueLimit `json:"vote,omitempty" yaml:"vote,omitempty"`             // Vote limits
	Weight     *QueueLimit `json:"weight,omitempty" yaml:"weight,omitempty"`         // Weight limits
	Score      *QueueLimit `json:"score,omitempty" yaml:"score,omitempty"`           // Score limits
}

// QueueLimit represents queue and concurrency limits
type QueueLimit struct {
	MaxConcurrent int `json:"max_concurrent,omitempty" yaml:"max_concurrent,omitempty"` // Maximum concurrent operations
	QueueSize     int `json:"queue_size,omitempty" yaml:"queue_size,omitempty"`         // Queue size (0 means unlimited)
}

// Provider represents a service provider configuration (chunking, embedding, converter, extraction, fetcher, searcher, etc.)
type Provider struct {
	ID          string            `json:"id" yaml:"id"`                               // Required, unique id for the provider
	Label       string            `json:"label" yaml:"label"`                         // Required, label for the provider, for display
	Description string            `json:"description" yaml:"description"`             // Required, description for the provider, for display
	Default     bool              `json:"default,omitempty" yaml:"default,omitempty"` // Optional, default is false, if true, will be used as the default provider
	Options     []*ProviderOption `json:"options" yaml:"options"`                     // Available preset provider options
}

// ProviderOption represents an option for a provider
type ProviderOption struct {
	Label       string                 `json:"label" yaml:"label"`                         // Required, label for the option, for display
	Value       string                 `json:"value" yaml:"value"`                         // Required, unique value for the option
	Description string                 `json:"description" yaml:"description"`             // Required, description for the option, for display
	Default     bool                   `json:"default,omitempty" yaml:"default,omitempty"` // Optional, default is false, if true, will be used as the default option
	Properties  map[string]interface{} `json:"properties" yaml:"properties"`               // Required, properties for the option
}

// ProviderSchema defines the unified schema for a provider's properties (data + UI in one)
type ProviderSchema struct {
	ID          string                     `json:"id" yaml:"id"`                                       // Provider ID this schema applies to
	Title       string                     `json:"title,omitempty" yaml:"title,omitempty"`             // Optional title for the schema
	Description string                     `json:"description,omitempty" yaml:"description,omitempty"` // Optional description
	Properties  map[string]*PropertySchema `json:"properties" yaml:"properties"`                       // Property definitions
	Required    []string                   `json:"required,omitempty" yaml:"required,omitempty"`       // Required property names
}

// PropertySchema defines both data structure and UI configuration for a single property
type PropertySchema struct {
	// Data Structure
	Type        PropertyType  `json:"type" yaml:"type"`                                   // Data type for this field
	Title       string        `json:"title,omitempty" yaml:"title,omitempty"`             // Short label displayed near the field
	Description string        `json:"description,omitempty" yaml:"description,omitempty"` // Helper text describing the field usage
	Default     interface{}   `json:"default,omitempty" yaml:"default,omitempty"`         // Default value applied when undefined
	Enum        []interface{} `json:"enum,omitempty" yaml:"enum,omitempty"`               // Enumerated options for select-like inputs (can be flat options or grouped options)

	// Validation
	Required       bool           `json:"required,omitempty" yaml:"required,omitempty"`             // Whether the field is required
	RequiredFields []string       `json:"requiredFields,omitempty" yaml:"requiredFields,omitempty"` // For object types: names of nested properties that are required when the object is provided
	MinLength      *int           `json:"minLength,omitempty" yaml:"minLength,omitempty"`           // Minimum length for string values
	MaxLength      *int           `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`           // Maximum length for string values
	Pattern        *string        `json:"pattern,omitempty" yaml:"pattern,omitempty"`               // Regex pattern a string must satisfy
	Minimum        *float64       `json:"minimum,omitempty" yaml:"minimum,omitempty"`               // Minimum numeric value (inclusive)
	Maximum        *float64       `json:"maximum,omitempty" yaml:"maximum,omitempty"`               // Maximum numeric value (inclusive)
	ErrorMessages  *ErrorMessages `json:"errorMessages,omitempty" yaml:"errorMessages,omitempty"`   // Error message templates with variable interpolation support

	// UI Configuration
	Component   string `json:"component,omitempty" yaml:"component,omitempty"`     // Input component to render from inputs/
	Placeholder string `json:"placeholder,omitempty" yaml:"placeholder,omitempty"` // Placeholder text for inputs
	Help        string `json:"help,omitempty" yaml:"help,omitempty"`               // Additional help text below the field
	Order       int    `json:"order,omitempty" yaml:"order,omitempty"`             // Field ordering index within a group/form
	Hidden      bool   `json:"hidden,omitempty" yaml:"hidden,omitempty"`           // If true, the field is not displayed
	Disabled    bool   `json:"disabled,omitempty" yaml:"disabled,omitempty"`       // If true, the field is disabled (non-interactive)
	ReadOnly    bool   `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`       // If true, the field is read-only
	Width       string `json:"width,omitempty" yaml:"width,omitempty"`             // Visual width hint (e.g. full, half, third, quarter)
	Group       string `json:"group,omitempty" yaml:"group,omitempty"`             // Grouping name for organizing fields in UI

	// Object / Array
	Properties map[string]*PropertySchema `json:"properties,omitempty" yaml:"properties,omitempty"` // Nested properties when type === 'object' (use with component: 'Nested')
	Items      *PropertySchema            `json:"items,omitempty" yaml:"items,omitempty"`           // Array item schema when type === 'array' (use with component: 'Items')
}

// ErrorMessages represents error message templates with variable interpolation support
type ErrorMessages struct {
	Required  string `json:"required,omitempty" yaml:"required,omitempty"`
	MinLength string `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength string `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Minimum   string `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum   string `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	Custom    string `json:"custom,omitempty" yaml:"custom,omitempty"`
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

// EnumOption represents a single option in an enumerated field
type EnumOption struct {
	Label       string `json:"label" yaml:"label"`                                 // Display label shown in UI
	Value       string `json:"value" yaml:"value"`                                 // Underlying machine value submitted/saved
	Description string `json:"description,omitempty" yaml:"description,omitempty"` // Optional helper text for this option
	Default     bool   `json:"default,omitempty" yaml:"default,omitempty"`         // Whether this option is the default selection
}

// OptionGroup represents a group of related options with a group label
type OptionGroup struct {
	GroupLabel string       `json:"groupLabel" yaml:"groupLabel"` // Group label displayed as section header
	Options    []EnumOption `json:"options" yaml:"options"`       // Array of options within this group
}

// RawConfig is an alias for Config to enable custom JSON marshaling/unmarshaling
type RawConfig Config
