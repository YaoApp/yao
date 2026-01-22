package types

import (
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/assistant"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	store "github.com/yaoapp/yao/agent/store/types"
)

// DSL AI assistant
type DSL struct {

	// Agent Global Settings
	// ===============================
	Uses         *Uses         `json:"uses,omitempty" yaml:"uses,omitempty"` // Which assistant to use default, title, prompt
	StoreSetting store.Setting `json:"store" yaml:"store"`                   // The store setting of the assistant
	Cache        string        `json:"cache" yaml:"cache"`                   // The cache store of the assistant, if not set, default is "__yao.agent.cache"

	// System Agents Connector Settings
	// ===============================
	// System configures connectors for system agents (__yao.keyword, __yao.querydsl, __yao.title, __yao.prompt)
	// Each agent can have its own connector, or use the default
	// If not set, fallback to the first connector that supports the required capabilities
	System *System `json:"system,omitempty" yaml:"system,omitempty"`

	// Global External Settings - model capabilities, tools, etc.
	// ===============================
	Models map[string]openai.Capabilities `json:"models,omitempty" yaml:"models,omitempty"` // The model capabilities configuration
	KB     *store.KBSetting               `json:"kb,omitempty" yaml:"kb,omitempty"`         // The knowledge base configuration loaded from agent/kb.yml
	Search *searchTypes.Config            `json:"search,omitempty" yaml:"search,omitempty"` // The search configuration loaded from agent/search.yao

	// Internal
	// ===============================
	// ID            string            `json:"-" yaml:"-"` // The id of the instance
	Assistant     assistant.API  `json:"-" yaml:"-"` // The default assistant
	Store         store.Store    `json:"-" yaml:"-"` // The store of the assistant
	GlobalPrompts []store.Prompt `json:"-" yaml:"-"` // Global prompts loaded from agent/prompts.yml
}

// Uses the default assistant settings
// ===============================
type Uses struct {
	Default     string `json:"default,omitempty" yaml:"default,omitempty"`           // The default assistant to use
	Title       string `json:"title,omitempty" yaml:"title,omitempty"`               // The assistant for generating the topic title.
	Prompt      string `json:"prompt,omitempty" yaml:"prompt,omitempty"`             // The assistant for generating the prompt.
	RobotPrompt string `json:"robot_prompt,omitempty" yaml:"robot_prompt,omitempty"` // The assistant for generating Robot's system prompt (responsibilities description).
	Vision      string `json:"vision,omitempty" yaml:"vision,omitempty"`             // The assistant for generating the image/video description, if the assistant enable the vision and model not support vision, use the vision model to describe the image/video, and return the messages with the image/video's description. Format: "agent" or "mcp:mcp_server_id"
	Audio       string `json:"audio,omitempty" yaml:"audio,omitempty"`               // The assistant for processing audio (speech-to-text, text-to-speech). If the model doesn't support audio, use this to convert audio to text. Format: "agent" or "mcp:mcp_server_id"
	Search      string `json:"search,omitempty" yaml:"search,omitempty"`             // The assistant for searching the knowledge, global web search. If not set, and the assistant enable the knowledge, it will search the result from the knowledge automatically.
	Fetch       string `json:"fetch,omitempty" yaml:"fetch,omitempty"`               // The assistant for fetching the http/https/ftp/sftp/etc. file, and return the file's content. if not set, use the http process to fetch the file.

	// Search-related processing tools (NLP)
	Web      string `json:"web,omitempty" yaml:"web,omitempty"`           // Web search handler: "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	Keyword  string `json:"keyword,omitempty" yaml:"keyword,omitempty"`   // Keyword extraction: "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	QueryDSL string `json:"querydsl,omitempty" yaml:"querydsl,omitempty"` // QueryDSL generation: "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	Rerank   string `json:"rerank,omitempty" yaml:"rerank,omitempty"`     // Result reranking: "builtin", "<assistant-id>", "mcp:<server>.<tool>"
}

// System configures connectors for system agents
// ===============================
type System struct {
	Default     string `json:"default,omitempty" yaml:"default,omitempty"`           // Default connector for all system agents
	Keyword     string `json:"keyword,omitempty" yaml:"keyword,omitempty"`           // Connector for __yao.keyword agent
	QueryDSL    string `json:"querydsl,omitempty" yaml:"querydsl,omitempty"`         // Connector for __yao.querydsl agent
	Title       string `json:"title,omitempty" yaml:"title,omitempty"`               // Connector for __yao.title agent
	Prompt      string `json:"prompt,omitempty" yaml:"prompt,omitempty"`             // Connector for __yao.prompt agent
	RobotPrompt string `json:"robot_prompt,omitempty" yaml:"robot_prompt,omitempty"` // Connector for __yao.robot_prompt agent
	NeedSearch  string `json:"needsearch,omitempty" yaml:"needsearch,omitempty"`     // Connector for __yao.needsearch agent
	Entity      string `json:"entity,omitempty" yaml:"entity,omitempty"`             // Connector for __yao.entity agent
}

// Mention Structure
// ===============================
type Mention struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
	Type   string `json:"type,omitempty"`
}
