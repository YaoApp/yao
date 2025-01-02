package store

// Setting represents the conversation configuration structure
// Used to configure basic conversation parameters including connector, user field, table name, etc.
type Setting struct {
	Connector string `json:"connector,omitempty"`                          // Name of the connector used to specify data storage method
	UserField string `json:"user_field,omitempty"`                         // User ID field name, defaults to "user_id"
	Prefix    string `json:"prefix,omitempty"`                             // Database table name prefix
	MaxSize   int    `json:"max_size,omitempty" yaml:"max_size,omitempty"` // Maximum storage size limit
	TTL       int    `json:"ttl,omitempty" yaml:"ttl,omitempty"`           // Time To Live in seconds
}

// ChatInfo represents the chat information structure
// Contains basic information and history for a single chat
type ChatInfo struct {
	Chat    map[string]interface{}   `json:"chat"`    // Basic chat information
	History []map[string]interface{} `json:"history"` // Chat history records
}

// ChatFilter represents the chat filter structure
// Used for filtering and pagination when retrieving chat lists
type ChatFilter struct {
	Keywords string `json:"keywords,omitempty"` // Keyword search
	Page     int    `json:"page,omitempty"`     // Page number, starting from 1
	PageSize int    `json:"pagesize,omitempty"` // Number of items per page
	Order    string `json:"order,omitempty"`    // Sort order: desc/asc
}

// ChatGroup represents the chat group structure
// Groups chats by date
type ChatGroup struct {
	Label string                   `json:"label"` // Group label (typically a date)
	Chats []map[string]interface{} `json:"chats"` // List of chats in this group
}

// ChatGroupResponse represents the paginated chat group response
// Contains paginated chat group information
type ChatGroupResponse struct {
	Groups   []ChatGroup `json:"groups"`    // List of chat groups
	Page     int         `json:"page"`      // Current page number
	PageSize int         `json:"pagesize"`  // Items per page
	Total    int64       `json:"total"`     // Total number of records
	LastPage int         `json:"last_page"` // Last page number
}

// AssistantFilter represents the assistant filter structure
// Used for filtering and pagination when retrieving assistant lists
type AssistantFilter struct {
	Tags        []string `json:"tags,omitempty"`         // Filter by tags
	Keywords    string   `json:"keywords,omitempty"`     // Search in name and description
	Connector   string   `json:"connector,omitempty"`    // Filter by connector
	AssistantID string   `json:"assistant_id,omitempty"` // Filter by assistant ID
	Mentionable *bool    `json:"mentionable,omitempty"`  // Filter by mentionable status
	Automated   *bool    `json:"automated,omitempty"`    // Filter by automation status
	BuiltIn     *bool    `json:"built_in,omitempty"`     // Filter by built-in status
	Page        int      `json:"page,omitempty"`         // Page number, starting from 1
	PageSize    int      `json:"pagesize,omitempty"`     // Items per page
	Select      []string `json:"select,omitempty"`       // Fields to return, returns all fields if empty
}

// AssistantResponse represents the assistant response structure
// Used for returning paginated assistant lists
type AssistantResponse struct {
	Data     []map[string]interface{} `json:"data"`     // The paginated data
	Page     int                      `json:"page"`     // Current page number
	PageSize int                      `json:"pagesize"` // Number of items per page
	PageCnt  int                      `json:"pagecnt"`  // Total number of pages
	Next     int                      `json:"next"`     // Next page number
	Prev     int                      `json:"prev"`     // Previous page number
	Total    int64                    `json:"total"`    // Total number of items
}

// Store defines the conversation storage interface
// Provides basic operations required for conversation management
type Store interface {
	// GetChats retrieves a list of chats
	// sid: Session ID
	// filter: Filter conditions
	// Returns: Grouped chat list and potential error
	GetChats(sid string, filter ChatFilter) (*ChatGroupResponse, error)

	// GetChat retrieves a single chat's information
	// sid: Session ID
	// cid: Chat ID
	// Returns: Chat information and potential error
	GetChat(sid string, cid string) (*ChatInfo, error)

	// GetHistory retrieves chat history
	// sid: Session ID
	// cid: Chat ID
	// Returns: History record list and potential error
	GetHistory(sid string, cid string) ([]map[string]interface{}, error)

	// SaveHistory saves chat history
	// sid: Session ID
	// messages: Message list
	// cid: Chat ID
	// context: Context information
	// Returns: Potential error
	SaveHistory(sid string, messages []map[string]interface{}, cid string, context map[string]interface{}) error

	// DeleteChat deletes a single chat
	// sid: Session ID
	// cid: Chat ID
	// Returns: Potential error
	DeleteChat(sid string, cid string) error

	// DeleteAllChats deletes all chats
	// sid: Session ID
	// Returns: Potential error
	DeleteAllChats(sid string) error

	// UpdateChatTitle updates chat title
	// sid: Session ID
	// cid: Chat ID
	// title: New title
	// Returns: Potential error
	UpdateChatTitle(sid string, cid string, title string) error

	// SaveAssistant saves assistant information
	// assistant: Assistant information
	// Returns: Potential error
	SaveAssistant(assistant map[string]interface{}) (interface{}, error)

	// DeleteAssistant deletes an assistant
	// assistantID: Assistant ID
	// Returns: Potential error
	DeleteAssistant(assistantID string) error

	// GetAssistants retrieves a list of assistants
	// filter: Filter conditions
	// Returns: Paginated assistant list and potential error
	GetAssistants(filter AssistantFilter) (*AssistantResponse, error)

	// GetAssistant retrieves a single assistant by ID
	// assistantID: Assistant ID
	// Returns: Assistant information and potential error
	GetAssistant(assistantID string) (map[string]interface{}, error)

	// DeleteAssistants deletes assistants based on filter conditions
	// filter: Filter conditions
	// Returns: Number of deleted records and potential error
	DeleteAssistants(filter AssistantFilter) (int64, error)

	// GetAssistantTags retrieves all unique tags from assistants
	// Returns: List of tags and potential error
	GetAssistantTags() ([]string, error)
}
