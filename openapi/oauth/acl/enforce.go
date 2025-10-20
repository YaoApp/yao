package acl

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Enforce checks if a user has access to a resource based on the request context
func (acl *ACL) Enforce(c *gin.Context) (bool, error) {
	// If ACL is not enabled, allow access
	if !acl.Enabled() {
		return true, nil
	}

	// If scope manager not loaded, deny access
	if acl.Scope == nil {
		return false, nil
	}

	// Get authorized info from context (set by OAuth guard middleware)
	authInfo := authorized.GetInfo(c)

	// Resolve all scopes (client + user + team) from authorized info
	// Note: This should include scope expansion from roles, aliases, etc.
	scopes := getScopes(authInfo)

	// Build access request (focused on scope-based access control)
	request := &AccessRequest{
		Method: c.Request.Method,
		Path:   c.Request.URL.Path,
		Scopes: scopes,
	}

	// Check scopes
	decision := acl.Scope.Check(request)

	if !decision.Allowed {
		// Return 403 Forbidden with details
		c.JSON(403, map[string]interface{}{
			"code":            403,
			"message":         "Access denied",
			"reason":          decision.Reason,
			"required_scopes": decision.RequiredScopes,
			"missing_scopes":  decision.MissingScopes,
		})
		c.Abort()
		return false, nil
	}

	return true, nil
}

// getScopes resolves all scopes from authorized info
// This function is responsible for the complete scope resolution process:
// 1. Get base scopes from token (authInfo.Scope)
// 2. Get user role scopes from database (if authInfo.UserID exists)
// 3. Get team role scopes from database (if authInfo.TeamID exists)
// 4. Merge all scopes and return the complete list
//
// Scope resolution logic:
// - Pure API call (no user_id): Returns client scopes from token
// - User call: Returns merged scopes (client + user roles + team roles)
//
// This keeps the ACL layer focused on scope-based access control,
// while relying on the authorized package for context extraction.
func getScopes(authInfo *types.AuthorizedInfo) []string {
	// TODO: Implement scope resolution
	// 1. Parse base scopes from authInfo.Scope (space-separated)
	// 2. Query user roles and convert to scopes (if authInfo.UserID exists)
	// 3. Query team roles and convert to scopes (if authInfo.TeamID exists)
	// 4. Merge and deduplicate all scopes

	// For now, just return scopes from token
	if authInfo.Scope == "" {
		return []string{}
	}
	// Split space-separated scopes
	// e.g., "read:users write:users" -> ["read:users", "write:users"]
	return strings.Split(authInfo.Scope, " ")
}
