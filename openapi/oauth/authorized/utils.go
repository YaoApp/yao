package authorized

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// ProcessAuthInfo extracts authorized information from the process
func ProcessAuthInfo(p *process.Process) *types.AuthorizedInfo {
	if p == nil {
		return nil
	}

	// Get authorized info from process
	processAuth := p.GetAuthorized()
	if processAuth == nil {
		return nil
	}

	// Convert process.AuthorizedInfo to types.AuthorizedInfo
	info := &types.AuthorizedInfo{
		Subject:    processAuth.Subject,
		ClientID:   processAuth.ClientID,
		UserID:     processAuth.UserID,
		Scope:      processAuth.Scope,
		TeamID:     processAuth.TeamID,
		TenantID:   processAuth.TenantID,
		SessionID:  processAuth.SessionID,
		RememberMe: processAuth.RememberMe,
	}

	// Convert constraints
	info.Constraints = types.DataConstraints{
		OwnerOnly:   processAuth.Constraints.OwnerOnly,
		CreatorOnly: processAuth.Constraints.CreatorOnly,
		EditorOnly:  processAuth.Constraints.EditorOnly,
		TeamOnly:    processAuth.Constraints.TeamOnly,
		Extra:       processAuth.Constraints.Extra,
	}

	return info
}

// GetInfo extracts authorized information from the gin context
// This function reads authorization data that was set by the OAuth guard middleware
func GetInfo(c *gin.Context) *types.AuthorizedInfo {
	info := &types.AuthorizedInfo{}

	if subject, ok := c.Get("__subject"); ok {
		info.Subject = subject.(string)
	}

	if clientID, ok := c.Get("__client_id"); ok {
		info.ClientID = clientID.(string)
	}

	if userID, ok := c.Get("__user_id"); ok {
		info.UserID = userID.(string)
	}

	if scope, ok := c.Get("__scope"); ok {
		info.Scope = scope.(string)
	}

	if teamID, ok := c.Get("__team_id"); ok {
		info.TeamID = teamID.(string)
	}

	if tenantID, ok := c.Get("__tenant_id"); ok {
		info.TenantID = tenantID.(string)
	}

	if sessionID, ok := c.Get("__sid"); ok {
		info.SessionID = sessionID.(string)
	}

	if rememberMe, ok := c.Get("__remember_me"); ok {
		if rmBool, ok := rememberMe.(bool); ok {
			info.RememberMe = rmBool
		}
	}

	// Get data access constraints (set by ACL enforcement)
	info.Constraints = GetConstraints(c)

	return info
}

// IsTeamMember checks if the user is a team member
func IsTeamMember(c *gin.Context) bool {
	authInfo := GetInfo(c)
	return authInfo != nil && authInfo.TeamID != "" && authInfo.UserID != ""
}

// GetConstraints extracts data access constraints from the gin context
// Returns a DataConstraints struct with all constraint flags
func GetConstraints(c *gin.Context) types.DataConstraints {
	constraints := types.DataConstraints{}

	// Built-in constraints
	if ownerOnly, ok := c.Get("__owner_only"); ok {
		if ownerOnlyBool, ok := ownerOnly.(bool); ok {
			constraints.OwnerOnly = ownerOnlyBool
		}
	}

	if creatorOnly, ok := c.Get("__creator_only"); ok {
		if creatorOnlyBool, ok := creatorOnly.(bool); ok {
			constraints.CreatorOnly = creatorOnlyBool
		}
	}

	if editorOnly, ok := c.Get("__editor_only"); ok {
		if editorOnlyBool, ok := editorOnly.(bool); ok {
			constraints.EditorOnly = editorOnlyBool
		}
	}

	if teamOnly, ok := c.Get("__team_only"); ok {
		if teamOnlyBool, ok := teamOnly.(bool); ok {
			constraints.TeamOnly = teamOnlyBool
		}
	}

	// Extra constraints
	if extraConstraints, ok := c.Get("__extra_constraints"); ok {
		if extra, ok := extraConstraints.(map[string]interface{}); ok {
			constraints.Extra = extra
		}
	}

	return constraints
}

// UpdateConstraints updates data access constraints in the gin context
// This should be called by ACL enforcement after successful permission check
// Accepts a map of constraints for flexible extension
func UpdateConstraints(c *gin.Context, constraints map[string]interface{}) {
	// Set each constraint in the context
	for key, value := range constraints {
		c.Set("__"+key, value)
	}
}

// SetInfo sets authorized information in the gin context
// This function should be called by the OAuth guard middleware after token validation
// userIDGetter is a function that resolves the user_id from clientID and subject
func SetInfo(c *gin.Context, claims *types.TokenClaims, sessionID string, userIDGetter func(clientID, subject string) (string, error)) {
	// Set session ID in context
	if sessionID != "" {
		c.Set("__sid", sessionID)
	}

	// Set user_id in context (resolve from claims)
	if userIDGetter != nil {
		userID, err := userIDGetter(claims.ClientID, claims.Subject)
		if err == nil && userID != "" {
			c.Set("__user_id", userID)
		}
	}

	// Set subject, scope, client_id in context
	c.Set("__subject", claims.Subject)
	c.Set("__scope", claims.Scope)
	c.Set("__client_id", claims.ClientID)

	// Set team_id and tenant_id in context if available
	if claims.TeamID != "" {
		c.Set("__team_id", claims.TeamID)
	}
	if claims.TenantID != "" {
		c.Set("__tenant_id", claims.TenantID)
	}

	// Set custom claims from Extra field into context
	if claims.Extra != nil {
		for key, value := range claims.Extra {
			c.Set("__"+key, value)
		}
	}
}
