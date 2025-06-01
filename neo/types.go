package neo

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/neo/assistant"
	"github.com/yaoapp/yao/neo/attachment"
	"github.com/yaoapp/yao/neo/rag"
	"github.com/yaoapp/yao/neo/store"
	"github.com/yaoapp/yao/neo/vision"
)

// DSL AI assistant
type DSL struct {

	// Neo Global Settings
	// ===============================
	Use              *Use          `json:"use,omitempty" yaml:"use,omitempty"`             // Which assistant to use default, title, prompt
	StoreSetting     store.Setting `json:"store" yaml:"store"`                             // The store setting of the assistant
	AuthSetting      *Auth         `json:"auth,omitempty" yaml:"auth,omitempty"`           // Authenticate Settings
	UploadSetting    *Upload       `json:"upload,omitempty" yaml:"upload,omitempty"`       // Upload Settings
	KnowledgeSetting *Knowledge    `json:"knowledge,omitempty" yaml:"knowledge,omitempty"` // Knowledge base Settings

	// Global External Settings - connectors, tools, etc.
	// ===============================
	Connectors map[string]assistant.ConnectorSetting `json:"connectors,omitempty" yaml:"connectors,omitempty"` // The connectors of the assistant

	// Neo API Settings
	// ===============================s
	Guard  string   `json:"guard,omitempty" yaml:"guard,omitempty"`   // The guard of the assistant
	Allows []string `json:"allows,omitempty" yaml:"allows,omitempty"` // The allowed domains of the assistant

	// Internal
	// ===============================
	ID            string            `json:"-" yaml:"-"` // The id of the instance
	Assistant     assistant.API     `json:"-" yaml:"-"` // The default assistant
	Store         store.Store       `json:"-" yaml:"-"` // The store of the assistant
	RAG           *rag.RAG          `json:"-" yaml:"-"`
	Vision        *vision.Vision    `json:"-" yaml:"-"`
	GuardHandlers []gin.HandlerFunc `json:"-" yaml:"-"`
}

// Use the default assistant settings
// ===============================
type Use struct {
	Default string `json:"default,omitempty" yaml:"default,omitempty"` // The default assistant to use
	Title   string `json:"title,omitempty" yaml:"title,omitempty"`     // The assistant for generating the topic title.
	Prompt  string `json:"prompt,omitempty" yaml:"prompt,omitempty"`   // The assistant for generating the prompt.
	Vision  string `json:"vision,omitempty" yaml:"vision,omitempty"`   // The assistant for generating the image/video description, if the assistant enable the vision and model not support vision, use the vision model to describe the image/video, and return the messages with the image/video's description.
	Search  string `json:"search,omitempty" yaml:"search,omitempty"`   // The assistant for searching the knowledge, global web search. If not set, and the assistant enable the knowledge, it will search the result from the knowledge automatically.
	Fetch   string `json:"fetch,omitempty" yaml:"fetch,omitempty"`     // The assistant for fetching the http/https/ftp/sftp/etc. file, and return the file's content. if not set, use the http process to fetch the file.
}

// Auth Authenticate Settings
// ===============================
type Auth struct {
	Models        *AuthModels        `json:"models,omitempty" yaml:"models,omitempty"`                 // The models of the user, it is used to handle the user, and the user is a user in the database. (Guest and User model must have the id and permission fields)
	Fields        *AuthFields        `json:"fields,omitempty" yaml:"fields,omitempty"`                 // The fields of the user model, it is used to handle the user, and the user is a user in the database. (Guest and User model must have the id and permission fields)
	SessionFields *AuthSessionFields `json:"session_fields,omitempty" yaml:"session_fields,omitempty"` // The session fields of the user, it is used to handle the user, and the user is a user in the database.
}

// AuthModels the auth model
type AuthModels struct {
	User  string `json:"user,omitempty" yaml:"user,omitempty"`   // default is admin.user, The user model is a special model, it is used to handle the user, and the user is a user in the database.
	Guest string `json:"guest,omitempty" yaml:"guest,omitempty"` // The guest model is a special model, it is used to handle the guest user, and the guest user is not a user in the database.
}

// AuthSessionFields the auth session field
type AuthSessionFields struct {
	ID    string `json:"id,omitempty" yaml:"id,omitempty"`       // the field name of the user id, default is user_id
	Roles string `json:"roles,omitempty" yaml:"roles,omitempty"` // the field name of the user roles, default is user_roles. the value must be an JSON array string.
	Guest string `json:"guest,omitempty" yaml:"guest,omitempty"` // the field name of the guest user, default is guest_id
}

// AuthFields the auth field
type AuthFields struct {
	ID         string `json:"id,omitempty" yaml:"id,omitempty"`                 // the field name of the user id, default is id
	Roles      string `json:"roles,omitempty" yaml:"roles,omitempty"`           // the field name of the user roles, default is roles, it must be an JSON field.
	Permission string `json:"permission,omitempty" yaml:"permission,omitempty"` // the field name of the user permission, default is permission
}

// Upload the upload setting
// ===============================
type Upload struct {
	Chat      *attachment.ManagerOption `json:"chat,omitempty" yaml:"chat,omitempty"`           // Chat conversation upload setting, if not set use the local and root path is `/attachments`.
	Assets    *attachment.ManagerOption `json:"assets,omitempty" yaml:"assets,omitempty"`       // Asset upload setting, if not set use the chat upload setting.
	Knowledge *attachment.ManagerOption `json:"knowledge,omitempty" yaml:"knowledge,omitempty"` // Knowledge base upload setting, if not set use the chat upload setting.
}

// UploadOption the upload option
type UploadOption struct {
	attachment.UploadOption
	Public       bool        `json:"public,omitempty" yaml:"public,omitempty, form:public"`                      // The public of the file, default is false
	Scope        interface{} `json:"scope,omitempty" yaml:"scope,omitempty, form:scope"`                         // The scope of the file, default is private
	CollectionID string      `json:"collection_id,omitempty" yaml:"collection_id,omitempty, form:collection_id"` // The collection id of the file, default is empty
}

// Knowledge base Settings
// ===============================
type Knowledge struct {
	Vector     KnowledgeVector     `json:"vector" yaml:"vector"`         // The vector database driver
	Graph      KnowledgeGraph      `json:"graph" yaml:"graph"`           // The graph database driver
	Vectorizer KnowledgeVectorizer `json:"vectorizer" yaml:"vectorizer"` // The vectorizer driver
}

// KnowledgeVectorizer the knowledge vectorizer
type KnowledgeVectorizer struct {
	Driver  string                 `json:"driver" yaml:"driver"`
	Options map[string]interface{} `json:"options" yaml:"options"`
}

// KnowledgeVector the knowledge vector
type KnowledgeVector struct {
	Driver  string                 `json:"driver" yaml:"driver"`
	Options map[string]interface{} `json:"options" yaml:"options"`
}

// KnowledgeGraph the knowledge graph
type KnowledgeGraph struct {
	Driver  string                 `json:"driver" yaml:"driver"`
	Options map[string]interface{} `json:"options" yaml:"options"`
}

// Mention Structure
// ===============================
type Mention struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
	Type   string `json:"type,omitempty"`
}
