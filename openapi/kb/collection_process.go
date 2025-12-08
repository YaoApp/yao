package kb

import (
	"encoding/json"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/kb"
	kbapi "github.com/yaoapp/yao/kb/api"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// ProcessCreateCollection creates a new collection via Yao process
// Process: kb.collection.Create
//
// Args[0]: params (map) - Collection creation parameters
//
//	{
//	  "id": "collection_id",
//	  "metadata": {
//	    "name": "Collection Name",
//	    "description": "Description"
//	  },
//	  "embedding_provider_id": "__yao.openai",
//	  "embedding_option_id": "text-embedding-3-small",
//	  "locale": "en",
//	  "config": {
//	    "distance": "cosine",
//	    "index_type": "hnsw",
//	    "m": 16,
//	    "ef_construction": 200,
//	    "ef_search": 64
//	  }
//	}
//
// Returns: map with collection_id and message
func ProcessCreateCollection(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	if kb.API == nil {
		exception.New("Knowledge base not initialized", 500).Throw()
	}

	// Get authorized info from process
	authInfo := authorized.ProcessAuthInfo(process)

	// Parse parameters using JSON for type safety
	paramsJSON, err := json.Marshal(process.Args[0])
	if err != nil {
		exception.New("Failed to encode parameters: "+err.Error(), 400).Throw()
	}

	var params kbapi.CreateCollectionParams
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		exception.New("Failed to decode parameters: "+err.Error(), 400).Throw()
	}

	// Apply auth scope from authorized info
	if authInfo != nil {
		authScope := authInfo.WithCreateScope(maps.MapStrAny{})
		params.AuthScope = authScope
	}

	// Call API
	result, err := kb.API.CreateCollection(process.Context, &params)
	if err != nil {
		log.Error("Failed to create collection: %v", err)
		exception.New(err.Error(), 500).Throw()
	}

	return maps.MapStrAny{
		"collection_id": result.CollectionID,
		"message":       result.Message,
	}
}

// ProcessRemoveCollection removes a collection via Yao process
// Process: kb.collection.Remove
//
// Args[0]: collection_id (string) - Collection ID to remove
//
// Returns: map with collection_id, removed status, documents_removed count, and message
func ProcessRemoveCollection(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	if kb.API == nil {
		exception.New("Knowledge base not initialized", 500).Throw()
	}

	// Get authorized info from process
	authInfo := authorized.ProcessAuthInfo(process)

	collectionID := process.ArgsString(0)
	if collectionID == "" {
		exception.New("Collection ID is required", 400).Throw()
	}

	// Check remove permission
	hasPermission, err := checkCollectionPermission(authInfo, collectionID)
	if err != nil {
		exception.New(err.Error(), 403).Throw()
	}

	if !hasPermission {
		exception.New("Forbidden: No permission to remove collection", 403).Throw()
	}

	result, err := kb.API.RemoveCollection(process.Context, collectionID)
	if err != nil {
		log.Error("Failed to remove collection: %v", err)
		exception.New(err.Error(), 500).Throw()
	}

	return maps.MapStrAny{
		"collection_id":     result.CollectionID,
		"removed":           result.Removed,
		"documents_removed": result.DocumentsRemoved,
		"message":           result.Message,
	}
}

// ProcessGetCollection retrieves a collection by ID via Yao process
// Process: kb.collection.Get
//
// Args[0]: collection_id (string) - Collection ID to retrieve
//
// Returns: map containing collection details
func ProcessGetCollection(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	if kb.API == nil {
		exception.New("Knowledge base not initialized", 500).Throw()
	}

	collectionID := process.ArgsString(0)
	if collectionID == "" {
		exception.New("Collection ID is required", 400).Throw()
	}

	collection, err := kb.API.GetCollection(process.Context, collectionID)
	if err != nil {
		log.Error("Failed to get collection: %v", err)
		exception.New(err.Error(), 500).Throw()
	}

	return collection
}

// ProcessCollectionExists checks if a collection exists via Yao process
// Process: kb.collection.Exists
//
// Args[0]: collection_id (string) - Collection ID to check
//
// Returns: map with collection_id and exists status
func ProcessCollectionExists(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	if kb.API == nil {
		exception.New("Knowledge base not initialized", 500).Throw()
	}

	collectionID := process.ArgsString(0)
	if collectionID == "" {
		exception.New("Collection ID is required", 400).Throw()
	}

	result, err := kb.API.CollectionExists(process.Context, collectionID)
	if err != nil {
		log.Error("Failed to check collection existence: %v", err)
		exception.New(err.Error(), 500).Throw()
	}

	return maps.MapStrAny{
		"collection_id": result.CollectionID,
		"exists":        result.Exists,
	}
}

