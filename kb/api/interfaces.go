package api

import (
	"context"

	"github.com/yaoapp/gou/graphrag/types"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// API defines the unified interface for all KB operations
type API interface {
	// Collection operations
	CreateCollection(ctx context.Context, params *CreateCollectionParams) (*CreateCollectionResult, error)
	RemoveCollection(ctx context.Context, collectionID string) (*RemoveCollectionResult, error)
	GetCollection(ctx context.Context, collectionID string) (map[string]interface{}, error)
	CollectionExists(ctx context.Context, collectionID string) (*CollectionExistsResult, error)
	ListCollections(ctx context.Context, filter *ListCollectionsFilter) (*ListCollectionsResult, error)
	UpdateCollectionMetadata(ctx context.Context, collectionID string, params *UpdateMetadataParams) (*UpdateMetadataResult, error)

	// Document operations
	ListDocuments(ctx context.Context, filter *ListDocumentsFilter) (*ListDocumentsResult, error)
	GetDocument(ctx context.Context, docID string, params *GetDocumentParams) (map[string]interface{}, error)
	GetDocumentsContent(ctx context.Context, docIDs []string) ([]map[string]interface{}, error)
	RemoveDocuments(ctx context.Context, params *RemoveDocumentsParams) (*RemoveDocumentsResult, error)

	// Document add operations (sync)
	AddFile(ctx context.Context, params *AddFileParams) (*AddDocumentResult, error)
	AddText(ctx context.Context, params *AddTextParams) (*AddDocumentResult, error)
	AddURL(ctx context.Context, params *AddURLParams) (*AddDocumentResult, error)

	// Document add operations (async)
	AddFileAsync(ctx context.Context, params *AddFileParams) (*AddDocumentAsyncResult, error)
	AddTextAsync(ctx context.Context, params *AddTextParams) (*AddDocumentAsyncResult, error)
	AddURLAsync(ctx context.Context, params *AddURLParams) (*AddDocumentAsyncResult, error)

	// Search operations
	Search(ctx context.Context, queries []Query) (*SearchResult, error)
}

// KBInstance holds the KB instance dependencies required by the API
type KBInstance struct {
	GraphRag types.GraphRag // GraphRag instance
	// for vector/graph operations
	Config    *kbtypes.Config         // KB configuration
	Providers *kbtypes.ProviderConfig // Provider configurations
}
