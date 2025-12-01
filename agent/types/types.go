package types

import (
	"github.com/yaoapp/yao/agent/assistant"
	store "github.com/yaoapp/yao/agent/store/types"
)

// DSL AI assistant
type DSL struct {

	// Agent Global Settings
	// ===============================
	Uses         *Uses         `json:"uses,omitempty" yaml:"uses,omitempty"` // Which assistant to use default, title, prompt
	StoreSetting store.Setting `json:"store" yaml:"store"`                   // The store setting of the assistant
	Cache        string        `json:"cache" yaml:"cache"`                   // The cache store of the assistant, if not set, default is "__yao.agent.cache"

	// Global External Settings - model capabilities, tools, etc.
	// ===============================
	Models map[string]assistant.ModelCapabilities `json:"models,omitempty" yaml:"models,omitempty"` // The model capabilities configuration

	// Internal
	// ===============================
	// ID            string            `json:"-" yaml:"-"` // The id of the instance
	Assistant assistant.API `json:"-" yaml:"-"` // The default assistant
	Store     store.Store   `json:"-" yaml:"-"` // The store of the assistant
}

// Uses the default assistant settings
// ===============================
type Uses struct {
	Default string `json:"default,omitempty" yaml:"default,omitempty"` // The default assistant to use
	Title   string `json:"title,omitempty" yaml:"title,omitempty"`     // The assistant for generating the topic title.
	Prompt  string `json:"prompt,omitempty" yaml:"prompt,omitempty"`   // The assistant for generating the prompt.
	Vision  string `json:"vision,omitempty" yaml:"vision,omitempty"`   // The assistant for generating the image/video description, if the assistant enable the vision and model not support vision, use the vision model to describe the image/video, and return the messages with the image/video's description. Format: "agent" or "mcp:mcp_server_id"
	Audio   string `json:"audio,omitempty" yaml:"audio,omitempty"`     // The assistant for processing audio (speech-to-text, text-to-speech). If the model doesn't support audio, use this to convert audio to text. Format: "agent" or "mcp:mcp_server_id"
	Search  string `json:"search,omitempty" yaml:"search,omitempty"`   // The assistant for searching the knowledge, global web search. If not set, and the assistant enable the knowledge, it will search the result from the knowledge automatically.
	Fetch   string `json:"fetch,omitempty" yaml:"fetch,omitempty"`     // The assistant for fetching the http/https/ftp/sftp/etc. file, and return the file's content. if not set, use the http process to fetch the file.
}

// Mention Structure
// ===============================
type Mention struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
	Type   string `json:"type,omitempty"`
}