// ProcessListCollections lists collections with pagination via Yao process
// Process: kb.collection.List
//
// Args[0]: filter (map) - Optional filter parameters
//
//	{
//	  "page": 1,
//	  "pagesize": 20,
//	  "keywords": "search term",
//	  "status": ["active"],
//	  "embedding_provider_id": "__yao.openai",
//	  "system": false,
//	  "select": ["id", "name", "status"],
//	  "sort": [{"column": "created_at", "option": "desc"}]
//	}
//
// Returns: map with data array and pagination info
func ProcessListCollections(process *process.Process) interface{} {

	if kb.API == nil {
		exception.New("Knowledge base not initialized", 500).Throw()
	}

	// Get authorized info from process
	authInfo := authorized.ProcessAuthInfo(process)

	// Default filter
	filter := &kbapi.ListCollectionsFilter{
		Page:     kbapi.DefaultPage,
		PageSize: kbapi.DefaultPageSize,
	}

	// Parse filter parameters using JSON (optional)
	if process.NumOfArgs() > 0 {
		filterJSON, err := json.Marshal(process.Args[0])
		if err != nil {
			exception.New("Failed to encode filter: "+err.Error(), 400).Throw()
		}

		if err := json.Unmarshal(filterJSON, filter); err != nil {
			exception.New("Failed to decode filter: "+err.Error(), 400).Throw()
		}
	}

	// Apply auth filters from authorized info
	if authInfo != nil {
		filter.AuthFilters = processAuthFilter(authInfo)
	}

	result, err := kb.API.ListCollections(process.Context, filter)
	if err != nil {
		log.Error("Failed to list collections: %v", err)
		exception.New(err.Error(), 500).Throw()
	}

	return maps.MapStrAny{
		"data":     result.Data,
		"next":     result.Next,
		"prev":     result.Prev,
		"page":     result.Page,
		"pagesize": result.PageSize,
		"total":    result.Total,
		"pagecnt":  result.PageCnt,
	}
}

// ProcessUpdateCollectionMetadata updates collection metadata via Yao process
// Process: kb.collection.UpdateMetadata
//
// Args[0]: collection_id (string) - Collection ID
// Args[1]: params (map) - Update parameters
//
//	{
//	  "metadata": {
//	    "name": "New Name",
//	    "description": "New Description"
//	  }
//	}
//
// Returns: map with collection_id and message
func ProcessUpdateCollectionMetadata(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	if kb.API == nil {
		exception.New("Knowledge base not initialized", 500).Throw()
	}

	// Get authorized info from process
	authInfo := authorized.ProcessAuthInfo(process)

	collectionID := process.ArgsString(0)
	if collectionID == "" {
		exception.New("Collection ID is required", 400).Throw()
	}

	// Check update permission
	hasPermission, err := checkCollectionPermission(authInfo, collectionID)
	if err != nil {
		exception.New(err.Error(), 403).Throw()
	}

	if !hasPermission {
		exception.New("Forbidden: No permission to update collection", 403).Throw()
	}

	// Parse parameters using JSON
	paramsJSON, err := json.Marshal(process.Args[1])
	if err != nil {
		exception.New("Failed to encode parameters: "+err.Error(), 400).Throw()
	}

	var params kbapi.UpdateMetadataParams
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		exception.New("Failed to decode parameters: "+err.Error(), 400).Throw()
	}

	if len(params.Metadata) == 0 {
		exception.New("Metadata is required and cannot be empty", 400).Throw()
	}

	// Apply auth scope from authorized info
	if authInfo != nil {
		authScope := authInfo.WithUpdateScope(maps.MapStrAny{})
		params.AuthScope = authScope
	}

	result, err := kb.API.UpdateCollectionMetadata(process.Context, collectionID, &params)
	if err != nil {
		log.Error("Failed to update collection metadata: %v", err)
		exception.New(err.Error(), 500).Throw()
	}

	return maps.MapStrAny{
		"collection_id": result.CollectionID,
		"message":       result.Message,
	}
}

// Helper functions for Process handlers

// processAuthFilter applies permission-based filtering to query wheres for process handlers
// This function builds where clauses based on the user's authorization constraints
func processAuthFilter(authInfo *oauthtypes.AuthorizedInfo) []model.QueryWhere {
	if authInfo == nil {
		return []model.QueryWhere{}
	}

	var wheres []model.QueryWhere
	scope := authInfo.AccessScope()

	// Team only - User can access:
	// 1. Public records (public = true)
	// 2. Records in their team where:
	//    - They created the record (__yao_created_by matches)
	//    - OR the record is shared with team (share = "team")
	if authInfo.Constraints.TeamOnly && authInfo.TeamID != "" && authInfo.UserID != "" {
		wheres = append(wheres, model.QueryWhere{
			Wheres: []model.QueryWhere{
				{Column: "public", Value: true, Method: "orwhere"},
				{Wheres: []model.QueryWhere{
					{Column: "__yao_team_id", Value: scope.TeamID},
					{Wheres: []model.QueryWhere{
						{Column: "__yao_created_by", Value: scope.CreatedBy},
						{Column: "share", Value: "team", Method: "orwhere"},
					}},
				}, Method: "orwhere"},
			},
		})
		return wheres
	}

	// Owner only - User can access:
	// 1. Public records (public = true)
	// 2. Records they created where:
	//    - __yao_team_id is null (not team records)
	//    - __yao_created_by matches their user ID
	if authInfo.Constraints.OwnerOnly && authInfo.UserID != "" {
		wheres = append(wheres, model.QueryWhere{
			Wheres: []model.QueryWhere{
				{Column: "public", Value: true, Method: "orwhere"},
				{Wheres: []model.QueryWhere{
					{Column: "__yao_team_id", OP: "null"},
					{Column: "__yao_created_by", Value: scope.CreatedBy},
				}, Method: "orwhere"},
			},
		})
		return wheres
	}

	return wheres
}
