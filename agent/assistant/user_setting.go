package assistant

import (
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
)

// UserAgentSetting stores per-user agent preferences (runners, image, secrets).
// Persisted in setting.Registry under namespace "agent.<assistant_id>".
type UserAgentSetting struct {
	Runners []string                      `json:"runners,omitempty"`
	Image   string                        `json:"image,omitempty"`
	Secrets map[string]*types.SecretEntry `json:"secrets,omitempty"`
	Options map[string]any                `json:"options,omitempty"`
}
