package acl

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/acl/role"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Enforce checks if a user has access to a resource based on the request context
func (acl *ACL) Enforce(c *gin.Context) (bool, error) {
	// If ACL is not enabled, allow access
	if !acl.Enabled() {
		log.Trace("[ACL] ACL is disabled, allowing access")
		return true, nil
	}

	// If scope manager not loaded, deny access
	if acl.Scope == nil {
		log.Trace("[ACL] Scope manager not loaded, denying access")
		return false, nil
	}

	// Get authorized info from context (set by OAuth guard middleware)
	authInfo := authorized.GetInfo(c)

	// Get request path and strip PathPrefix if configured
	requestPath := c.Request.URL.Path
	if acl.Config.PathPrefix != "" && strings.HasPrefix(requestPath, acl.Config.PathPrefix) {
		requestPath = strings.TrimPrefix(requestPath, acl.Config.PathPrefix)
		log.Trace("[ACL] Stripped path prefix %s from request path, new path: %s", acl.Config.PathPrefix, requestPath)
	}

	// Build access request
	request := &AccessRequest{
		Method: c.Request.Method,
		Path:   requestPath,
	}

	log.Trace("[ACL] Starting enforcement chain: method=%s, path=%s (original=%s), client_id=%s, user_id=%s, team_id=%s, scope=%s",
		request.Method, request.Path, c.Request.URL.Path, authInfo.ClientID, authInfo.UserID, authInfo.TeamID, authInfo.Scope)

	// Execute enforcement chain and collect endpoint info
	allowed, endpointInfo, err := acl.enforce(c.Request.Context(), authInfo, request)
	if err != nil {
		log.Trace("[ACL] Enforcement failed with error: %v", err)
		return false, err
	}

	if !allowed {
		log.Trace("[ACL] Access denied by enforcement chain")
		return false, nil
	}

	// Update context with data access constraints from matched endpoint
	if endpointInfo != nil {
		constraints := endpointInfo.GetConstraints()
		authorized.UpdateConstraints(c, constraints)
		log.Trace("[ACL] Access granted, constraints applied: %+v", constraints)
	} else {
		log.Trace("[ACL] Access granted, no constraints")
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
		log.Trace("[ACL] Enforcement chain terminated: client permission check failed")
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
			log.Trace("[ACL] Enforcement chain terminated: token scope check failed")
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied: token scope check failed",
				Stage:   EnforcementStageScope,
			}
		}
		// Update endpoint info (later stages override earlier ones)
		if endpoint != nil {
			matchedEndpoint = endpoint
			log.Trace("[ACL] Step 2: Updated matched endpoint from token scope check")
		}
	} else {
		log.Trace("[ACL] Step 2: Token scope is empty, skipping scope check")
	}

	// Step 3: Check team or user permissions
	// 3.1: If TeamID is present, this is a team login
	if authInfo.TeamID != "" {
		log.Trace("[ACL] Detected team login (team_id=%s), checking team and member permissions", authInfo.TeamID)

		// 3.1.1: Check team permissions - MUST pass
		allowed, endpoint, err := acl.enforceTeam(ctx, authInfo, request)
		if err != nil {
			return false, nil, err
		}
		if !allowed {
			log.Trace("[ACL] Enforcement chain terminated: team permission check failed")
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied: team permission check failed",
				Stage:   EnforcementStageTeam,
			}
		}
		// Update endpoint info (later stages override earlier ones)
		if endpoint != nil {
			matchedEndpoint = endpoint
			log.Trace("[ACL] Step 3.1: Updated matched endpoint from team check")
		}

		// 3.1.2: Check member permissions (user's role in the team) - MUST pass
		// This is the final stage for team login, its constraints take precedence
		allowed, endpoint, err = acl.enforceMember(ctx, authInfo, request)
		if err != nil {
			return false, nil, err
		}
		if !allowed {
			log.Trace("[ACL] Enforcement chain terminated: member permission check failed")
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied: member permission check failed",
				Stage:   EnforcementStageMember,
			}
		}
		// Update endpoint info (FINAL stage for team login - takes precedence)
		if endpoint != nil {
			matchedEndpoint = endpoint
			log.Trace("[ACL] Step 3.1.2: Updated matched endpoint from member check (FINAL)")
		}

		// All checks passed for team login
		log.Trace("[ACL] Enforcement chain completed successfully: all team login checks passed")
		return true, matchedEndpoint, nil
	}

	// 3.2: This is a user login (no TeamID)
	if authInfo.UserID != "" {
		log.Trace("[ACL] Detected user login (user_id=%s), checking user permissions", authInfo.UserID)

		// This is the final stage for user login, its constraints take precedence
		allowed, endpoint, err := acl.enforceUser(ctx, authInfo, request)
		if err != nil {
			return false, nil, err
		}
		if !allowed {
			log.Trace("[ACL] Enforcement chain terminated: user permission check failed")
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied: user permission check failed",
				Stage:   EnforcementStageUser,
			}
		}
		// Update endpoint info (FINAL stage for user login - takes precedence)
		if endpoint != nil {
			matchedEndpoint = endpoint
			log.Trace("[ACL] Step 3.2: Updated matched endpoint from user check (FINAL)")
		}

		// All checks passed for user login
		log.Trace("[ACL] Enforcement chain completed successfully: all user login checks passed")
		return true, matchedEndpoint, nil
	}

	// All checks passed (pure API call - only client check required)
	log.Trace("[ACL] Enforcement chain completed successfully: pure API call (client only)")
	return true, matchedEndpoint, nil
}

