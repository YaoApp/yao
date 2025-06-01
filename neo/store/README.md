# YAO Neo Store

YAO Neo Store is a comprehensive storage abstraction layer for managing conversations, assistants, attachments, and knowledge collections in the YAO Neo platform. It provides a unified interface that supports multiple storage backends including databases (via Xun), Redis, and MongoDB.

## Table of Contents

- [Architecture](#architecture)
- [Storage Backends](#storage-backends)
- [Configuration](#configuration)
- [Initialization](#initialization)
- [API Reference](#api-reference)
- [Data Models](#data-models)
- [Usage Examples](#usage-examples)
- [Testing](#testing)

## Architecture

The store package provides a unified `Store` interface that abstracts different storage implementations:

```
┌─────────────────┐
│   Store API     │  ← Unified Interface
├─────────────────┤
│ Xun (Database)  │  ← Primary Implementation
│ Redis           │  ← Cache/Memory Store
│ MongoDB         │  ← Document Store
└─────────────────┘
```

### Core Entities

1. **Conversations & Chat History** - Manage chat sessions and message history
2. **Assistants** - AI assistant configurations and metadata
3. **Attachments** - File attachments with metadata and access control
4. **Knowledge Collections** - Knowledge bases for AI assistants

## Storage Backends

### 1. Xun (Database) - Primary Backend

The main implementation using SQL databases with automatic schema management:

- **Supported Databases**: MySQL, PostgreSQL, SQLite, etc.
- **Features**: ACID transactions, complex queries, automatic migrations
- **Use Case**: Production environments requiring data consistency

### 2. Redis - Cache Backend

Redis implementation for high-performance caching:

- **Features**: In-memory storage, pub/sub capabilities
- **Use Case**: Session management, temporary data, real-time features

### 3. MongoDB - Document Backend

MongoDB implementation for document-based storage:

- **Features**: Schema flexibility, horizontal scaling
- **Use Case**: Large-scale deployments, unstructured data

## Configuration

### Setting Structure

```go
type Setting struct {
    Connector string `json:"connector,omitempty"`                          // Storage connector name
    UserField string `json:"user_field,omitempty"`                         // User ID field name (default: "user_id")
    Prefix    string `json:"prefix,omitempty"`                             // Database table name prefix
    MaxSize   int    `json:"max_size,omitempty" yaml:"max_size,omitempty"` // Maximum history size limit
    TTL       int    `json:"ttl,omitempty" yaml:"ttl,omitempty"`           // Time To Live in seconds
}
```

### Configuration Examples

#### Database Configuration

```yaml
# neo.yml
neo:
  store:
    connector: "mysql" # or "postgresql", "sqlite", "default"
    prefix: "neo_" # Table prefix
    max_size: 100 # Maximum chat history size
    ttl: 7200 # 2 hours TTL for conversations
    user_field: "user_id" # User identification field
```

#### Redis Configuration

```yaml
neo:
  store:
    connector: "redis"
    prefix: "neo:"
    ttl: 3600
```

#### MongoDB Configuration

```yaml
neo:
  store:
    connector: "mongodb"
    prefix: "neo_"
    ttl: 7200
```

## Initialization

### Automatic Initialization (Recommended)

The store is automatically initialized when the Neo system starts:

```go
// From yao/neo/load.go
func initStore() error {
    var err error
    if Neo.StoreSetting.Connector == "default" || Neo.StoreSetting.Connector == "" {
        Neo.Store, err = store.NewXun(Neo.StoreSetting)
        return err
    }

    // Other connector types
    conn, err := connector.Select(Neo.StoreSetting.Connector)
    if err != nil {
        return err
    }

    if conn.Is(connector.DATABASE) {
        Neo.Store, err = store.NewXun(Neo.StoreSetting)
        return err
    } else if conn.Is(connector.REDIS) {
        Neo.Store = store.NewRedis()
        return nil
    } else if conn.Is(connector.MONGO) {
        Neo.Store = store.NewMongo()
        return nil
    }

    return fmt.Errorf("%s store connector %s not support", Neo.ID, Neo.StoreSetting.Connector)
}
```

### Manual Initialization

```go
import "github.com/yaoapp/yao/neo/store"

// Database backend
setting := store.Setting{
    Connector: "mysql",
    Prefix:    "neo_",
    MaxSize:   100,
    TTL:       3600,
}
store, err := store.NewXun(setting)

// Redis backend
redisStore := store.NewRedis()

// MongoDB backend
mongoStore := store.NewMongo()
```

## API Reference

### Store Interface

```go
type Store interface {
    // Chat Management
    GetChats(sid string, filter ChatFilter, locale ...string) (*ChatGroupResponse, error)
    GetChat(sid string, cid string, locale ...string) (*ChatInfo, error)
    GetChatWithFilter(sid string, cid string, filter ChatFilter, locale ...string) (*ChatInfo, error)
    UpdateChatTitle(sid string, cid string, title string) error
    DeleteChat(sid string, cid string) error
    DeleteAllChats(sid string) error

    // Message History
    GetHistory(sid string, cid string, locale ...string) ([]map[string]interface{}, error)
    GetHistoryWithFilter(sid string, cid string, filter ChatFilter, locale ...string) ([]map[string]interface{}, error)
    SaveHistory(sid string, messages []map[string]interface{}, cid string, context map[string]interface{}) error

    // Assistant Management
    SaveAssistant(assistant map[string]interface{}) (interface{}, error)
    GetAssistants(filter AssistantFilter, locale ...string) (*AssistantResponse, error)
    GetAssistant(assistantID string, locale ...string) (map[string]interface{}, error)
    DeleteAssistant(assistantID string) error
    DeleteAssistants(filter AssistantFilter) (int64, error)
    GetAssistantTags(locale ...string) ([]Tag, error)

    // Attachment Management
    SaveAttachment(attachment map[string]interface{}) (interface{}, error)
    GetAttachments(filter AttachmentFilter, locale ...string) (*AttachmentResponse, error)
    GetAttachment(fileID string, locale ...string) (map[string]interface{}, error)
    DeleteAttachment(fileID string) error
    DeleteAttachments(filter AttachmentFilter) (int64, error)

    // Knowledge Management
    SaveKnowledge(knowledge map[string]interface{}) (interface{}, error)
    GetKnowledges(filter KnowledgeFilter, locale ...string) (*KnowledgeResponse, error)
    GetKnowledge(collectionID string, locale ...string) (map[string]interface{}, error)
    DeleteKnowledge(collectionID string) error
    DeleteKnowledges(filter KnowledgeFilter) (int64, error)

    // Resource Management
    Close() error
}
```

## Data Models

### Database Schema

#### 1. History Table (Conversations)

```sql
CREATE TABLE neo_history (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    sid VARCHAR(255) INDEX,                    -- Session ID
    cid VARCHAR(200) INDEX,                    -- Chat ID
    uid VARCHAR(255) INDEX,                    -- User ID
    role VARCHAR(200) INDEX,                   -- Message role (user/assistant/system)
    name VARCHAR(200),                         -- Message sender name
    content TEXT,                              -- Message content
    context JSON,                              -- Message context
    assistant_id VARCHAR(200) INDEX,           -- Associated assistant ID
    assistant_name VARCHAR(200),               -- Assistant name
    assistant_avatar VARCHAR(200),             -- Assistant avatar URL
    mentions JSON,                             -- Mentions in the message
    silent BOOLEAN DEFAULT FALSE INDEX,        -- Silent message flag
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP INDEX,
    updated_at TIMESTAMP INDEX,
    expired_at TIMESTAMP INDEX                 -- TTL expiration
);
```

#### 2. Chat Table

```sql
CREATE TABLE neo_chat (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    chat_id VARCHAR(200) UNIQUE INDEX,         -- Unique chat identifier
    title VARCHAR(200),                        -- Chat title
    assistant_id VARCHAR(200) INDEX,           -- Associated assistant
    sid VARCHAR(255) INDEX,                    -- Session ID
    silent BOOLEAN DEFAULT FALSE INDEX,        -- Silent chat flag
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP INDEX,
    updated_at TIMESTAMP INDEX
);
```

#### 3. Assistant Table

```sql
CREATE TABLE neo_assistant (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    assistant_id VARCHAR(200) UNIQUE INDEX,    -- Unique assistant identifier
    type VARCHAR(200) DEFAULT 'assistant' INDEX, -- Assistant type
    name VARCHAR(200),                         -- Assistant name
    avatar VARCHAR(200),                       -- Avatar URL
    connector VARCHAR(200) NOT NULL,           -- LLM connector
    description VARCHAR(600) INDEX,            -- Description (searchable)
    path VARCHAR(200),                         -- Storage path
    sort INTEGER DEFAULT 9999 INDEX,           -- Sort order
    built_in BOOLEAN DEFAULT FALSE INDEX,      -- Built-in assistant flag
    placeholder JSON,                          -- UI placeholder text
    options JSON,                              -- Assistant options
    prompts JSON,                              -- System prompts
    workflow JSON,                             -- Workflow configuration
    knowledge JSON,                            -- Knowledge base references
    tools JSON,                                -- Available tools
    tags JSON,                                 -- Assistant tags
    readonly BOOLEAN DEFAULT FALSE INDEX,      -- Read-only flag
    permissions JSON,                          -- Access permissions
    locales JSON,                              -- Internationalization data
    automated BOOLEAN DEFAULT TRUE INDEX,      -- Automation enabled
    mentionable BOOLEAN DEFAULT TRUE INDEX,    -- Can be mentioned in chats
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP INDEX,
    updated_at TIMESTAMP INDEX
);
```

#### 4. Attachment Table

```sql
CREATE TABLE neo_attachment (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    file_id VARCHAR(255) UNIQUE INDEX,         -- Unique file identifier
    uid VARCHAR(255) INDEX,                    -- Owner user ID
    guest BOOLEAN DEFAULT FALSE INDEX,         -- Guest upload flag
    manager VARCHAR(200) INDEX,                -- Storage manager
    content_type VARCHAR(200) INDEX,           -- MIME type
    name VARCHAR(500) INDEX,                   -- File name (searchable)
    public BOOLEAN DEFAULT FALSE INDEX,        -- Public access flag
    scope JSON,                                -- Access scope
    gzip BOOLEAN DEFAULT FALSE INDEX,          -- Compression flag
    bytes BIGINT INDEX,                        -- File size
    collection_id VARCHAR(200) INDEX,          -- Associated knowledge collection
    status ENUM('uploading', 'uploaded', 'indexing', 'indexed', 'upload_failed', 'index_failed') DEFAULT 'uploading' INDEX, -- Processing status
    progress VARCHAR(200),                     -- Progress information (nullable)
    error VARCHAR(600),                        -- Error message (nullable)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP INDEX,
    updated_at TIMESTAMP INDEX
);
```

#### 5. Knowledge Table

```sql
CREATE TABLE neo_knowledge (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    collection_id VARCHAR(200) UNIQUE INDEX,   -- Unique collection identifier
    name VARCHAR(200) INDEX,                   -- Collection name (searchable)
    description VARCHAR(600) INDEX,            -- Description (searchable)
    uid VARCHAR(255) INDEX,                    -- Owner user ID
    public BOOLEAN DEFAULT FALSE INDEX,        -- Public access flag
    scope JSON,                                -- Access scope
    readonly BOOLEAN DEFAULT FALSE INDEX,      -- Read-only flag
    option JSON,                               -- Collection options
    system BOOLEAN DEFAULT FALSE INDEX,        -- System collection flag
    sort INTEGER DEFAULT 9999 INDEX,           -- Sort order
    cover VARCHAR(500),                        -- Cover image URL
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP INDEX,
    updated_at TIMESTAMP INDEX
);
```

### Filter Structures

#### ChatFilter

```go
type ChatFilter struct {
    Keywords string `json:"keywords,omitempty"` // Search keywords
    Page     int    `json:"page,omitempty"`     // Page number (starts from 1)
    PageSize int    `json:"pagesize,omitempty"` // Items per page
    Order    string `json:"order,omitempty"`    // Sort order (desc/asc)
    Silent   *bool  `json:"silent,omitempty"`   // Include silent messages
}
```

#### AssistantFilter

```go
type AssistantFilter struct {
    Tags         []string `json:"tags,omitempty"`          // Filter by tags
    Type         string   `json:"type,omitempty"`          // Filter by type
    Keywords     string   `json:"keywords,omitempty"`      // Search keywords
    Connector    string   `json:"connector,omitempty"`     // Filter by connector
    AssistantID  string   `json:"assistant_id,omitempty"`  // Specific assistant ID
    AssistantIDs []string `json:"assistant_ids,omitempty"` // Multiple assistant IDs
    Mentionable  *bool    `json:"mentionable,omitempty"`   // Mentionable status
    Automated    *bool    `json:"automated,omitempty"`     // Automation status
    BuiltIn      *bool    `json:"built_in,omitempty"`      // Built-in status
    Page         int      `json:"page,omitempty"`          // Page number
    PageSize     int      `json:"pagesize,omitempty"`      // Items per page
    Select       []string `json:"select,omitempty"`        // Fields to return
}
```

#### AttachmentFilter

```go
type AttachmentFilter struct {
    UID          string   `json:"uid,omitempty"`           // Filter by user ID
    Guest        *bool    `json:"guest,omitempty"`         // Filter by guest status
    Manager      string   `json:"manager,omitempty"`       // Filter by upload manager
    ContentType  string   `json:"content_type,omitempty"`  // Filter by content type
    Name         string   `json:"name,omitempty"`          // Filter by filename
    Public       *bool    `json:"public,omitempty"`        // Filter by public status
    Gzip         *bool    `json:"gzip,omitempty"`          // Filter by gzip compression
    CollectionID string   `json:"collection_id,omitempty"` // Filter by knowledge collection ID
    Status       string   `json:"status,omitempty"`        // Filter by processing status
    Keywords     string   `json:"keywords,omitempty"`      // Search in filename
    Page         int      `json:"page,omitempty"`          // Page number
    PageSize     int      `json:"pagesize,omitempty"`      // Items per page
    Select       []string `json:"select,omitempty"`        // Fields to return
}
```

#### KnowledgeFilter

```go
type KnowledgeFilter struct {
    UID      string   `json:"uid,omitempty"`      // Filter by user ID
    Name     string   `json:"name,omitempty"`     // Filter by collection name
    Keywords string   `json:"keywords,omitempty"` // Search in name and description
    Public   *bool    `json:"public,omitempty"`   // Filter by public status
    Readonly *bool    `json:"readonly,omitempty"` // Filter by readonly status
    System   *bool    `json:"system,omitempty"`   // Filter by system status
    Page     int      `json:"page,omitempty"`     // Page number
    PageSize int      `json:"pagesize,omitempty"` // Items per page
    Select   []string `json:"select,omitempty"`   // Fields to return
}
```

## Usage Examples

### 1. Chat Management

```go
// Save chat history
messages := []map[string]interface{}{
    {"role": "user", "content": "Hello, how are you?"},
    {"role": "assistant", "content": "I'm doing well, thank you!"},
}
context := map[string]interface{}{
    "assistant_id": "gpt-4",
    "silent": false,
}
err := store.SaveHistory("user123", messages, "chat456", context)

// Get chat history
history, err := store.GetHistory("user123", "chat456")

// Get chat list with pagination
filter := ChatFilter{
    Page: 1,
    PageSize: 20,
    Order: "desc",
}
chats, err := store.GetChats("user123", filter)

// Update chat title
err = store.UpdateChatTitle("user123", "chat456", "New Chat Title")
```

### 2. Assistant Management

```go
// Create an assistant
assistant := map[string]interface{}{
    "name": "Code Helper",
    "type": "assistant",
    "connector": "gpt-4",
    "description": "A helpful coding assistant",
    "tags": []string{"coding", "development"},
    "sort": 100,
    "options": map[string]interface{}{
        "temperature": 0.7,
        "max_tokens": 2000,
    },
    "prompts": []string{
        "You are a helpful coding assistant.",
    },
    "mentionable": true,
    "automated": true,
}
assistantID, err := store.SaveAssistant(assistant)

// Get assistants with filtering
filter := AssistantFilter{
    Tags: []string{"coding"},
    Keywords: "helper",
    Page: 1,
    PageSize: 10,
}
assistants, err := store.GetAssistants(filter)

// Get specific assistant
assistant, err := store.GetAssistant("assistant123")
```

### 3. Attachment Management

```go
// Save attachment metadata
attachment := map[string]interface{}{
    "file_id": "file123",
    "uid": "user123",
    "manager": "local",
    "content_type": "image/jpeg",
    "name": "profile.jpg",
    "public": false,
    "bytes": 102400,
    "collection_id": "knowledge456",
    "scope": []string{"user", "admin"},
    "status": "uploaded",           // Status: uploading, uploaded, indexing, indexed, upload_failed, index_failed
    "progress": "Upload completed", // Progress information (optional)
    "error": nil,                   // Error message (optional, for failed statuses)
}
fileID, err := store.SaveAttachment(attachment)

// Update attachment status during processing workflow
attachment["status"] = "indexing"
attachment["progress"] = "Processing file for indexing..."
_, err = store.SaveAttachment(attachment)

// Handle failed upload
attachment["status"] = "upload_failed"
attachment["progress"] = nil
attachment["error"] = "Network connection timeout"
_, err = store.SaveAttachment(attachment)

// Complete indexing
attachment["status"] = "indexed"
attachment["progress"] = "File indexed successfully"
attachment["error"] = nil
_, err = store.SaveAttachment(attachment)

// Get attachments with filtering
filter := AttachmentFilter{
    UID: "user123",
    ContentType: "image/jpeg",
    Status: "indexed", // Filter by status
    Page: 1,
    PageSize: 20,
}
attachments, err := store.GetAttachments(filter)

// Get all failed uploads
failedFilter := AttachmentFilter{
    UID: "user123",
    Status: "upload_failed",
    Page: 1,
    PageSize: 10,
}
failedUploads, err := store.GetAttachments(failedFilter)
```

#### Attachment Status Workflow

The attachment system supports a complete file processing workflow with the following status values:

- **`uploading`** (default): File upload is in progress
- **`uploaded`**: File upload completed successfully
- **`indexing`**: File is being processed for search indexing
- **`indexed`**: File has been indexed and is ready for use
- **`upload_failed`**: File upload failed (check `error` field for details)
- **`index_failed`**: File indexing failed (check `error` field for details)

#### Additional Fields

- **`progress`**: Human-readable progress information (string, nullable)
- **`error`**: Error message for failed operations (string, nullable, max 600 characters)

### 4. Knowledge Collection Management

```go
// Create knowledge collection
knowledge := map[string]interface{}{
    "collection_id": "kb123",
    "name": "Programming Guide",
    "description": "Comprehensive programming tutorials and examples",
    "uid": "user123",
    "public": true,
    "readonly": false,
    "sort": 100,
    "option": map[string]interface{}{
        "embedding": "openai",
        "chunk_size": 1000,
    },
    "scope": []string{"developers", "students"},
}
collectionID, err := store.SaveKnowledge(knowledge)

// Get knowledge collections with filtering
filter := KnowledgeFilter{
    UID: "user123",
    Keywords: "programming",
    Public: &[]bool{true}[0],
    Page: 1,
    PageSize: 10,
}
collections, err := store.GetKnowledges(filter)

// Get system knowledge collections
systemFilter := KnowledgeFilter{
    System: &[]bool{true}[0],
    Page: 1,
    PageSize: 20,
}
systemCollections, err := store.GetKnowledges(systemFilter)

// Get readonly knowledge collections with specific fields
readonlyFilter := KnowledgeFilter{
    Readonly: &[]bool{true}[0],
    Select: []string{"collection_id", "name", "description", "sort"},
    Page: 1,
    PageSize: 15,
}
readonlyCollections, err := store.GetKnowledges(readonlyFilter)
```

### 5. Internationalization Support

```go
// Get assistants with locale
assistants, err := store.GetAssistants(filter, "zh-CN")

// Get chat with locale
chat, err := store.GetChat("user123", "chat456", "en-US")
```

### 6. Advanced Filtering and Sorting

```go
// Complex assistant filtering
filter := AssistantFilter{
    Tags: []string{"ai", "assistant"},
    Keywords: "helpful",
    Connector: "gpt-4",
    Mentionable: &[]bool{true}[0],
    BuiltIn: &[]bool{false}[0],
    Select: []string{"assistant_id", "name", "description", "tags"},
    Page: 1,
    PageSize: 50,
}
assistants, err := store.GetAssistants(filter)

// Results are automatically sorted by:
// 1. sort field (ASC) - lower numbers appear first
// 2. created_at/updated_at (DESC) - newer items appear first
```

## Testing

### Running Tests

```bash
# Run all tests
go test -v

# Run specific test
go test -run TestXunKnowledgeCRUD -v

# Run with coverage
go test -cover
```

### Test Structure

The test suite includes comprehensive coverage for:

- **CRUD Operations**: Create, Read, Update, Delete for all entities
- **Filtering**: Various filter combinations and edge cases
- **Sorting**: Verify sort order and pagination
- **Error Handling**: Invalid inputs and edge cases
- **Internationalization**: Locale-specific operations
- **Concurrency**: Multiple concurrent operations

### Test Database Setup

Tests use isolated table prefixes to avoid conflicts:

```go
store, err := NewXun(Setting{
    Connector: "default",
    Prefix:    "__unit_test_conversation_",
    TTL:       3600,
})
```

## Performance Considerations

### Database Optimization

1. **Indexes**: All frequently queried fields have indexes
2. **TTL**: Automatic cleanup of expired data
3. **Pagination**: All list operations support pagination
4. **Connection Pooling**: Efficient database connection management

### Caching Strategy

1. **Redis Backend**: For high-frequency read operations
2. **Memory Caching**: In-application caching for static data
3. **Query Optimization**: Efficient filtering and sorting

### Scaling

1. **Horizontal Scaling**: MongoDB support for distributed deployments
2. **Read Replicas**: Database read/write splitting
3. **Sharding**: Data partitioning strategies

## Migration and Upgrades

### Schema Evolution

The Xun backend automatically handles schema migrations:

- New tables are created automatically
- New fields are added with default values
- Indexes are created during initialization

### Data Migration

When switching between backends:

1. Export data from source backend
2. Transform data format if necessary
3. Import to target backend
4. Verify data integrity

## Security

### Access Control

1. **User Isolation**: All operations are user-scoped
2. **Permission System**: Fine-grained access control
3. **Public/Private Flags**: Content visibility management

### Data Protection

1. **Input Validation**: All inputs are validated and sanitized
2. **SQL Injection Prevention**: Parameterized queries
3. **XSS Protection**: Content encoding and sanitization

## Troubleshooting

### Common Issues

1. **Connection Errors**: Check connector configuration
2. **Schema Errors**: Verify database permissions
3. **Performance Issues**: Check indexes and query patterns
4. **Memory Issues**: Monitor TTL and cleanup processes

### Debugging

Enable debug logging:

```go
import "github.com/yaoapp/kun/log"

log.SetLevel(log.DebugLevel)
```

### Monitoring

Key metrics to monitor:

- Database connection pool usage
- Query performance and slow queries
- Memory usage and garbage collection
- TTL cleanup effectiveness

## Contributing

### Development Setup

1. Clone the repository
2. Install dependencies: `go mod download`
3. Run tests: `go test -v`
4. Follow Go coding standards

### Adding New Features

1. Update the Store interface
2. Implement in all backends (Xun, Redis, MongoDB)
3. Add comprehensive tests
4. Update documentation

## License

This project is part of the Yao App Engine and follows the same license terms.
