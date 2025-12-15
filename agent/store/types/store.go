package types

// ChatStore defines the chat storage interface
// Provides operations for chat, message, and resume management
type ChatStore interface {
	// ==========================================================================
	// Chat Management
	// ==========================================================================

	// CreateChat creates a new chat session
	// chat: Chat session to create
	// Returns: Potential error
	CreateChat(chat *Chat) error

	// GetChat retrieves a single chat by ID
	// chatID: Chat ID
	// Returns: Chat information and potential error
	GetChat(chatID string) (*Chat, error)

	// UpdateChat updates chat fields
	// chatID: Chat ID
	// updates: Map of fields to update
	// Returns: Potential error
	UpdateChat(chatID string, updates map[string]interface{}) error

	// DeleteChat deletes a chat and its associated messages
	// chatID: Chat ID
	// Returns: Potential error
	DeleteChat(chatID string) error

	// ListChats retrieves a paginated list of chats with optional grouping
	// filter: Filter conditions including time range, sorting, and grouping
	// Returns: Paginated chat list (flat or grouped) and potential error
	ListChats(filter ChatFilter) (*ChatList, error)

	// ==========================================================================
	// Message Management
	// ==========================================================================

	// SaveMessages batch saves messages for a chat
	// This is the primary write method - messages are buffered during execution
	// and batch-written at the end of a request
	// chatID: Parent chat ID
	// messages: Messages to save (includes user input and assistant responses)
	// Returns: Potential error
	SaveMessages(chatID string, messages []*Message) error

	// GetMessages retrieves messages for a chat with filtering
	// chatID: Chat ID
	// filter: Filter conditions (role, type, block, thread, etc.)
	// Returns: Message list and potential error
	GetMessages(chatID string, filter MessageFilter) ([]*Message, error)

	// UpdateMessage updates a single message
	// messageID: Message ID
	// updates: Map of fields to update
	// Returns: Potential error
	UpdateMessage(messageID string, updates map[string]interface{}) error

	// DeleteMessages deletes specific messages from a chat
	// chatID: Chat ID
	// messageIDs: List of message IDs to delete
	// Returns: Potential error
	DeleteMessages(chatID string, messageIDs []string) error

	// ==========================================================================
	// Resume Management (only called on failure/interrupt)
	// ==========================================================================

	// SaveResume batch saves resume records
	// Only called when request is interrupted or failed
	// records: Resume records to save
	// Returns: Potential error
	SaveResume(records []*Resume) error

	// GetResume retrieves all resume records for a chat
	// chatID: Chat ID
	// Returns: Resume records and potential error
	GetResume(chatID string) ([]*Resume, error)

	// GetLastResume retrieves the last (most recent) resume record for a chat
	// chatID: Chat ID
	// Returns: Last resume record and potential error
	GetLastResume(chatID string) (*Resume, error)

	// GetResumeByStackID retrieves resume records for a specific stack
	// stackID: Stack ID
	// Returns: Resume records and potential error
	GetResumeByStackID(stackID string) ([]*Resume, error)

	// GetStackPath returns the stack path from root to the given stack
	// stackID: Current stack ID
	// Returns: Stack path [root_stack_id, ..., current_stack_id] and potential error
	GetStackPath(stackID string) ([]string, error)

	// DeleteResume deletes all resume records for a chat
	// Called after successful resume to clean up
	// chatID: Chat ID
	// Returns: Potential error
	DeleteResume(chatID string) error

	// ==========================================================================
	// Search Management
	// ==========================================================================

	// SaveSearch saves a search record for a request
	// Used for citation support, debugging, and replay
	// search: Search record to save
	// Returns: Potential error
	SaveSearch(search *Search) error

	// GetSearches retrieves all search records for a request
	// requestID: Request ID
	// Returns: Search records and potential error
	GetSearches(requestID string) ([]*Search, error)

	// GetReference retrieves a single reference by request ID and index
	// Used for citation click handling
	// requestID: Request ID
	// index: Reference index (1-based)
	// Returns: Reference and potential error
	GetReference(requestID string, index int) (*Reference, error)

	// DeleteSearches deletes all search records for a chat
	// Called when deleting a chat
	// chatID: Chat ID
	// Returns: Potential error
	DeleteSearches(chatID string) error
}

// AssistantStore defines the assistant storage interface
// Separated from ChatStore for clearer responsibility
type AssistantStore interface {
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
	// fields: List of fields to select, empty/nil means default fields
	// locale: Optional locale for i18n translations
	// Returns: Assistant information and potential error
	GetAssistant(assistantID string, fields []string, locale ...string) (*AssistantModel, error)

	// DeleteAssistants deletes assistants based on filter conditions
	// filter: Filter conditions
	// Returns: Number of deleted records and potential error
	DeleteAssistants(filter AssistantFilter) (int64, error)
}

// Store combines ChatStore and AssistantStore interfaces
// This is the main interface for the storage layer
type Store interface {
	ChatStore
	AssistantStore
}

// SpaceStore defines the interface for Space snapshot operations
// Note: Space itself uses plan.Space interface, this is for persistence
type SpaceStore interface {
	// Snapshot returns all key-value pairs in the space
	Snapshot() map[string]interface{}

	// Restore sets multiple key-value pairs from a snapshot
	Restore(data map[string]interface{}) error
}