// enforceClient checks client permissions independently
// Returns: (allowed bool, endpointInfo *EndpointInfo, error)
func (acl *ACL) enforceClient(ctx context.Context, authInfo *types.AuthorizedInfo, request *AccessRequest) (bool, *EndpointInfo, error) {
	log.Trace("[ACL] Step 1: enforceClient - Starting client permission check for client_id=%s", authInfo.ClientID)

	// Get client role
	clientRole, err := role.RoleManager.GetClientRole(ctx, authInfo.ClientID)
	if err != nil {
		log.Trace("[ACL] Step 1: enforceClient - Failed to get client role: %v", err)
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get client role [client_id=%s]: %v", authInfo.ClientID, err),
			Stage:   EnforcementStageClient,
		}
	}
	log.Trace("[ACL] Step 1: enforceClient - Retrieved client role: %s", clientRole)

	// Get scopes for client role
	allowedScopes, restrictedScopes, err := role.RoleManager.GetScopes(ctx, clientRole)
	if err != nil {
		log.Trace("[ACL] Step 1: enforceClient - Failed to get scopes for role %s: %v", clientRole, err)
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get client scopes [client_id=%s, role=%s]: %v", authInfo.ClientID, clientRole, err),
			Stage:   EnforcementStageClient,
		}
	}
	log.Trace("[ACL] Step 1: enforceClient - Retrieved scopes: allowed=%v, restricted=%v", allowedScopes, restrictedScopes)

	// Step 1: Check if allowed scopes grant access
	allowedRequest := &AccessRequest{
		Method: request.Method,
		Path:   request.Path,
		Scopes: allowedScopes,
	}

	decision := acl.Scope.Check(allowedRequest)
	log.Trace("[ACL] Step 1: enforceClient - Allowed scopes check: allowed=%v, reason=%s, required_scopes=%v, missing_scopes=%v, matched_pattern=%s",
		decision.Allowed, decision.Reason, decision.RequiredScopes, decision.MissingScopes, decision.MatchedPattern)

	if !decision.Allowed {
		log.Trace("[ACL] Step 1: enforceClient - Access denied by allowed scopes check")
		return false, nil, &Error{
			Type:    ErrorTypePermissionDenied,
			Message: decision.Reason,
			Stage:   EnforcementStageClient,
			Details: map[string]interface{}{
				"client_id":       authInfo.ClientID,
				"method":          request.Method,
				"path":            request.Path,
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
		log.Trace("[ACL] Step 1: enforceClient - Restricted scopes check: allowed=%v, reason=%s, matched_pattern=%s",
			restrictDecision.Allowed, restrictDecision.Reason, restrictDecision.MatchedPattern)

		if !restrictDecision.Allowed {
			log.Trace("[ACL] Step 1: enforceClient - Access denied by restricted scopes")
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied by restriction: " + restrictDecision.Reason,
				Stage:   EnforcementStageClient,
				Details: map[string]interface{}{
					"client_id":         authInfo.ClientID,
					"method":            request.Method,
					"path":              request.Path,
					"restricted_scopes": restrictedScopes,
					"matched_pattern":   restrictDecision.MatchedPattern,
				},
			}
		}
	}

	// Return matched endpoint info with scope-specific constraints
	var endpointInfo *EndpointInfo
	if decision.MatchedEndpoint != nil {
		if decision.MatchedScope != "" {
			// Get constraints for the specific matched scope
			endpointInfo = acl.Scope.GetScopeConstraints(decision.MatchedScope, request.Method, decision.MatchedPattern)
			log.Trace("[ACL] Step 1: enforceClient - Success, matched scope '%s': %+v", decision.MatchedScope, endpointInfo)
		} else {
			// No specific scope matched (e.g., public endpoint)
			endpointInfo = decision.MatchedEndpoint
			log.Trace("[ACL] Step 1: enforceClient - Success, matched endpoint: %+v", endpointInfo)
		}
	} else {
		log.Trace("[ACL] Step 1: enforceClient - Success, no endpoint info")
	}
	return true, endpointInfo, nil
}

