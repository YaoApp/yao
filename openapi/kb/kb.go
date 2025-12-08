package kb

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func init() {
	// Register kb process handlers
	process.RegisterGroup("kb", map[string]process.Handler{
		// Collection processes
		"collection.create":         ProcessCreateCollection,
		"collection.remove":         ProcessRemoveCollection,
		"collection.get":            ProcessGetCollection,
		"collection.exists":         ProcessCollectionExists,
		"collection.list":           ProcessListCollections,
		"collection.updatemetadata": ProcessUpdateCollectionMetadata,

		// Document processes
		"documents.addfile": ProcessAddFile,
		"documents.addtext": ProcessAddText,
		"documents.addurl":  ProcessAddURL,
	})
}

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
	group.GET("/collections", ListCollections)
	group.GET("/collections/:collectionID", GetCollection)
	group.GET("/collections/:collectionID/exists", CollectionExists)
	group.POST("/collections", CreateCollection)
	group.PUT("/collections/:collectionID/metadata", UpdateCollectionMetadata)
	group.DELETE("/collections/:collectionID", RemoveCollection)

	// Document Management
	group.GET("/documents", ListDocuments)
	group.GET("/documents/:docID", GetDocument)
	group.POST("/collections/:collectionID/documents/file", AddFile)
	group.POST("/collections/:collectionID/documents/file/async", AddFileAsync)
	group.POST("/collections/:collectionID/documents/text", AddText)
	group.POST("/collections/:collectionID/documents/text/async", AddTextAsync)
	group.POST("/collections/:collectionID/documents/url", AddURL)
	group.POST("/collections/:collectionID/documents/url/async", AddURLAsync)
	group.DELETE("/documents", RemoveDocs)

	// Segment Management
	group.GET("/documents/:docID/segments", ScrollSegments)
	group.GET("/documents/:docID/segments/search", GetSegments)
	group.GET("/documents/:docID/segments/:segmentID", GetSegment)
	group.GET("/documents/:docID/segments/:segmentID/parents", GetSegmentParents)
	group.POST("/documents/:docID/segments", AddSegments)
	group.POST("/documents/:docID/segments/async", AddSegmentsAsync)
	group.PUT("/documents/:docID/segments", UpdateSegments)
	group.PUT("/documents/:docID/segments/async", UpdateSegmentsAsync)
	group.DELETE("/documents/:docID/segments", RemoveSegments)
	group.DELETE("/documents/:docID/segments/all", RemoveSegmentsByDocID)

	// Segment Graph Management
	group.GET("/documents/:docID/segments/:segmentID/graph", GetSegmentGraph)
	group.GET("/documents/:docID/segments/:segmentID/entities", GetSegmentEntities)
	group.GET("/documents/:docID/segments/:segmentID/relationships", GetSegmentRelationships)
	group.GET("/documents/:docID/segments/:segmentID/relationships/by-entities", GetSegmentRelationshipsByEntities)
	group.POST("/documents/:docID/segments/:segmentID/extract", ExtractSegmentGraph)
	group.POST("/documents/:docID/segments/:segmentID/extract/async", ExtractSegmentGraphAsync)

	// Segment score and weight management (batch operations)
	group.PUT("/documents/:docID/segments/scores", UpdateScores)
	group.PUT("/documents/:docID/segments/weights", UpdateWeights)

	// Segment votes management
	group.GET("/documents/:docID/segments/:segmentID/votes", ScrollVotes)
	group.GET("/documents/:docID/segments/:segmentID/votes/search", GetVotes)
	group.GET("/documents/:docID/segments/:segmentID/votes/:voteID", GetVote)
	group.POST("/documents/:docID/segments/:segmentID/votes", AddVotes)
	group.DELETE("/documents/:docID/segments/:segmentID/votes", RemoveVotes)

	// Segment hits management
	group.GET("/documents/:docID/segments/:segmentID/hits", ScrollHits)
	group.GET("/documents/:docID/segments/:segmentID/hits/search", GetHits)
	group.GET("/documents/:docID/segments/:segmentID/hits/:hitID", GetHit)
	group.POST("/documents/:docID/segments/:segmentID/hits", AddHits)
	group.DELETE("/documents/:docID/segments/:segmentID/hits", RemoveHits)

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
