# KB API

The `kb/api` package provides a unified Go API for Knowledge Base operations including collection management, document ingestion, and semantic search.

## Quick Start

```go
import (
    "context"
    "github.com/yaoapp/yao/kb"
    "github.com/yaoapp/yao/kb/api"
)

// After kb.Load(), use kb.API to access all operations
ctx := context.Background()
```

## API Interface

```go
type API interface {
    // Collection operations
    CreateCollection(ctx, params) (*CreateCollectionResult, error)
    RemoveCollection(ctx, collectionID) (*RemoveCollectionResult, error)
    GetCollection(ctx, collectionID) (map[string]interface{}, error)
    CollectionExists(ctx, collectionID) (*CollectionExistsResult, error)
    ListCollections(ctx, filter) (*ListCollectionsResult, error)
    UpdateCollectionMetadata(ctx, collectionID, params) (*UpdateMetadataResult, error)

    // Document operations
    ListDocuments(ctx, filter) (*ListDocumentsResult, error)
    GetDocument(ctx, docID, params) (map[string]interface{}, error)
    RemoveDocuments(ctx, params) (*RemoveDocumentsResult, error)

    // Document add operations (sync)
    AddFile(ctx, params) (*AddDocumentResult, error)
    AddText(ctx, params) (*AddDocumentResult, error)
    AddURL(ctx, params) (*AddDocumentResult, error)

    // Document add operations (async)
    AddFileAsync(ctx, params) (*AddDocumentAsyncResult, error)
    AddTextAsync(ctx, params) (*AddDocumentAsyncResult, error)
    AddURLAsync(ctx, params) (*AddDocumentAsyncResult, error)

    // Search operations
    Search(ctx, queries) (*SearchResult, error)
}
```

## Collection Operations

### Create Collection

```go
params := &api.CreateCollectionParams{
    ID: "my_collection",
    Metadata: map[string]interface{}{
        "name":        "My Knowledge Base",
        "description": "Collection description",
    },
    EmbeddingProviderID: "__yao.openai",
    EmbeddingOptionID:   "text-embedding-3-small",
    Locale:              "en",
    Config: &types.CreateCollectionOptions{
        Distance:  "cosine",
        IndexType: "hnsw",
    },
}

result, err := kb.API.CreateCollection(ctx, params)
// result.CollectionID = "my_collection"
```

### Get Collection

```go
collection, err := kb.API.GetCollection(ctx, "my_collection")
// collection["id"], collection["name"], collection["config"], etc.
```

### List Collections

```go
filter := &api.ListCollectionsFilter{
    Page:     1,
    PageSize: 20,
    Keywords: "knowledge",
    Status:   []string{"active"},
}

result, err := kb.API.ListCollections(ctx, filter)
// result.Data, result.Total, result.PageCnt
```

### Remove Collection

```go
result, err := kb.API.RemoveCollection(ctx, "my_collection")
// result.Removed = true
```

## Document Operations

### Add Text

```go
params := &api.AddTextParams{
    CollectionID: "my_collection",
    Text:         "Einstein developed the theory of relativity...",
    DocID:        "einstein_bio", // optional, auto-generated if empty
    Metadata: map[string]interface{}{
        "title":  "Einstein Biography",
        "author": "John Doe",
    },
    Chunking: &api.ProviderConfigParams{
        ProviderID: "__yao.structured",
        OptionID:   "standard",
    },
    Embedding: &api.ProviderConfigParams{
        ProviderID: "__yao.openai",
        OptionID:   "text-embedding-3-small",
    },
    Extraction: &api.ProviderConfigParams{ // optional, for graph extraction
        ProviderID: "__yao.openai",
        OptionID:   "gpt-4o-mini",
    },
}

result, err := kb.API.AddText(ctx, params)
// result.DocID = "einstein_bio"
```

### Add File

```go
params := &api.AddFileParams{
    CollectionID: "my_collection",
    FileID:       "uploaded_file_id",
    Uploader:     "local", // or "s3", etc.
    Chunking:     &api.ProviderConfigParams{...},
    Embedding:    &api.ProviderConfigParams{...},
}

result, err := kb.API.AddFile(ctx, params)
```

### Add URL

```go
params := &api.AddURLParams{
    CollectionID: "my_collection",
    URL:          "https://example.com/article",
    Chunking:     &api.ProviderConfigParams{...},
    Embedding:    &api.ProviderConfigParams{...},
}

result, err := kb.API.AddURL(ctx, params)
```

### List Documents

