package acl

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/acl/role"
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

	// Build access request
	request := &AccessRequest{
		Method: c.Request.Method,
		Path:   c.Request.URL.Path,
	}

	// Execute enforcement chain and collect endpoint info
	allowed, endpointInfo, err := acl.enforce(c.Request.Context(), authInfo, request)
	if err != nil {
		return false, err
	}

	if !allowed {
		return false, nil
	}

	// Update context with data access constraints from matched endpoint
	if endpointInfo != nil {
		constraints := endpointInfo.GetConstraints()
		authorized.UpdateConstraints(c, constraints)
	}

	return true, nil
}

// enforce is the main enforcement chain that orchestrates all permission checks
// Each step independently validates permissions against the endpoint
// ALL checks must pass (AND logic) - if any check fails, access is denied
// Returns: (allowed bool, endpointInfo *EndpointInfo, error)
func (acl *ACL) enforce(ctx context.Context, authInfo *types.AuthorizedInfo, request *AccessRequest) (bool, *EndpointInfo, error) {
	// Step 1: Check client permission - MUST pass
	allowed, matchedEndpoint, err := acl.enforceClient(ctx, authInfo, request)
	if err != nil {
		return false, nil, err
	}
	if !allowed {
		return false, nil, &Error{
			Type:    ErrorTypePermissionDenied,
			Message: "access denied: client permission check failed",
			Stage:   EnforcementStageClient,
		}
	}

	// Step 2: Check explicit scopes from token (if any) - MUST pass if present
	// If token scope is empty, skip this check
	if authInfo.Scope != "" {
		allowed, endpoint, err := acl.enforceScope(ctx, authInfo, request)
		if err != nil {
			return false, nil, err
		}
		if !allowed {
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied: token scope check failed",
				Stage:   EnforcementStageScope,
			}
		}
		// Collect endpoint info
		if matchedEndpoint == nil && endpoint != nil {
			matchedEndpoint = endpoint
		}
	}

	// Step 3: Check team or user permissions
	// 3.1: If TeamID is present, this is a team login
	if authInfo.TeamID != "" {
		// 3.1.1: Check team permissions - MUST pass
		allowed, endpoint, err := acl.enforceTeam(ctx, authInfo, request)
		if err != nil {
			return false, nil, err
		}
		if !allowed {
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied: team permission check failed",
				Stage:   EnforcementStageTeam,
			}
		}
		// Collect endpoint info
		if matchedEndpoint == nil && endpoint != nil {
			matchedEndpoint = endpoint
		}

		// 3.1.2: Check member permissions (user's role in the team) - MUST pass
		allowed, endpoint, err = acl.enforceMember(ctx, authInfo, request)
		if err != nil {
			return false, nil, err
		}
		if !allowed {
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied: member permission check failed",
				Stage:   EnforcementStageMember,
			}
		}
		// Collect endpoint info
		if matchedEndpoint == nil && endpoint != nil {
			matchedEndpoint = endpoint
		}

		// All checks passed for team login
		return true, matchedEndpoint, nil
	}

	// 3.2: This is a user login (no TeamID)
	if authInfo.UserID != "" {
		allowed, endpoint, err := acl.enforceUser(ctx, authInfo, request)
		if err != nil {
			return false, nil, err
		}
		if !allowed {
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied: user permission check failed",
				Stage:   EnforcementStageUser,
			}
		}
		// Collect endpoint info
		if matchedEndpoint == nil && endpoint != nil {
			matchedEndpoint = endpoint
		}

		// All checks passed for user login
		return true, matchedEndpoint, nil
	}

	// All checks passed (pure API call - only client check required)
	return true, matchedEndpoint, nil
}

