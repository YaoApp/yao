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
	group.GET("/collections/:collectionID/exists", CollectionExists)
	group.GET("/collections", GetCollections)

	// Document Management
	group.POST("/collections/:collectionID/documents/file", AddFile)
	group.POST("/collections/:collectionID/documents/text", AddText)
	group.POST("/collections/:collectionID/documents/url", AddURL)
	group.GET("/documents", ListDocuments)
	group.GET("/documents/scroll", ScrollDocuments)
	group.GET("/documents/:docID", GetDocument)
	group.DELETE("/documents", RemoveDocs)

	// Segment Management
	group.POST("/documents/:docID/segments", AddSegments)
	group.PUT("/segments", UpdateSegments)
	group.DELETE("/segments", RemoveSegments)
	group.DELETE("/documents/:docID/segments", RemoveSegmentsByDocID)
	group.GET("/segments", GetSegments)
	group.GET("/segments/:segmentID", GetSegment)
	group.GET("/documents/:docID/segments", ListSegments)
	group.GET("/documents/:docID/segments/scroll", ScrollSegments)

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
}