// enforceScope checks the explicit scopes from token independently
// Returns: (allowed bool, endpointInfo *EndpointInfo, error)
func (acl *ACL) enforceScope(_ context.Context, authInfo *types.AuthorizedInfo, request *AccessRequest) (bool, *EndpointInfo, error) {
	log.Trace("[ACL] Step 2: enforceScope - Starting token scope check")

	// Parse scopes from token (space-separated)
	if authInfo.Scope == "" {
		log.Trace("[ACL] Step 2: enforceScope - Token scope is empty, skipping")
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
	log.Trace("[ACL] Step 2: enforceScope - Parsed token scopes: %v", scopes)

	// Build request with token scopes and check
	checkRequest := &AccessRequest{
		Method: request.Method,
		Path:   request.Path,
		Scopes: scopes,
	}

	decision := acl.Scope.Check(checkRequest)
	log.Trace("[ACL] Step 2: enforceScope - Token scope check: allowed=%v, reason=%s, required_scopes=%v, missing_scopes=%v, matched_pattern=%s",
		decision.Allowed, decision.Reason, decision.RequiredScopes, decision.MissingScopes, decision.MatchedPattern)

	if !decision.Allowed {
		log.Trace("[ACL] Step 2: enforceScope - Access denied by token scope")
		return false, nil, &Error{
			Type:    ErrorTypePermissionDenied,
			Message: decision.Reason,
			Stage:   EnforcementStageScope,
			Details: map[string]interface{}{
				"client_id":       authInfo.ClientID,
				"user_id":         authInfo.UserID,
				"method":          request.Method,
				"path":            request.Path,
				"required_scopes": decision.RequiredScopes,
				"missing_scopes":  decision.MissingScopes,
			},
		}
	}

	// Return matched endpoint info with scope-specific constraints
	var endpointInfo *EndpointInfo
	if decision.MatchedEndpoint != nil {
		if decision.MatchedScope != "" {
			// Get constraints for the specific matched scope
			endpointInfo = acl.Scope.GetScopeConstraints(decision.MatchedScope, request.Method, decision.MatchedPattern)
			log.Trace("[ACL] Step 2: enforceScope - Success, matched scope '%s': %+v", decision.MatchedScope, endpointInfo)
		} else {
			// No specific scope matched (e.g., public endpoint)
			endpointInfo = decision.MatchedEndpoint
			log.Trace("[ACL] Step 2: enforceScope - Success, matched endpoint: %+v", endpointInfo)
		}
	} else {
		log.Trace("[ACL] Step 2: enforceScope - Success, no endpoint info")
	}
	return true, endpointInfo, nil
}

// enforceUser checks user permissions independently
// Returns: (allowed bool, endpointInfo *EndpointInfo, error)
func (acl *ACL) enforceUser(ctx context.Context, authInfo *types.AuthorizedInfo, request *AccessRequest) (bool, *EndpointInfo, error) {
	log.Trace("[ACL] Step 3.2: enforceUser - Starting user permission check for user_id=%s", authInfo.UserID)

	// Get user role
	userRole, err := role.RoleManager.GetUserRole(ctx, authInfo.UserID)
	if err != nil {
		log.Trace("[ACL] Step 3.2: enforceUser - Failed to get user role: %v", err)
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get user role [user_id=%s]: %v", authInfo.UserID, err),
			Stage:   EnforcementStageUser,
		}
	}
	log.Trace("[ACL] Step 3.2: enforceUser - Retrieved user role: %s", userRole)

	// Get scopes for user role
	allowedScopes, restrictedScopes, err := role.RoleManager.GetScopes(ctx, userRole)
	if err != nil {
		log.Trace("[ACL] Step 3.2: enforceUser - Failed to get scopes for role %s: %v", userRole, err)
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get user scopes [user_id=%s, role=%s]: %v", authInfo.UserID, userRole, err),
			Stage:   EnforcementStageUser,
		}
	}
	log.Trace("[ACL] Step 3.2: enforceUser - Retrieved scopes: allowed=%v, restricted=%v", allowedScopes, restrictedScopes)

	// Step 1: Check if allowed scopes grant access
	allowedRequest := &AccessRequest{
		Method: request.Method,
		Path:   request.Path,
		Scopes: allowedScopes,
	}

	decision := acl.Scope.Check(allowedRequest)
	log.Trace("[ACL] Step 3.2: enforceUser - Allowed scopes check: allowed=%v, reason=%s, required_scopes=%v, missing_scopes=%v, matched_pattern=%s",
		decision.Allowed, decision.Reason, decision.RequiredScopes, decision.MissingScopes, decision.MatchedPattern)

	if !decision.Allowed {
		log.Trace("[ACL] Step 3.2: enforceUser - Access denied by allowed scopes check")
		return false, nil, &Error{
			Type:    ErrorTypePermissionDenied,
			Message: decision.Reason,
			Stage:   EnforcementStageUser,
			Details: map[string]interface{}{
				"user_id":         authInfo.UserID,
				"method":          request.Method,
				"path":            request.Path,
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
		log.Trace("[ACL] Step 3.2: enforceUser - Restricted scopes check: allowed=%v, reason=%s, matched_pattern=%s",
			restrictDecision.Allowed, restrictDecision.Reason, restrictDecision.MatchedPattern)

		if !restrictDecision.Allowed {
			log.Trace("[ACL] Step 3.2: enforceUser - Access denied by restricted scopes")
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied by restriction: " + restrictDecision.Reason,
				Stage:   EnforcementStageUser,
				Details: map[string]interface{}{
					"user_id":           authInfo.UserID,
					"method":            request.Method,
					"path":              request.Path,
					"restricted_scopes": restrictedScopes,
					"matched_pattern":   restrictDecision.MatchedPattern,
				},
			}
		}
	}

	// Return matched endpoint info with scope-specific constraints
	var endpointInfo *EndpointInfo
	if decision.MatchedEndpoint != nil {
		if decision.MatchedScope != "" {
			// Get constraints for the specific matched scope
			endpointInfo = acl.Scope.GetScopeConstraints(decision.MatchedScope, request.Method, decision.MatchedPattern)
			log.Trace("[ACL] Step 3.2: enforceUser - Success, matched scope '%s': %+v", decision.MatchedScope, endpointInfo)
		} else {
			// No specific scope matched (e.g., public endpoint)
			endpointInfo = decision.MatchedEndpoint
			log.Trace("[ACL] Step 3.2: enforceUser - Success, matched endpoint: %+v", endpointInfo)
		}
	} else {
		log.Trace("[ACL] Step 3.2: enforceUser - Success, no endpoint info")
	}
	return true, endpointInfo, nil
}