```go
filter := &api.ListDocumentsFilter{
    Page:         1,
    PageSize:     20,
    CollectionID: "my_collection",
    Status:       []string{"active"},
}

result, err := kb.API.ListDocuments(ctx, filter)
```

### Remove Documents

```go
params := &api.RemoveDocumentsParams{
    DocumentIDs: []string{"doc1", "doc2"},
}

result, err := kb.API.RemoveDocuments(ctx, params)
```

## Search Operations

The Search API supports batch queries with three search modes:

| Mode     | Description                                            |
| -------- | ------------------------------------------------------ |
| `vector` | Pure vector similarity search                          |
| `graph`  | Graph traversal to find related segments via entities  |
| `expand` | Graph-based entity expansion + vector search (default) |

### Basic Vector Search

```go
queries := []api.Query{
    {
        CollectionID: "my_collection",
        Input:        "What is the theory of relativity?",
        Mode:         api.SearchModeVector,
        PageSize:     10,
    },
}

result, err := kb.API.Search(ctx, queries)
// result.Segments - matched text segments with scores
// result.Total - total count
```

### Graph-Enhanced Search (Expand Mode)

```go
queries := []api.Query{
    {
        CollectionID: "my_collection",
        Input:        "Einstein's contributions to physics",
        Mode:         api.SearchModeExpand, // default
        MaxDepth:     2, // graph traversal depth
        PageSize:     10,
    },
}

result, err := kb.API.Search(ctx, queries)
// result.Segments - segments from vector + graph expansion
// result.Graph.Nodes - related entities
// result.Graph.Relationships - entity relationships
```

### Multi-Query Search

Queries can span multiple collections; results are merged and deduplicated:

```go
queries := []api.Query{
    {
        CollectionID: "science_kb",
        Input:        "quantum mechanics",
        Mode:         api.SearchModeVector,
    },
    {
        CollectionID: "tech_kb",
        Input:        "machine learning",
        Mode:         api.SearchModeVector,
    },
}

result, err := kb.API.Search(ctx, queries)
// Merged results from both collections
```

### Search with Messages (Conversation Context)

```go
queries := []api.Query{
    {
        CollectionID: "my_collection",
        Messages: []types.ChatMessage{
            {Role: "user", Content: "Tell me about Einstein"},
            {Role: "assistant", Content: "Einstein was a physicist..."},
            {Role: "user", Content: "What about his discoveries?"}, // used as query
        },
        Mode: api.SearchModeExpand,
    },
}

result, err := kb.API.Search(ctx, queries)
```

### Search with Filters

```go
queries := []api.Query{
    {
        CollectionID: "my_collection",
        Input:        "physics",
        DocumentID:   "specific_doc_id", // filter to specific document
        Threshold:    0.5,               // similarity threshold
        Metadata: map[string]interface{}{
            "category": "science",
        },
        Page:     1,
        PageSize: 20,
    },
}

result, err := kb.API.Search(ctx, queries)
```

## Query Parameters

| Field          | Type          | Description                                            |
| -------------- | ------------- | ------------------------------------------------------ |
| `CollectionID` | string        | Collection to search (required)                        |
| `Input`        | string        | Direct query text                                      |
| `Messages`     | []ChatMessage | Conversation history (last user message used as query) |
| `Mode`         | SearchMode    | `vector`, `graph`, or `expand` (default: `expand`)     |
| `DocumentID`   | string        | Filter to specific document                            |
| `Threshold`    | float64       | Similarity threshold (0-1)                             |
| `Metadata`     | map           | Filter by metadata fields                              |
| `MaxDepth`     | int           | Graph traversal depth (default: 2)                     |
| `Page`         | int           | Page number (1-based)                                  |
| `PageSize`     | int           | Results per page                                       |

## Search Result

```go
type SearchResult struct {
    Segments   []types.Segment // Matched segments with scores
    Graph      *GraphData      // Nodes and relationships (graph/expand mode)
    Total      int             // Total results count
    Page       int             // Current page
    PageSize   int             // Results per page
    TotalPages int             // Total pages
    Next       int             // Next page number
    Prev       int             // Previous page number
}
```

## Provider Configuration

Providers handle text processing:

```go
type ProviderConfigParams struct {
    ProviderID string // e.g., "__yao.openai", "__yao.structured"
    OptionID   string // e.g., "text-embedding-3-small", "gpt-4o-mini"
}
```

Common providers:

- **Chunking**: `__yao.structured` - text splitting
- **Embedding**: `__yao.openai` - vector embeddings
- **Extraction**: `__yao.openai` - entity/relationship extraction for graph
