package llm

import "time"

// ---------------------------------------------------------------------------
// API — wire protocol identifiers (mirrors PiG ai.API)
// ---------------------------------------------------------------------------

// API identifies which provider-API protocol a model uses on the wire.
// Values match PiG ai.API exactly; the string base type keeps forward-compat.
type API string

const (
	// OpenAI family
	APIOpenAICompletions API = "openai-completions"
	APIOpenAIResponses   API = "openai-responses"
	APIAzureResponses    API = "azure-openai-responses"
	APICodexResponses    API = "openai-codex-responses"

	// Anthropic
	APIAnthropicMessages API = "anthropic-messages"

	// Google
	APIGoogleGenerativeAI API = "google-generative-ai"
	APIGoogleVertex       API = "google-vertex"

	// AWS
	APIBedrockConverse API = "bedrock-converse-stream"

	// Other
	APIMistral    API = "mistral-conversations"
	APIPiMessages API = "pi-messages"
)

// ImageAPI identifies an image-generation provider wire protocol (= ai.ImageAPI).
type ImageAPI string

const (
	ImageAPIOpenRouter ImageAPI = "openrouter-images"
)

// DecisionAPI identifies a decision/classifier provider wire protocol.
// PiG uses ai.ClassifierAPI; we use "Decision" naming.
type DecisionAPI string

const (
	DecisionAPITypesafeSystemOne    DecisionAPI = "typesafe-system-one"
	DecisionAPICFWorkersAISystemOne DecisionAPI = "cloudflare-workers-ai-system-one"
	DecisionAPILlamaCppClassify     DecisionAPI = "llama-cpp-classify"
)

// ---------------------------------------------------------------------------
// ModelType — model kind discriminant (mirrors PiG ai.ModelType)
// ---------------------------------------------------------------------------

// ModelType is a discriminant for model kind.
// "decision" is our name; PiG uses "classifier" — the only value mapping
// in the entire system (handled by toAIModelType in convert.go).
type ModelType string

const (
	ModelTypeChat     ModelType = "chat"     // conversation model (default)
	ModelTypeImage    ModelType = "image"    // image generation (DALL-E, FLUX, Seedream)
	ModelTypeDecision ModelType = "decision" // structured decision (choice/score)
)

// ---------------------------------------------------------------------------
// ProviderID — vendor identity (mirrors PiG ai.ProviderMetadata.ProviderID)
// ---------------------------------------------------------------------------

// ProviderID identifies an LLM vendor for UI presets, icons, and
// PiG static catalog lookup ("provider/model-id").
type ProviderID string

const (
	ProviderOpenAI     ProviderID = "openai"
	ProviderAnthropic  ProviderID = "anthropic"
	ProviderGoogle     ProviderID = "google"
	ProviderAzure      ProviderID = "azure"
	ProviderBedrock    ProviderID = "bedrock"
	ProviderDeepSeek   ProviderID = "deepseek"
	ProviderMoonshot   ProviderID = "moonshotai"
	ProviderMistral    ProviderID = "mistral"
	ProviderOpenRouter ProviderID = "openrouter"
	ProviderTogether   ProviderID = "together"
	ProviderGroq       ProviderID = "groq"
)

// ---------------------------------------------------------------------------
// ThinkingLevel — reasoning intensity (mirrors PiG ai.ThinkingLevel)
// ---------------------------------------------------------------------------

// ThinkingLevel controls extended-thinking depth.
type ThinkingLevel string

const (
	ThinkingOff     ThinkingLevel = "off"
	ThinkingNone    ThinkingLevel = ThinkingOff // backward-compatible alias
	ThinkingMinimal ThinkingLevel = "minimal"
	ThinkingLow     ThinkingLevel = "low"
	ThinkingMedium  ThinkingLevel = "medium"
	ThinkingHigh    ThinkingLevel = "high"
	ThinkingXHigh   ThinkingLevel = "xhigh"
	ThinkingMax     ThinkingLevel = "max"
)