// enforceTeam checks team permissions independently
// Returns: (allowed bool, endpointInfo *EndpointInfo, error)
func (acl *ACL) enforceTeam(ctx context.Context, authInfo *types.AuthorizedInfo, request *AccessRequest) (bool, *EndpointInfo, error) {
	log.Trace("[ACL] Step 3.1: enforceTeam - Starting team permission check for team_id=%s", authInfo.TeamID)

	// Get team role
	teamRole, err := role.RoleManager.GetTeamRole(ctx, authInfo.TeamID)
	if err != nil {
		log.Trace("[ACL] Step 3.1: enforceTeam - Failed to get team role: %v", err)
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get team role [team_id=%s]: %v", authInfo.TeamID, err),
			Stage:   EnforcementStageTeam,
		}
	}
	log.Trace("[ACL] Step 3.1: enforceTeam - Retrieved team role: %s", teamRole)

	// Get scopes for team role
	allowedScopes, restrictedScopes, err := role.RoleManager.GetScopes(ctx, teamRole)
	if err != nil {
		log.Trace("[ACL] Step 3.1: enforceTeam - Failed to get scopes for role %s: %v", teamRole, err)
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get team scopes [team_id=%s, role=%s]: %v", authInfo.TeamID, teamRole, err),
			Stage:   EnforcementStageTeam,
		}
	}
	log.Trace("[ACL] Step 3.1: enforceTeam - Retrieved scopes: allowed=%v, restricted=%v", allowedScopes, restrictedScopes)

	// Step 1: Check if allowed scopes grant access
	allowedRequest := &AccessRequest{
		Method: request.Method,
		Path:   request.Path,
		Scopes: allowedScopes,
	}

	decision := acl.Scope.Check(allowedRequest)
	log.Trace("[ACL] Step 3.1: enforceTeam - Allowed scopes check: allowed=%v, reason=%s, required_scopes=%v, missing_scopes=%v, matched_pattern=%s",
		decision.Allowed, decision.Reason, decision.RequiredScopes, decision.MissingScopes, decision.MatchedPattern)

	if !decision.Allowed {
		log.Trace("[ACL] Step 3.1: enforceTeam - Access denied by allowed scopes check")
		return false, nil, &Error{
			Type:    ErrorTypePermissionDenied,
			Message: decision.Reason,
			Stage:   EnforcementStageTeam,
			Details: map[string]interface{}{
				"team_id":         authInfo.TeamID,
				"user_id":         authInfo.UserID,
				"method":          request.Method,
				"path":            request.Path,
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
		log.Trace("[ACL] Step 3.1: enforceTeam - Restricted scopes check: allowed=%v, reason=%s, matched_pattern=%s",
			restrictDecision.Allowed, restrictDecision.Reason, restrictDecision.MatchedPattern)

		if !restrictDecision.Allowed {
			log.Trace("[ACL] Step 3.1: enforceTeam - Access denied by restricted scopes")
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied by restriction: " + restrictDecision.Reason,
				Stage:   EnforcementStageTeam,
				Details: map[string]interface{}{
					"team_id":           authInfo.TeamID,
					"user_id":           authInfo.UserID,
					"method":            request.Method,
					"path":              request.Path,
					"restricted_scopes": restrictedScopes,
					"matched_pattern":   restrictDecision.MatchedPattern,
				},
			}
		}
	}

	// Return matched endpoint info with scope-specific constraints
	var endpointInfo *EndpointInfo
	if decision.MatchedEndpoint != nil {
		if decision.MatchedScope != "" {
			// Get constraints for the specific matched scope
			endpointInfo = acl.Scope.GetScopeConstraints(decision.MatchedScope, request.Method, decision.MatchedPattern)
			log.Trace("[ACL] Step 3.1: enforceTeam - Success, matched scope '%s': %+v", decision.MatchedScope, endpointInfo)
		} else {
			// No specific scope matched (e.g., public endpoint)
			endpointInfo = decision.MatchedEndpoint
			log.Trace("[ACL] Step 3.1: enforceTeam - Success, matched endpoint: %+v", endpointInfo)
		}
	} else {
		log.Trace("[ACL] Step 3.1: enforceTeam - Success, no endpoint info")
	}
	return true, endpointInfo, nil
}

