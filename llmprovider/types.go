package llmprovider

// Identity abstracts a caller's user/team context for scope-aware lookups.
// Implemented by oauthTypes.AuthorizedInfo and any struct with UserID/TeamID.
type Identity interface {
	GetUserID() string
	GetTeamID() string
}

// Provider represents a configured LLM provider (one vendor connection with multiple models).
// Fields align with the frontend ProviderConfig interface.
type Provider struct {
	Key         string         `json:"key"`
	ConnectorID string         `json:"connector_id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	APIURL      string         `json:"api_url"`
	APIKey      string         `json:"api_key"`
	Models      []ModelInfo    `json:"models"`
	Enabled     bool           `json:"enabled"`
	Status      string         `json:"status"`
	IsCustom    bool           `json:"is_custom,omitempty"`
	PresetKey   string         `json:"preset_key,omitempty"`
	RequireKey  bool           `json:"require_key"`
	Source      ProviderSource `json:"source"`
	Owner       ProviderOwner  `json:"owner"`
}

// ModelInfo describes a single model within a provider.
// Fields align with the frontend ModelInfo interface.
type ModelInfo struct {
	ID              string                 `json:"id" yaml:"id"`
	Model           string                 `json:"model,omitempty" yaml:"model,omitempty"`
	Name            string                 `json:"name" yaml:"name"`
	Capabilities    []string               `json:"capabilities" yaml:"capabilities"`
	Enabled         bool                   `json:"enabled" yaml:"enabled"`
	MaxInputTokens  int                    `json:"max_input_tokens,omitempty" yaml:"max_input_tokens,omitempty"`
	MaxOutputTokens int                    `json:"max_output_tokens,omitempty" yaml:"max_output_tokens,omitempty"`
	Options         map[string]interface{} `json:"options,omitempty" yaml:"options,omitempty"`
}

// ProviderOwner identifies who owns a provider.
type ProviderOwner struct {
	Type   string `json:"type"`
	TeamID string `json:"team_id,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

// ProviderSource distinguishes dynamic (registry-created) from builtin (DSL-loaded) providers.
type ProviderSource string

const (
	ProviderSourceDynamic ProviderSource = "dynamic"
	ProviderSourceBuiltIn ProviderSource = "builtin"
	ProviderSourceAll     ProviderSource = "all"
)

// ProviderFilter specifies criteria for listing providers.
type ProviderFilter struct {
	Owner        *ProviderOwner
	Enabled      *bool
	Source       ProviderSource // defaults to "dynamic" when zero-value
	Type         *string
	PresetKey    *string
	Capabilities []string // AND filter: provider matches if any model satisfies all
	Keyword      string
}

// ProviderPreset is a static UI-only template for creating providers.
// Fields align with the frontend ProviderPreset interface.
type ProviderPreset struct {
	Key           string      `json:"key" yaml:"key"`
	Name          string      `json:"name" yaml:"name"`
	Locale        string      `json:"locale,omitempty" yaml:"locale,omitempty"`
	Type          string      `json:"type" yaml:"type"`
	APIURL        string      `json:"api_url" yaml:"api_url"`
	RequireKey    bool        `json:"require_key" yaml:"require_key"`
	IsCloud       bool        `json:"is_cloud,omitempty" yaml:"is_cloud,omitempty"`
	URLEditable   bool        `json:"url_editable,omitempty" yaml:"url_editable,omitempty"`
	DefaultModels []ModelInfo `json:"default_models" yaml:"default_models"`
}

// ProviderTestResult holds the outcome of a provider connectivity test.
type ProviderTestResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// RoleTarget identifies a provider and model for a given role.
type RoleTarget struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}
