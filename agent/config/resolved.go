package config

// SecretInfo holds per-key secret metadata and status for API responses.
// Used by both Agent settings and Task settings — the output format is identical.
type SecretInfo struct {
	HasValue    bool   `json:"has_value"`
	Predefined  bool   `json:"predefined"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Multiline   bool   `json:"multiline,omitempty"`
	Source      string `json:"source,omitempty"`
}

// Resolved holds the final merged configuration across all layers.
type Resolved struct {
	Runner   string                `json:"runner,omitempty"`
	Runners  []string              `json:"runners,omitempty"`
	Model    string                `json:"model,omitempty"`
	Image    string                `json:"image,omitempty"`
	Timeout  string                `json:"timeout,omitempty"`
	MaxTurns int                   `json:"max_turns,omitempty"`
	Secrets  map[string]SecretInfo `json:"secrets,omitempty"`
	Services []ServiceDecl         `json:"services,omitempty"`
	Skills   []string              `json:"skills,omitempty"`
	Schedule *ScheduleConfig       `json:"schedule,omitempty"`

	ResolvedFrom map[string]string `json:"_resolved_from,omitempty"`
}