// enforceMember checks member permissions independently
// Returns: (allowed bool, endpointInfo *EndpointInfo, error)
func (acl *ACL) enforceMember(ctx context.Context, authInfo *types.AuthorizedInfo, request *AccessRequest) (bool, *EndpointInfo, error) {
	log.Trace("[ACL] Step 3.1.2: enforceMember - Starting member permission check for team_id=%s, user_id=%s", authInfo.TeamID, authInfo.UserID)

	// Get member role (user's role in the team)
	memberRole, err := role.RoleManager.GetMemberRole(ctx, authInfo.TeamID, authInfo.UserID)
	if err != nil {
		log.Trace("[ACL] Step 3.1.2: enforceMember - Failed to get member role: %v", err)
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get member role [team_id=%s, user_id=%s]: %v", authInfo.TeamID, authInfo.UserID, err),
			Stage:   EnforcementStageMember,
		}
	}
	log.Trace("[ACL] Step 3.1.2: enforceMember - Retrieved member role: %s", memberRole)

	// Get scopes for member role
	allowedScopes, restrictedScopes, err := role.RoleManager.GetScopes(ctx, memberRole)
	if err != nil {
		log.Trace("[ACL] Step 3.1.2: enforceMember - Failed to get scopes for role %s: %v", memberRole, err)
		return false, nil, &Error{
			Type:    ErrorTypeInternal,
			Message: fmt.Sprintf("failed to get member scopes [team_id=%s, user_id=%s, role=%s]: %v", authInfo.TeamID, authInfo.UserID, memberRole, err),
			Stage:   EnforcementStageMember,
		}
	}
	log.Trace("[ACL] Step 3.1.2: enforceMember - Retrieved scopes: allowed=%v, restricted=%v", allowedScopes, restrictedScopes)

	// Step 1: Check if allowed scopes grant access
	allowedRequest := &AccessRequest{
		Method: request.Method,
		Path:   request.Path,
		Scopes: allowedScopes,
	}

	decision := acl.Scope.Check(allowedRequest)
	log.Trace("[ACL] Step 3.1.2: enforceMember - Allowed scopes check: allowed=%v, reason=%s, required_scopes=%v, missing_scopes=%v, matched_pattern=%s",
		decision.Allowed, decision.Reason, decision.RequiredScopes, decision.MissingScopes, decision.MatchedPattern)

	if !decision.Allowed {
		log.Trace("[ACL] Step 3.1.2: enforceMember - Access denied by allowed scopes check")
		return false, nil, &Error{
			Type:    ErrorTypePermissionDenied,
			Message: decision.Reason,
			Stage:   EnforcementStageMember,
			Details: map[string]interface{}{
				"team_id":         authInfo.TeamID,
				"user_id":         authInfo.UserID,
				"method":          request.Method,
				"path":            request.Path,
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
		log.Trace("[ACL] Step 3.1.2: enforceMember - Restricted scopes check: allowed=%v, reason=%s, matched_pattern=%s",
			restrictDecision.Allowed, restrictDecision.Reason, restrictDecision.MatchedPattern)

		if !restrictDecision.Allowed {
			log.Trace("[ACL] Step 3.1.2: enforceMember - Access denied by restricted scopes")
			return false, nil, &Error{
				Type:    ErrorTypePermissionDenied,
				Message: "access denied by restriction: " + restrictDecision.Reason,
				Stage:   EnforcementStageMember,
				Details: map[string]interface{}{
					"team_id":           authInfo.TeamID,
					"user_id":           authInfo.UserID,
					"method":            request.Method,
					"path":              request.Path,
					"restricted_scopes": restrictedScopes,
					"matched_pattern":   restrictDecision.MatchedPattern,
				},
			}
		}
	}

	// Return matched endpoint info with scope-specific constraints
	var endpointInfo *EndpointInfo
	if decision.MatchedEndpoint != nil {
		if decision.MatchedScope != "" {
			// Get constraints for the specific matched scope
			endpointInfo = acl.Scope.GetScopeConstraints(decision.MatchedScope, request.Method, decision.MatchedPattern)
			log.Trace("[ACL] Step 3.1.2: enforceMember - Success, matched scope '%s': %+v", decision.MatchedScope, endpointInfo)
		} else {
			// No specific scope matched (e.g., public endpoint)
			endpointInfo = decision.MatchedEndpoint
			log.Trace("[ACL] Step 3.1.2: enforceMember - Success, matched endpoint: %+v", endpointInfo)
		}
	} else {
		log.Trace("[ACL] Step 3.1.2: enforceMember - Success, no endpoint info")
	}
	return true, endpointInfo, nil
}
