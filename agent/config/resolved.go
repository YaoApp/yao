package config

// Resolved holds the final merged configuration across all layers.
type Resolved struct {
	Runner   string            `json:"runner,omitempty"`
	Runners  []string          `json:"runners,omitempty"`
	Model    string            `json:"model,omitempty"`
	Image    string            `json:"image,omitempty"`
	Timeout  string            `json:"timeout,omitempty"`
	MaxTurns int               `json:"max_turns,omitempty"`
	Secrets  map[string]string `json:"secrets,omitempty"`
	Services []ServiceDecl     `json:"services,omitempty"`
	Skills   []string          `json:"skills,omitempty"`
	Schedule *ScheduleConfig   `json:"schedule,omitempty"`

	ResolvedFrom map[string]string `json:"_resolved_from,omitempty"`
}