// enforceClient checks client permissions independently
// Returns: (allowed bool, endpointInfo *EndpointInfo, error)
func (acl *ACL) enforceClient(ctx context.Context, authInfo *types.AuthorizedInfo, request *AccessRequest) (bool, *EndpointInfo, error) {
	// Get client role
	clientRole, err := role.RoleManager.GetClientRole(ctx, authInfo.ClientID)
	if err != nil {
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get client role: %v", err),
			Stage:   EnforcementStageClient,
		}
	}

	// Get scopes for client role
	allowedScopes, restrictedScopes, err := role.RoleManager.GetScopes(ctx, clientRole)
	if err != nil {
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get client scopes: %v", err),
			Stage:   EnforcementStageClient,
		}
	}

	// Step 1: Check if allowed scopes grant access
	allowedRequest := &AccessRequest{
		Method: request.Method,
		Path:   request.Path,
		Scopes: allowedScopes,
	}

	decision := acl.Scope.Check(allowedRequest)
	if !decision.Allowed {
		return false, nil, &Error{
			Type:    ErrorTypePermissionDenied,
			Message: decision.Reason,
			Stage:   EnforcementStageClient,
			Details: map[string]interface{}{
				"required_scopes": decision.RequiredScopes,
				"missing_scopes":  decision.MissingScopes,
			},
		}
	}

	// Step 2: Check if restricted scopes block access
	if len(restrictedScopes) > 0 {
		restrictedRequest := &AccessRequest{
			Method: request.Method,
			Path:   request.Path,
			Scopes: restrictedScopes,
		}

		restrictDecision := acl.Scope.CheckRestricted(restrictedRequest)
		if !restrictDecision.Allowed {
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied by restriction: " + restrictDecision.Reason,
				Stage:   EnforcementStageClient,
				Details: map[string]interface{}{
					"restricted_scopes": restrictedScopes,
					"matched_pattern":   restrictDecision.MatchedPattern,
				},
			}
		}
	}

	// Return matched endpoint info (contains OwnerOnly, TeamOnly, and future constraints)
	return true, decision.MatchedEndpoint, nil
}

// enforceScope checks the explicit scopes from token independently
// Returns: (allowed bool, endpointInfo *EndpointInfo, error)
func (acl *ACL) enforceScope(_ context.Context, authInfo *types.AuthorizedInfo, request *AccessRequest) (bool, *EndpointInfo, error) {
	// Parse scopes from token (space-separated)
	if authInfo.Scope == "" {
		return false, nil, nil
	}

	// Split space-separated scopes
	// e.g., "read:users write:users" -> ["read:users", "write:users"]
	tokenScopes := strings.Split(authInfo.Scope, " ")

	// Filter out empty strings
	var scopes []string
	for _, scope := range tokenScopes {
		if scope != "" {
			scopes = append(scopes, scope)
		}
	}

	// Build request with token scopes and check
	checkRequest := &AccessRequest{
		Method: request.Method,
		Path:   request.Path,
		Scopes: scopes,
	}

	decision := acl.Scope.Check(checkRequest)
	if !decision.Allowed {
		return false, nil, &Error{
			Type:    ErrorTypePermissionDenied,
			Message: decision.Reason,
			Stage:   EnforcementStageScope,
			Details: map[string]interface{}{
				"required_scopes": decision.RequiredScopes,
				"missing_scopes":  decision.MissingScopes,
			},
		}
	}

	// Return matched endpoint info
	return true, decision.MatchedEndpoint, nil
}

// enforceUser checks user permissions independently
// Returns: (allowed bool, endpointInfo *EndpointInfo, error)
func (acl *ACL) enforceUser(ctx context.Context, authInfo *types.AuthorizedInfo, request *AccessRequest) (bool, *EndpointInfo, error) {
	// Get user role
	userRole, err := role.RoleManager.GetUserRole(ctx, authInfo.UserID)
	if err != nil {
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get user role: %v", err),
			Stage:   EnforcementStageUser,
		}
	}

	// Get scopes for user role
	allowedScopes, restrictedScopes, err := role.RoleManager.GetScopes(ctx, userRole)
	if err != nil {
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get user scopes: %v", err),
			Stage:   EnforcementStageUser,
		}
	}

	// Step 1: Check if allowed scopes grant access
	allowedRequest := &AccessRequest{
		Method: request.Method,
		Path:   request.Path,
		Scopes: allowedScopes,
	}

	decision := acl.Scope.Check(allowedRequest)
	if !decision.Allowed {
		return false, nil, &Error{
			Type:    ErrorTypePermissionDenied,
			Message: decision.Reason,
			Stage:   EnforcementStageUser,
			Details: map[string]interface{}{
				"required_scopes": decision.RequiredScopes,
				"missing_scopes":  decision.MissingScopes,
			},
		}
	}

	// Step 2: Check if restricted scopes block access
	if len(restrictedScopes) > 0 {
		restrictedRequest := &AccessRequest{
			Method: request.Method,
			Path:   request.Path,
			Scopes: restrictedScopes,
		}

		restrictDecision := acl.Scope.CheckRestricted(restrictedRequest)
		if !restrictDecision.Allowed {
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied by restriction: " + restrictDecision.Reason,
				Stage:   EnforcementStageUser,
				Details: map[string]interface{}{
					"restricted_scopes": restrictedScopes,
					"matched_pattern":   restrictDecision.MatchedPattern,
				},
			}
		}
	}

	// Return matched endpoint info
	return true, decision.MatchedEndpoint, nil
}

