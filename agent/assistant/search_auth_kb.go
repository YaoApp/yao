package assistant

import (
	"context"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/kb"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// FilterKBCollectionsByAuth filters collections based on user authorization.
// Returns only collections that the user has permission to access.
// Permission is determined by Collection's metadata (public, share, __yao_team_id, __yao_created_by).
func FilterKBCollectionsByAuth(ctx *agentContext.Context, collections []string) []string {
	if ctx == nil || ctx.Authorized == nil {
		return collections // No auth context, return all
	}

	authInfo := ctx.Authorized

	// No constraints, return all collections
	if !authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
		return collections
	}

	// Check KB API
	if kb.API == nil {
		return collections // KB not initialized, return all
	}

	var allowed []string
	bgCtx := context.Background()

	for _, collectionID := range collections {
		// Get collection metadata
		collection, err := kb.API.GetCollection(bgCtx, collectionID)
		if err != nil {
			continue // Skip if can't get collection
		}

		if hasCollectionAccess(authInfo, collection) {
			allowed = append(allowed, collectionID)
		}
	}

	return allowed
}

// hasCollectionAccess checks if user has access to a collection based on its metadata.
func hasCollectionAccess(authInfo *oauthtypes.AuthorizedInfo, collection map[string]interface{}) bool {
	if authInfo == nil {
		return true
	}

	// No constraints, allow access
	if !authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
		return true
	}

	// Check public access (handle different types: bool, int, float64)
	if isPublicValue(collection["public"]) {
		return true
	}

	// Get metadata for permission fields
	metadata, _ := collection["metadata"].(map[string]interface{})
	if metadata == nil {
		metadata = collection
	}

	// Team only check
	if authInfo.Constraints.TeamOnly && authInfo.TeamID != "" {
		teamID, _ := metadata["__yao_team_id"].(string)
		if teamID == "" {
			teamID, _ = collection["__yao_team_id"].(string)
		}

		if teamID == authInfo.TeamID {
			createdBy, _ := metadata["__yao_created_by"].(string)
			if createdBy == "" {
				createdBy, _ = collection["__yao_created_by"].(string)
			}
			share, _ := metadata["share"].(string)
			if share == "" {
				share, _ = collection["share"].(string)
			}

			if createdBy == authInfo.UserID || share == "team" {
				return true
			}
		}
	}

	// Owner only check
	if authInfo.Constraints.OwnerOnly && authInfo.UserID != "" {
		createdBy, _ := metadata["__yao_created_by"].(string)
		if createdBy == "" {
			createdBy, _ = collection["__yao_created_by"].(string)
		}
		if createdBy == authInfo.UserID {
			return true
		}
	}

	return false
}

// isPublicValue checks if a value represents "public" access
func isPublicValue(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val == 1
	case int64:
		return val == 1
	case float64:
		return val == 1
	case string:
		return val == "true" || val == "1"
	}
	return false
}
