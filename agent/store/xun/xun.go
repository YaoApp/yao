package xun

import (
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
	"github.com/yaoapp/yao/agent/store/types"
)

// Package store provides functionality for managing chat conversations and assistants.

// Xun implements the Store interface using a database backend.
// It provides functionality for:
// - Managing chat sessions and their messages
// - Organizing chats with pagination and date-based grouping
// - Handling chat metadata like titles and creation dates
// - Managing AI assistants with their configurations and metadata
// - Managing resume records for recovery from interruptions
type Xun struct {
	query   query.Query
	schema  schema.Schema
	setting types.Setting
}

// Public interface methods:
//
// NewXun creates a new store instance with the given settings
//
// Chat Management:
// CreateChat creates a new chat session
// GetChat retrieves a single chat by ID
// UpdateChat updates chat fields
// DeleteChat deletes a chat and its associated messages
// ListChats retrieves a paginated list of chats with optional grouping
//
// Message Management:
// SaveMessages batch saves messages for a chat
// GetMessages retrieves messages for a chat with filtering
// UpdateMessage updates a single message
// DeleteMessages deletes specific messages from a chat
//
// Resume Management:
// SaveResume batch saves resume records (only on failure/interrupt)
// GetResume retrieves all resume records for a chat
// GetLastResume retrieves the last resume record for a chat
// GetResumeByStackID retrieves resume records for a specific stack
// GetStackPath returns the stack path from root to the given stack
// DeleteResume deletes all resume records for a chat
//
// Assistant Management:
// SaveAssistant creates or updates an assistant
// UpdateAssistant updates assistant fields
// DeleteAssistant deletes an assistant by assistant_id
// GetAssistants retrieves a paginated list of assistants with filtering
// GetAssistant retrieves a single assistant by assistant_id
// DeleteAssistants deletes assistants based on filter conditions
// GetAssistantTags retrieves all unique tags from assistants

// NewXun create a new xun store
func NewXun(setting types.Setting) (types.Store, error) {
	store := &Xun{setting: setting}
	if setting.Connector == "default" || setting.Connector == "" {
		store.query = capsule.Global.Query()
		store.schema = capsule.Global.Schema()
	} else {
		conn, err := connector.Select(setting.Connector)
		if err != nil {
			return nil, fmt.Errorf("select store connector %s error: %s", setting.Connector, err.Error())
		}

		store.query, err = conn.Query()
		if err != nil {
			return nil, fmt.Errorf("query store connector %s error: %s", setting.Connector, err.Error())
		}

		store.schema, err = conn.Schema()
		if err != nil {
			return nil, err
		}
	}

	return store, nil
}

// =============================================================================
// Query Builders
// =============================================================================

// newQueryChat creates a new query builder for the chat table
func (store *Xun) newQueryChat() query.Query {
	qb := store.query.New()
	qb.Table(store.getChatTable())
	return qb
}

// newQueryMessage creates a new query builder for the message table
func (store *Xun) newQueryMessage() query.Query {
	qb := store.query.New()
	qb.Table(store.getMessageTable())
	return qb
}

// newQueryResume creates a new query builder for the resume table
func (store *Xun) newQueryResume() query.Query {
	qb := store.query.New()
	qb.Table(store.getResumeTable())
	return qb
}

// newQueryAssistant creates a new query builder for the assistant table
func (store *Xun) newQueryAssistant() query.Query {
	qb := store.query.New()
	qb.Table(store.getAssistantTable())
	return qb
}

// =============================================================================
// Table Name Getters
// =============================================================================

// getChatTable returns the chat table name
func (store *Xun) getChatTable() string {
	m := model.Select("__yao.agent.chat")
	if m != nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return "agent_chat"
}

// getMessageTable returns the message table name
func (store *Xun) getMessageTable() string {
	m := model.Select("__yao.agent.message")
	if m != nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return "agent_message"
}

// getResumeTable returns the resume table name
func (store *Xun) getResumeTable() string {
	m := model.Select("__yao.agent.resume")
	if m != nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return "agent_resume"
}

// getAssistantTable returns the assistant table name
func (store *Xun) getAssistantTable() string {
	m := model.Select("__yao.agent.assistant")
	if m != nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return "agent_assistant"
}

// =============================================================================
// Utility Functions
// =============================================================================

// parseJSONFields parses JSON string fields into their corresponding Go types
func (store *Xun) parseJSONFields(data map[string]interface{}, fields []string) {
	for _, field := range fields {
		if val := data[field]; val != nil {
			if strVal, ok := val.(string); ok && strVal != "" {
				var parsed interface{}
				if err := jsoniter.UnmarshalFromString(strVal, &parsed); err == nil {
					data[field] = parsed
				}
			}
		}
	}
}

// GenerateAssistantID generates a random-looking 6-digit ID
func (store *Xun) GenerateAssistantID() (string, error) {
	maxAttempts := 10 // Maximum number of attempts to generate a unique ID
	for i := 0; i < maxAttempts; i++ {
		// Generate a random number using timestamp and some bit operations
		timestamp := time.Now().UnixNano()
		random := (timestamp ^ (timestamp >> 12)) % 1000000
		hash := fmt.Sprintf("%06d", random)

		// Check if this ID already exists
		exists, err := store.query.New().
			Table(store.getAssistantTable()).
			Where("assistant_id", hash).
			Exists()

		if err != nil {
			return "", err
		}

		if !exists {
			return hash, nil
		}

		// If ID exists, wait a bit and try again
		time.Sleep(time.Millisecond)
	}

	return "", fmt.Errorf("failed to generate unique ID after %d attempts", maxAttempts)
}