// ThinkingLevelMap maps a thinking level to the wire-format string the
// provider expects. A nil *string means provider-specific disable.
// Mirrors PiG ai.ThinkingLevelMap.
type ThinkingLevelMap map[ThinkingLevel]*string

// SamplingParamsByThinkingLevel holds sampling overrides keyed by thinking level.
// Mirrors PiG ai.SamplingParamsByThinkingLevel.
type SamplingParamsByThinkingLevel = map[ThinkingLevel]map[string]any

// ---------------------------------------------------------------------------
// Scope — three-level configuration cascade
// ---------------------------------------------------------------------------

// Scope identifies the configuration ownership level.
type Scope string

const (
	ScopeSystem Scope = "system"
	ScopeTeam   Scope = "team"
	ScopeUser   Scope = "user"
)

// ---------------------------------------------------------------------------
// Provider — one LLM backend connection
// (corresponds to PiG ai.Provider + ai.ProviderMetadata)
// ---------------------------------------------------------------------------

// Provider represents an LLM service provider connection.
// Corresponds to Yao model __yao.llm.provider (table llm_provider).
type Provider struct {
	ID         int        `json:"id" db:"id"`
	Name       string     `json:"name" db:"name"`
	ProviderID ProviderID `json:"provider_id" db:"provider_id"`
	API        API        `json:"api" db:"api"`
	Protocols  []API      `json:"protocols,omitempty" db:"protocols"`
	BaseURL    string     `json:"base_url" db:"base_url"`
	APIKey     string     `json:"api_key,omitempty" db:"api_key"`
	Headers    JSONMap    `json:"headers,omitempty" db:"headers"`
	Compat     JSONMap    `json:"compat,omitempty" db:"compat"`
	Enabled    bool       `json:"enabled" db:"enabled"`
	Status     string     `json:"status" db:"status"`
	PresetKey  string     `json:"preset_key,omitempty" db:"preset_key"`
	Scope      Scope      `json:"scope" db:"scope"`
	TeamID     string     `json:"team_id,omitempty" db:"team_id"`
	UserID     string     `json:"user_id,omitempty" db:"user_id"`
	Priority   int        `json:"priority" db:"priority"`
	Metadata   JSONMap    `json:"metadata,omitempty" db:"metadata"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// HeadersMap returns Headers as map[string]string for PiG compatibility.
// Returns nil when Headers is nil or empty.
func (p *Provider) HeadersMap() map[string]string {
	if len(p.Headers) == 0 {
		return nil
	}
	out := make(map[string]string, len(p.Headers))
	for k, v := range p.Headers {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Model — one concrete model under a Provider
// (corresponds to PiG ai.Model + ai.GeneratedModel)
// ---------------------------------------------------------------------------

// Model represents an LLM model configuration.
// Corresponds to Yao model __yao.llm.model (table llm_model).
// Field names align with PiG ai.Model / ai.GeneratedModel one-to-one.
type Model struct {
	// Identity
	ID          int       `json:"id" db:"id"`
	ProviderRef int       `json:"provider_ref" db:"provider_ref"`
	ModelID     string    `json:"model_id" db:"model_id"`
	Name        string    `json:"name" db:"name"`
	Type        ModelType `json:"type,omitempty" db:"type"`
	ModelAPI    string    `json:"model_api,omitempty" db:"model_api"`
	Family      string    `json:"family" db:"family"`
	Enabled     bool      `json:"enabled" db:"enabled"`

	// Capabilities (= ai.ModelCapabilities)
	ContextWindow   int  `json:"context_window" db:"context_window"`
	MaxOutputTokens int  `json:"max_output_tokens" db:"max_output_tokens"`
	SupportsImages  bool `json:"supports_images" db:"supports_images"`
	SupportsToolUse bool `json:"supports_tool_use" db:"supports_tool_use"`
	Reasoning       bool `json:"reasoning" db:"reasoning"`

	// Extended capabilities (Yao-specific, not modeled by PiG)
	Audio                 bool   `json:"audio,omitempty" db:"audio"`
	STT                   bool   `json:"stt,omitempty" db:"stt"`
	Streaming             bool   `json:"streaming,omitempty" db:"streaming"`
	JSON                  bool   `json:"json_mode,omitempty" db:"json_mode"`
	Embedding             bool   `json:"embedding,omitempty" db:"embedding"`
	ImageGeneration       bool   `json:"image_generation,omitempty" db:"image_generation"`
	ImageEditing          string `json:"image_editing,omitempty" db:"image_editing"`
	OCR                   bool   `json:"ocr,omitempty" db:"ocr"`
	TemperatureAdjustable bool   `json:"temperature_adjustable,omitempty" db:"temperature_adjustable"`

	// Modalities (= ai.Model.Input / ai.ImageModel.Output)
	Input  []string `json:"input" db:"input"`
	Output []string `json:"output,omitempty" db:"output"`

	// Reasoning (= ai.Model thinking fields)
	ThinkingLevelMap              ThinkingLevelMap              `json:"thinking_level_map,omitempty" db:"thinking_level_map"`
	MaxThinking                   ThinkingLevel                 `json:"max_thinking,omitempty" db:"max_thinking"`
	SamplingParams                map[string]any                `json:"sampling_params,omitempty" db:"sampling_params"`
	SamplingParamsByThinkingLevel SamplingParamsByThinkingLevel `json:"sampling_params_by_level,omitempty" db:"sampling_params_by_level"`

	// Cost (= ai.ModelCapabilities pricing fields)
	InputCostPer1M      float64    `json:"input_cost_per_1m,omitempty" db:"input_cost_per_1m"`
	OutputCostPer1M     float64    `json:"output_cost_per_1m,omitempty" db:"output_cost_per_1m"`
	CacheReadCostPer1M  float64    `json:"cache_read_cost_per_1m,omitempty" db:"cache_read_cost_per_1m"`
	CacheWriteCostPer1M float64    `json:"cache_write_cost_per_1m,omitempty" db:"cache_write_cost_per_1m"`
	CostTiers           []CostTier `json:"cost_tiers,omitempty" db:"cost_tiers"`

	// Input limits (= ai.Model.InputLimits)
	InputLimits *ModelInputLimits `json:"input_limits,omitempty" db:"input_limits"`

	// Cache (= ai.Model.PromptCache)
	PromptCache ModelPromptCache `json:"prompt_cache,omitempty" db:"prompt_cache"`

	// Capability tags (= ai.GeneratedModel.Capabilities)
	Capabilities []string `json:"capabilities" db:"capabilities"`

	// Model-level compat override (merged with Provider.Compat at runtime)
	Compat JSONMap `json:"compat,omitempty" db:"compat"`

	// Catalog metadata (= ai.GeneratedModel.Lab)
	Lab string `json:"lab,omitempty" db:"lab"`

	// Metadata
	Priority  int       `json:"priority" db:"priority"`
	Metadata  JSONMap   `json:"metadata,omitempty" db:"metadata"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ---------------------------------------------------------------------------
// RoleBinding — role-to-model mapping with three-level override
// ---------------------------------------------------------------------------

// RoleBinding maps a role name to a specific provider+model combination
// within a scope level. Corresponds to Yao model __yao.llm.role_binding.
type RoleBinding struct {
	ID          int       `json:"id" db:"id"`
	Role        string    `json:"role" db:"role"`
	ProviderRef int       `json:"provider_ref" db:"provider_ref"`
	ModelID     string    `json:"model_id" db:"model_id"`
	Scope       Scope     `json:"scope" db:"scope"`
	TeamID      string    `json:"team_id,omitempty" db:"team_id"`
	UserID      string    `json:"user_id,omitempty" db:"user_id"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Role name constants matching the existing llmprovider role system.
const (
	RoleDefault   = "default"
	RoleHeavy     = "heavy"
	RoleLight     = "light"
	RoleVision    = "vision"
	RoleAudio     = "audio"
	RoleEmbedding = "embedding"
	RoleDecision  = "decision"
)

// ---------------------------------------------------------------------------
// Nested value types — JSON tag alignment with PiG ai types
// ---------------------------------------------------------------------------

// CostTier is a volume-based pricing tier (= ai.CostTier).
type CostTier struct {
	InputTokensAbove    int     `json:"inputTokensAbove"`
	InputCostPer1M      float64 `json:"input"`
	OutputCostPer1M     float64 `json:"output"`
	CacheReadCostPer1M  float64 `json:"cacheRead"`
	CacheWriteCostPer1M float64 `json:"cacheWrite"`
}

// ModelInputLimits carries provider input limits (= ai.ModelInputLimits).
type ModelInputLimits struct {
	MaxRequestBytes int                    `json:"maxRequestBytes,omitempty"`
	Images          *ModelImageInputLimits `json:"images,omitempty"`
}

// ModelImageInputLimits describes per-image constraints (= ai.ModelImageInputLimits).
type ModelImageInputLimits struct {
	Resize        *ModelImageResizeOptions `json:"resize,omitempty"`
	MaxPerMessage int                      `json:"maxPerMessage,omitempty"`
	MaxPerRequest int                      `json:"maxPerRequest,omitempty"`
}

// ModelImageResizeOptions controls cache-safe image resizing (= ai.ModelImageResizeOptions).
type ModelImageResizeOptions struct {
	MaxWidth    int `json:"maxWidth,omitempty"`
	MaxHeight   int `json:"maxHeight,omitempty"`
	MaxBytes    int `json:"maxBytes,omitempty"`
	JPEGQuality int `json:"jpegQuality,omitempty"`
}

// ModelPromptCache gives cache lifetime in seconds per retention tier (= ai.ModelPromptCache).
type ModelPromptCache = map[string]int

// JSONMap is a generic JSON object used for Compat, Headers, and Metadata fields.
type JSONMap = map[string]interface{}

// ---------------------------------------------------------------------------
// DSH cordis.yml configuration types
// ---------------------------------------------------------------------------

// PiAiRoute configures one dsh-llm-pi-ai provider route in cordis.yml.
// Defined here (not in dsh/) because it is a pure LLM config description
// with no dependency on the runner runtime.
type PiAiRoute struct {
	Name            string            `json:"name"`            // provider route key (e.g. "yao-primary", "yao-heavy")
	API             string            `json:"api"`             // wire protocol ("openai-completions" or "anthropic-messages")
	BaseURL         string            `json:"baseURL"`         // endpoint URL
	APIKeyEnv       string            `json:"apiKeyEnv"`       // env var name for the API key
	Models          []PiAiModelConfig `json:"models"`          // model catalog for this route
	Reasoning       string            `json:"reasoning"`       // pi-ai reasoning level; empty = omit
	BudgetTokens    int               `json:"budgetTokens"`    // > 0 → output thinkingBudgets
	NoDeveloperRole bool              `json:"noDeveloperRole"` // true → supportsDeveloperRole: false
	ThinkingFormat  string            `json:"thinkingFormat"`  // thinking wire format (e.g. "deepseek"); empty = let pi-ai detect
}

// PiAiModelConfig declares a model's identity and capabilities for pi-ai.
type PiAiModelConfig struct {
	ID               string            `json:"id"`               // model identifier
	Input            []string          `json:"input"`            // e.g. ["text"] or ["text", "image"]
	ContextWindow    int               `json:"contextWindow"`    // > 0 → render contextWindow
	MaxTokens        int               `json:"maxTokens"`        // > 0 → render maxTokens
	ReasoningEfforts map[string]string `json:"reasoningEfforts"` // pi-ai level→wire value; nil = omit
	DisableReasoning bool              `json:"disableReasoning"` // true → render "reasoningEfforts: false"
}
