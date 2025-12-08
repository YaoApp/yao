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

	// Document operations (future)
	// AddDocument(ctx context.Context, params *AddDocumentParams) (*AddDocumentResult, error)
	// RemoveDocument(ctx context.Context, documentID string) (*RemoveDocumentResult, error)
	// ...

	// Segment operations (future)
	// ...
}

// KBInstance holds the KB instance dependencies required by the API
type KBInstance struct {
	GraphRag  types.GraphRag          // GraphRag instance for vector/graph operations
	Config    *kbtypes.Config         // KB configuration
	Providers *kbtypes.ProviderConfig // Provider configurations
}
