package types

import (
	"encoding/json"
	"time"
)

// Config - robot_config in __yao.member
type Config struct {
	Triggers  *Triggers            `json:"triggers,omitempty"`
	Clock     *Clock               `json:"clock,omitempty"`
	Identity  *Identity            `json:"identity"`
	Quota     *Quota               `json:"quota,omitempty"`
	KB        *KB                  `json:"kb,omitempty"`    // shared knowledge base (same as assistant)
	DB        *DB                  `json:"db,omitempty"`    // shared database (same as assistant)
	Learn     *Learn               `json:"learn,omitempty"` // learning config for private KB
	Resources *Resources           `json:"resources,omitempty"`
	Delivery  *DeliveryPreferences `json:"delivery,omitempty"` // delivery preferences (see robot.go)
	Events    []Event              `json:"events,omitempty"`
	Executor  *ExecutorConfig      `json:"executor,omitempty"` // executor mode settings
}

// ExecutorConfig - executor settings
type ExecutorConfig struct {
	Mode        ExecutorMode `json:"mode,omitempty"`         // standard | dryrun | sandbox
	MaxDuration string       `json:"max_duration,omitempty"` // max execution time (e.g., "30m")
}

// GetMode returns the executor mode (default: standard)
func (e *ExecutorConfig) GetMode() ExecutorMode {
	if e == nil || e.Mode == "" {
		return ExecutorStandard
	}
	return e.Mode
}

// GetMaxDuration returns the max duration (default: 30m)
func (e *ExecutorConfig) GetMaxDuration() time.Duration {
	if e == nil || e.MaxDuration == "" {
		return 30 * time.Minute
	}
	d, err := time.ParseDuration(e.MaxDuration)
	if err != nil {
		return 30 * time.Minute
	}
	return d
}

// Validate validates the config
func (c *Config) Validate() error {
	if c.Identity == nil || c.Identity.Role == "" {
		return ErrMissingIdentity
	}
	if c.Clock != nil {
		if err := c.Clock.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Triggers - trigger enable/disable
type Triggers struct {
	Clock     *TriggerSwitch `json:"clock,omitempty"`
	Intervene *TriggerSwitch `json:"intervene,omitempty"`
	Event     *TriggerSwitch `json:"event,omitempty"`
}

// TriggerSwitch - trigger enable/disable switch
type TriggerSwitch struct {
	Enabled bool     `json:"enabled"`
	Actions []string `json:"actions,omitempty"` // for intervene
}

// IsEnabled checks if trigger is enabled (default: true)
func (t *Triggers) IsEnabled(typ TriggerType) bool {
	if t == nil {
		return true
	}
	switch typ {
	case TriggerClock:
		return t.Clock == nil || t.Clock.Enabled
	case TriggerHuman:
		return t.Intervene == nil || t.Intervene.Enabled
	case TriggerEvent:
		return t.Event == nil || t.Event.Enabled
	}
	return false
}

// Clock - when to wake up
type Clock struct {
	Mode    ClockMode `json:"mode"`              // times | interval | daemon
	Times   []string  `json:"times,omitempty"`   // ["09:00", "14:00"]
	Days    []string  `json:"days,omitempty"`    // ["Mon", "Tue"] or ["*"]
	Every   string    `json:"every,omitempty"`   // "30m", "1h"
	TZ      string    `json:"tz,omitempty"`      // "Asia/Shanghai"
	Timeout string    `json:"timeout,omitempty"` // "30m"
}

// Validate validates clock config
func (c *Clock) Validate() error {
	switch c.Mode {
	case ClockTimes:
		if len(c.Times) == 0 {
			return ErrClockTimesEmpty
		}
	case ClockInterval:
		if c.Every == "" {
			return ErrClockIntervalEmpty
		}
	case ClockDaemon:
		// no extra validation
	default:
		return ErrClockModeInvalid
	}
	return nil
}

// GetTimeout returns parsed timeout duration
func (c *Clock) GetTimeout() time.Duration {
	if c.Timeout == "" {
		return 30 * time.Minute // default
	}
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 30 * time.Minute
	}
	return d
}

// GetLocation returns timezone location
func (c *Clock) GetLocation() *time.Location {
	if c.TZ == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(c.TZ)
	if err != nil {
		return time.Local
	}
	return loc
}

// Identity - who is this robot
type Identity struct {
	Role   string   `json:"role"`
	Duties []string `json:"duties,omitempty"`
	Rules  []string `json:"rules,omitempty"`
}

// Quota - concurrency limits
type Quota struct {
	Max      int `json:"max"`      // max running (default: 2)
	Queue    int `json:"queue"`    // queue size (default: 10)
	Priority int `json:"priority"` // 1-10 (default: 5)
}

// GetMax returns max with default
func (q *Quota) GetMax() int {
	if q == nil || q.Max <= 0 {
		return 2
	}
	return q.Max
}

// GetQueue returns queue size with default
func (q *Quota) GetQueue() int {
	if q == nil || q.Queue <= 0 {
		return 10
	}
	return q.Queue
}

// GetPriority returns priority with default
func (q *Quota) GetPriority() int {
	if q == nil || q.Priority <= 0 {
		return 5
	}
	return q.Priority
}

// KB - knowledge base config (same as assistant, from store/types)
// Shared KB collections accessible by this robot
type KB struct {
	Collections []string               `json:"collections,omitempty"` // KB collection IDs
	Options     map[string]interface{} `json:"options,omitempty"`
}

// DB - database config (same as assistant, from store/types)
// Shared database models accessible by this robot
type DB struct {
	Models  []string               `json:"models,omitempty"` // database model names
	Options map[string]interface{} `json:"options,omitempty"`
}

// Learn - learning config for robot's private KB
// Private KB is auto-created: robot_{team_id}_{member_id}_kb
type Learn struct {
	On    bool     `json:"on"`
	Types []string `json:"types,omitempty"` // execution, feedback, insight
	Keep  int      `json:"keep,omitempty"`  // days, 0 = forever
}

// Resources - available agents and tools
type Resources struct {
	Phases map[Phase]string `json:"phases,omitempty"` // phase -> agent ID
	Agents []string         `json:"agents,omitempty"`
	MCP    []MCPConfig      `json:"mcp,omitempty"`
}

// GetPhaseAgent returns agent ID for phase (default: __yao.{phase})
func (r *Resources) GetPhaseAgent(phase Phase) string {
	if r != nil && r.Phases != nil {
		if id, ok := r.Phases[phase]; ok && id != "" {
			return id
		}
	}
	return "__yao." + string(phase)
}

// MCPConfig - MCP server configuration
type MCPConfig struct {
	ID    string   `json:"id"`
	Tools []string `json:"tools,omitempty"` // empty = all
}

// Event - event trigger config
type Event struct {
	Type   EventSource            `json:"type"`   // webhook | database
	Source string                 `json:"source"` // webhook path or table name
	Filter map[string]interface{} `json:"filter,omitempty"`
}

// ParseConfig parses robot_config from various formats (string, []byte, map)
func ParseConfig(data interface{}) (*Config, error) {
	if data == nil {
		return nil, nil
	}

	var configBytes []byte

	switch v := data.(type) {
	case string:
		if v == "" {
			return nil, nil
		}
		configBytes = []byte(v)
	case []byte:
		if len(v) == 0 {
			return nil, nil
		}
		configBytes = v
	case map[string]interface{}:
		var err error
		configBytes, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
	default:
		var err error
		configBytes, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
	}

	var config Config
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
