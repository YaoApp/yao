package config

// ResolveOptions specifies identifiers for the multi-layer config lookup.
type ResolveOptions struct {
	AssistantID string // required
	ChatID      string // optional: empty skips task-config layers
	UserID      string // optional: empty skips user/team scope merge
	TeamID      string // optional: empty skips team scope merge
}

// AssistantDefaults represents Layer 0 values extracted from the assistant DSL
// (package.yao / sandbox.yao).
type AssistantDefaults struct {
	Connector string
	Runner    string
	Image     string
	MaxTurns  int
	Secrets   map[string]string
	Services  []ServiceDecl
	Skills    []string
}

// ServiceDecl declares a service exposed by the task container.
type ServiceDecl struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Public   bool   `json:"public"`
}

// ScheduleConfig defines task scheduling parameters.
type ScheduleConfig struct {
	Enabled       bool     `json:"enabled"`
	Mode          string   `json:"mode"`
	Times         []string `json:"times,omitempty"`
	Days          []string `json:"days,omitempty"`
	IntervalValue int      `json:"interval_value,omitempty"`
	IntervalUnit  string   `json:"interval_unit,omitempty"`
	Timezone      string   `json:"timezone,omitempty"`
	StartDate     string   `json:"start_date,omitempty"`
	EndDate       string   `json:"end_date,omitempty"`
}
