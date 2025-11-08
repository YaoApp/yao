package types

// Store defines the conversation storage interface
// Provides basic operations required for conversation management
type Store interface {
	// GetChats retrieves a list of chats
	// sid: Session ID
	// filter: Filter conditions
	// Returns: Grouped chat list and potential error
	GetChats(sid string, filter ChatFilter, locale ...string) (*ChatGroupResponse, error)

	// GetChat retrieves a single chat's information
	// sid: Session ID
	// cid: Chat ID
	// Returns: Chat information and potential error
	GetChat(sid string, cid string, locale ...string) (*ChatInfo, error)

	// GetChatWithFilter retrieves a single chat's information with filter options
	// sid: Session ID
	// cid: Chat ID
	// filter: Filter conditions
	// Returns: Chat information and potential error
	GetChatWithFilter(sid string, cid string, filter ChatFilter, locale ...string) (*ChatInfo, error)

	// GetHistory retrieves chat history
	// sid: Session ID
	// cid: Chat ID
	// Returns: History record list and potential error
	GetHistory(sid string, cid string, locale ...string) ([]map[string]interface{}, error)

	// GetHistoryWithFilter retrieves chat history with filter options
	// sid: Session ID
	// cid: Chat ID
	// filter: Filter conditions
	// Returns: History record list and potential error
	GetHistoryWithFilter(sid string, cid string, filter ChatFilter, locale ...string) ([]map[string]interface{}, error)

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
	// Returns: Assistant ID and potential error
	SaveAssistant(assistant *AssistantModel) (string, error)

	// UpdateAssistant updates assistant fields
	// assistantID: Assistant ID
	// updates: Map of fields to update
	// Returns: Potential error
	UpdateAssistant(assistantID string, updates map[string]interface{}) error

	// DeleteAssistant deletes an assistant
	// assistantID: Assistant ID
	// Returns: Potential error
	DeleteAssistant(assistantID string) error

	// GetAssistants retrieves a paginated list of assistants with filtering
	// filter: Filter conditions for querying assistants
	// locale: Optional locale for i18n translations
	// Returns: Paginated assistant list and potential error
	GetAssistants(filter AssistantFilter, locale ...string) (*AssistantList, error)

	// GetAssistantTags retrieves all unique tags from assistants with filtering
	// filter: Filter conditions including QueryFilter for permission filtering
	// locale: Optional locale for i18n translations
	// Returns: List of tags and potential error
	GetAssistantTags(filter AssistantFilter, locale ...string) ([]Tag, error)

	// GetAssistant retrieves a single assistant by ID
	// assistantID: Assistant ID
	// Returns: Assistant information and potential error
	GetAssistant(assistantID string, locale ...string) (*AssistantModel, error)

	// DeleteAssistants deletes assistants based on filter conditions
	// filter: Filter conditions
	// Returns: Number of deleted records and potential error
	DeleteAssistants(filter AssistantFilter) (int64, error)

	// Close closes the store and releases any resources
	// Returns: Potential error
	Close() error
}
