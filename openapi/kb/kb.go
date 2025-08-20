package kb

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Attach attaches the Knowledge Base API to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// Validate the GraphRag instance
	if kb.Instance == nil {
		log.Warn("[OpenAPI] GraphRag instance is not set, please check the configuration")
		return
	}

	// Protect all endpoints with OAuth
	group.Use(oauth.Guard)

	// Collection Management
	group.POST("/collections", CreateCollection)
	group.DELETE("/collections/:collectionID", RemoveCollection)
	group.GET("/collections/:collectionID", GetCollection)
	group.GET("/collections/:collectionID/exists", CollectionExists)
	group.GET("/collections", GetCollections)
	group.PUT("/collections/:collectionID/metadata", UpdateCollectionMetadata)

	// Document Management
	group.POST("/collections/:collectionID/documents/file", AddFile)
	group.POST("/collections/:collectionID/documents/file/async", AddFileAsync)
	group.POST("/collections/:collectionID/documents/text", AddText)
	group.POST("/collections/:collectionID/documents/text/async", AddTextAsync)
	group.POST("/collections/:collectionID/documents/url", AddURL)
	group.POST("/collections/:collectionID/documents/url/async", AddURLAsync)
	group.GET("/documents", ListDocuments)
	group.GET("/documents/:docID", GetDocument)
	group.DELETE("/documents", RemoveDocs)

	// Segment Management
	group.POST("/documents/:docID/segments", AddSegments)
	group.PUT("/documents/:docID/segments", UpdateSegments)
	group.DELETE("/documents/:docID/segments", RemoveSegmentsByDocID)
	group.GET("/documents/:docID/segments", ScrollSegments)

	// Global segment operations (not tied to specific document)
	group.DELETE("/segments", RemoveSegments)
	group.GET("/segments", GetSegments)
	group.GET("/segments/:segmentID", GetSegment)

	// Segment Voting, Scoring, Weighting
	group.PUT("/segments/vote", UpdateVote)
	group.PUT("/segments/score", UpdateScore)
	group.PUT("/segments/weight", UpdateWeight)

	// Search Management
	group.POST("/search", Search)
	group.POST("/search/multi", MultiSearch)

	// Collection Backup and Restore
	group.POST("/collections/:collectionID/backup", Backup)
	group.POST("/collections/:collectionID/restore", Restore)

	// Provider Management (Chunking, Converter, Embedding, Extraction, Fetcher ...)
	group.GET("/providers/:providerType", GetProviders)
	group.GET("/providers/:providerType/:providerID/schema", GetProviderSchema)
}