// enforceTeam checks team permissions independently
// Returns: (allowed bool, endpointInfo *EndpointInfo, error)
func (acl *ACL) enforceTeam(ctx context.Context, authInfo *types.AuthorizedInfo, request *AccessRequest) (bool, *EndpointInfo, error) {
	// Get team role
	teamRole, err := role.RoleManager.GetTeamRole(ctx, authInfo.TeamID)
	if err != nil {
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get team role: %v", err),
			Stage:   EnforcementStageTeam,
		}
	}

	// Get scopes for team role
	allowedScopes, restrictedScopes, err := role.RoleManager.GetScopes(ctx, teamRole)
	if err != nil {
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get team scopes: %v", err),
			Stage:   EnforcementStageTeam,
		}
	}

	// Step 1: Check if allowed scopes grant access
	allowedRequest := &AccessRequest{
		Method: request.Method,
		Path:   request.Path,
		Scopes: allowedScopes,
	}

	decision := acl.Scope.Check(allowedRequest)
	if !decision.Allowed {
		return false, nil, &Error{
			Type:    ErrorTypePermissionDenied,
			Message: decision.Reason,
			Stage:   EnforcementStageTeam,
			Details: map[string]interface{}{
				"required_scopes": decision.RequiredScopes,
				"missing_scopes":  decision.MissingScopes,
			},
		}
	}

	// Step 2: Check if restricted scopes block access
	if len(restrictedScopes) > 0 {
		restrictedRequest := &AccessRequest{
			Method: request.Method,
			Path:   request.Path,
			Scopes: restrictedScopes,
		}

		restrictDecision := acl.Scope.CheckRestricted(restrictedRequest)
		if !restrictDecision.Allowed {
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied by restriction: " + restrictDecision.Reason,
				Stage:   EnforcementStageTeam,
				Details: map[string]interface{}{
					"restricted_scopes": restrictedScopes,
					"matched_pattern":   restrictDecision.MatchedPattern,
				},
			}
		}
	}

	// Return matched endpoint info
	return true, decision.MatchedEndpoint, nil
}

// enforceMember checks member permissions independently
// Returns: (allowed bool, endpointInfo *EndpointInfo, error)
func (acl *ACL) enforceMember(ctx context.Context, authInfo *types.AuthorizedInfo, request *AccessRequest) (bool, *EndpointInfo, error) {
	// Get member role (user's role in the team)
	memberRole, err := role.RoleManager.GetMemberRole(ctx, authInfo.TeamID, authInfo.UserID)
	if err != nil {
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get member role: %v", err),
			Stage:   EnforcementStageMember,
		}
	}

	// Get scopes for member role
	allowedScopes, restrictedScopes, err := role.RoleManager.GetScopes(ctx, memberRole)
	if err != nil {
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get member scopes: %v", err),
			Stage:   EnforcementStageMember,
		}
	}

	// Step 1: Check if allowed scopes grant access
	allowedRequest := &AccessRequest{
		Method: request.Method,
		Path:   request.Path,
		Scopes: allowedScopes,
	}

	decision := acl.Scope.Check(allowedRequest)
	if !decision.Allowed {
		return false, nil, &Error{
			Type:    ErrorTypePermissionDenied,
			Message: decision.Reason,
			Stage:   EnforcementStageMember,
			Details: map[string]interface{}{
				"required_scopes": decision.RequiredScopes,
				"missing_scopes":  decision.MissingScopes,
			},
		}
	}

	// Step 2: Check if restricted scopes block access
	if len(restrictedScopes) > 0 {
		restrictedRequest := &AccessRequest{
			Method: request.Method,
			Path:   request.Path,
			Scopes: restrictedScopes,
		}

		restrictDecision := acl.Scope.CheckRestricted(restrictedRequest)
		if !restrictDecision.Allowed {
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied by restriction: " + restrictDecision.Reason,
				Stage:   EnforcementStageMember,
				Details: map[string]interface{}{
					"restricted_scopes": restrictedScopes,
					"matched_pattern":   restrictDecision.MatchedPattern,
				},
			}
		}
	}

	// Return matched endpoint info
	return true, decision.MatchedEndpoint, nil
}
