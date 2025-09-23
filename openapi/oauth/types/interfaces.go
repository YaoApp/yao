package types

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// OAuth interface defines the complete OAuth 2.1 and MCP authorization server functionality
type OAuth interface {
	// AuthorizationServer returns the authorization server endpoint URL
	// This endpoint is used to initiate the authorization flow
	AuthorizationServer(ctx context.Context) string

	// ProtectedResource returns the protected resource endpoint URL
	// This endpoint is used to access protected resources using access tokens
	ProtectedResource(ctx context.Context) string

	// Authorize processes an authorization request and returns an authorization code
	// The authorization code can be exchanged for an access token
	Authorize(ctx context.Context, request *AuthorizationRequest) (*AuthorizationResponse, error)

	// Token exchanges an authorization code for an access token
	// This is the core token endpoint functionality
	Token(ctx context.Context, grantType string, code string, clientID string, codeVerifier string) (*Token, error)

	// Revoke revokes an access token or refresh token
	// Once revoked, the token cannot be used for accessing protected resources
	Revoke(ctx context.Context, token string, tokenTypeHint string) error

	// Introspect returns information about an access token
	// This endpoint allows resource servers to validate tokens
	Introspect(ctx context.Context, token string) (*TokenIntrospectionResponse, error)

	// Register registers a new OAuth client with the authorization server
	// This is used for static client registration
	Register(ctx context.Context, clientInfo *ClientInfo) (*ClientInfo, error)

	// JWKS returns the JSON Web Key Set for token verification
	// This endpoint provides public keys for validating JWT tokens
	JWKS(ctx context.Context) (*JWKSResponse, error)

	// Endpoints returns a map of all available OAuth endpoints
	// This provides endpoint discovery for clients
	Endpoints(ctx context.Context) (map[string]string, error)

	// RefreshToken exchanges a refresh token for a new access token
	// This allows clients to obtain fresh access tokens without user interaction
	// scope is optional - if provided, validates against originally granted scopes
	RefreshToken(ctx context.Context, refreshToken string, scope ...string) (*RefreshTokenResponse, error)

	// DeviceAuthorization initiates the device authorization flow
	// This is used for devices with limited input capabilities
	DeviceAuthorization(ctx context.Context, clientID string, scope string) (*DeviceAuthorizationResponse, error)

	// UserInfo returns user information for a given access token
	// This endpoint provides user profile information in the format defined by the UserProvider
	UserInfo(ctx context.Context, accessToken string) (interface{}, error)

	// GenerateCodeChallenge generates a code challenge from a code verifier
	// This is used for PKCE (Proof Key for Code Exchange) flow
	GenerateCodeChallenge(ctx context.Context, codeVerifier string, method string) (string, error)

	// ValidateCodeChallenge validates a code verifier against a code challenge
	// This verifies the PKCE code challenge during token exchange
	ValidateCodeChallenge(ctx context.Context, codeVerifier string, codeChallenge string, method string) error

	// PushAuthorizationRequest processes a pushed authorization request
	// This implements RFC 9126 for enhanced security
	PushAuthorizationRequest(ctx context.Context, request *PushedAuthorizationRequest) (*PushedAuthorizationResponse, error)

	// TokenExchange exchanges one token for another token
	// This implements RFC 8693 for token exchange scenarios
	TokenExchange(ctx context.Context, subjectToken string, subjectTokenType string, audience string, scope string) (*TokenExchangeResponse, error)

	// UpdateClient updates an existing OAuth client configuration
	// This allows modification of client metadata
	UpdateClient(ctx context.Context, clientID string, clientInfo *ClientInfo) (*ClientInfo, error)

	// DeleteClient removes an OAuth client from the authorization server
	// This permanently deletes the client and invalidates all associated tokens
	DeleteClient(ctx context.Context, clientID string) error

	// ValidateScope validates requested scopes against available scopes
	// This ensures clients only request permitted scopes
	ValidateScope(ctx context.Context, requestedScopes []string, clientID string) (*ValidationResult, error)

	// GetServerMetadata returns OAuth 2.0 Authorization Server Metadata
	// This implements RFC 8414 for server discovery
	GetServerMetadata(ctx context.Context) (*AuthorizationServerMetadata, error)

	// MCP Requirements

	// ValidateResourceParameter validates an OAuth 2.0 resource parameter
	// This ensures the resource parameter is valid and properly formatted
	ValidateResourceParameter(ctx context.Context, resource string) (*ValidationResult, error)

	// GetCanonicalResourceURI returns the canonical form of a resource URI
	// This normalizes resource URIs for consistent processing
	GetCanonicalResourceURI(ctx context.Context, serverURI string) (string, error)

	// GetProtectedResourceMetadata returns OAuth 2.0 Protected Resource Metadata
	// This implements RFC 9728 for MCP server discovery
	GetProtectedResourceMetadata(ctx context.Context) (*ProtectedResourceMetadata, error)

	// HandleWWWAuthenticate processes WWW-Authenticate challenges
	// This handles authentication challenges from protected resources
	HandleWWWAuthenticate(ctx context.Context, challenge string) (*WWWAuthenticateChallenge, error)

	// DynamicClientRegistration handles dynamic client registration
	// This implements RFC 7591 for automatic client registration
	DynamicClientRegistration(ctx context.Context, request *DynamicClientRegistrationRequest) (*DynamicClientRegistrationResponse, error)

	// ValidateStateParameter validates OAuth state parameters
	// This prevents CSRF attacks by verifying state parameters
	ValidateStateParameter(ctx context.Context, state string, clientID string) (*ValidationResult, error)

	// GenerateStateParameter generates a secure state parameter
	// This creates cryptographically secure state values for CSRF protection
	GenerateStateParameter(ctx context.Context, clientID string) (*StateParameter, error)

	// ValidateTokenAudience validates token audience claims
	// This ensures tokens are only used with their intended audiences
	ValidateTokenAudience(ctx context.Context, token string, expectedAudience string) (*ValidationResult, error)

	// MCP Security Requirements

	// ValidateRedirectURI validates redirect URIs against registered URIs
	// This prevents open redirect attacks by enforcing exact URI matching
	ValidateRedirectURI(ctx context.Context, redirectURI string, registeredURIs []string) (*ValidationResult, error)

	// RotateRefreshToken rotates a refresh token and invalidates the old one
	// This implements refresh token rotation for enhanced security
	// requestedScope is optional - if provided, validates against originally granted scopes
	RotateRefreshToken(ctx context.Context, oldToken string, requestedScope ...string) (*RefreshTokenResponse, error)

	// ValidateTokenBinding validates token binding information
	// This ensures tokens are bound to the correct client or device
	ValidateTokenBinding(ctx context.Context, token string, binding *TokenBinding) (*ValidationResult, error)

	// Guard is the OAuth guard middleware
	Guard(c *gin.Context)
}

// UserProvider interface for user information retrieval and management
type UserProvider interface {
	// ============================================================================
	// User Resource
	// ============================================================================

	// User Basic Operations
	GetUser(ctx context.Context, userID string) (maps.MapStrAny, error)
	GetUserWithScopes(ctx context.Context, userID string) (maps.MapStrAny, error)
	UserExists(ctx context.Context, userID string) (bool, error)
	UserExistsByEmail(ctx context.Context, email string) (bool, error)
	UserExistsByPreferredUsername(ctx context.Context, preferredUsername string) (bool, error)

	GetUserByPreferredUsername(ctx context.Context, preferredUsername string) (maps.MapStrAny, error)
	GetUserByEmail(ctx context.Context, email string) (maps.MapStrAny, error)
	GetUserForAuth(ctx context.Context, identifier string, identifierType string) (maps.MapStrAny, error)
	VerifyPassword(ctx context.Context, password string, passwordHash string) (bool, error)
	UpdatePassword(ctx context.Context, userID string, newPassword string) error
	ResetPassword(ctx context.Context, userID string) (string, error)

	CreateUser(ctx context.Context, userData maps.MapStrAny) (string, error)
	UpdateUser(ctx context.Context, userID string, userData maps.MapStrAny) error
	DeleteUser(ctx context.Context, userID string) error
	UpdateUserLastLogin(ctx context.Context, userID string, ip string) error
	UpdateUserStatus(ctx context.Context, userID string, status string) error

	// User List and Search
	GetUsers(ctx context.Context, param model.QueryParam) ([]maps.MapStr, error)
	PaginateUsers(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error)
	CountUsers(ctx context.Context, param model.QueryParam) (int64, error)

	// User Role and Type Management
	GetUserRole(ctx context.Context, userID string) (maps.MapStrAny, error)
	SetUserRole(ctx context.Context, userID string, roleID string) error
	ClearUserRole(ctx context.Context, userID string) error
	UserHasRole(ctx context.Context, userID string) (bool, error)
	GetUserType(ctx context.Context, userID string) (maps.MapStrAny, error)
	SetUserType(ctx context.Context, userID string, typeID string) error
	ClearUserType(ctx context.Context, userID string) error
	UserHasType(ctx context.Context, userID string) (bool, error)
	ValidateUserScope(ctx context.Context, userID string, scopes []string) (bool, error)

	// User MFA Management
	GenerateMFASecret(ctx context.Context, userID string, options *MFAOptions) (string, string, error)
	EnableMFA(ctx context.Context, userID string, secret string, code string) error
	DisableMFA(ctx context.Context, userID string, code string) error
	VerifyMFACode(ctx context.Context, userID string, code string) (bool, error)
	GenerateRecoveryCodes(ctx context.Context, userID string) ([]string, error)
	VerifyRecoveryCode(ctx context.Context, userID string, code string) (bool, error)
	IsMFAEnabled(ctx context.Context, userID string) (bool, error)
	GetMFAConfig(ctx context.Context, userID string) (maps.MapStrAny, error)

	// ============================================================================
	// OAuth Account Resource
	// ============================================================================

	CreateOAuthAccount(ctx context.Context, userID string, oauthData maps.MapStrAny) (interface{}, error)
	GetOAuthAccount(ctx context.Context, provider string, subject string) (maps.MapStrAny, error)
	OAuthAccountExists(ctx context.Context, provider string, subject string) (bool, error)
	GetUserOAuthAccounts(ctx context.Context, userID string) ([]maps.MapStrAny, error)
	UpdateOAuthAccount(ctx context.Context, provider string, subject string, oauthData maps.MapStrAny) error
	DeleteOAuthAccount(ctx context.Context, provider string, subject string) error
	DeleteUserOAuthAccounts(ctx context.Context, userID string) error

	GetOAuthAccounts(ctx context.Context, param model.QueryParam) ([]maps.MapStr, error)
	PaginateOAuthAccounts(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error)
	CountOAuthAccounts(ctx context.Context, param model.QueryParam) (int64, error)

	// ============================================================================
	// Role Resource
	// ============================================================================

	GetRole(ctx context.Context, roleID string) (maps.MapStrAny, error)
	RoleExists(ctx context.Context, roleID string) (bool, error)
	CreateRole(ctx context.Context, roleData maps.MapStrAny) (string, error)
	UpdateRole(ctx context.Context, roleID string, roleData maps.MapStrAny) error
	DeleteRole(ctx context.Context, roleID string) error

	GetRoles(ctx context.Context, param model.QueryParam) ([]maps.MapStr, error)
	PaginateRoles(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error)
	CountRoles(ctx context.Context, param model.QueryParam) (int64, error)

	GetRolePermissions(ctx context.Context, roleID string) (maps.MapStrAny, error)
	SetRolePermissions(ctx context.Context, roleID string, permissions maps.MapStrAny) error
	ValidateRolePermissions(ctx context.Context, roleID string, requiredPermissions []string) (bool, error)

	// ============================================================================
	// Type Resource
	// ============================================================================

	GetType(ctx context.Context, typeID string) (maps.MapStrAny, error)
	TypeExists(ctx context.Context, typeID string) (bool, error)
	CreateType(ctx context.Context, typeData maps.MapStrAny) (string, error)
	UpdateType(ctx context.Context, typeID string, typeData maps.MapStrAny) error
	DeleteType(ctx context.Context, typeID string) error

	GetTypes(ctx context.Context, param model.QueryParam) ([]maps.MapStr, error)
	PaginateTypes(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error)
	CountTypes(ctx context.Context, param model.QueryParam) (int64, error)

	GetTypeConfiguration(ctx context.Context, typeID string) (maps.MapStrAny, error)
	SetTypeConfiguration(ctx context.Context, typeID string, config maps.MapStrAny) error

	// ============================================================================
	// Team Resource
	// ============================================================================

	// Team Basic Operations
	GetTeam(ctx context.Context, teamID string) (maps.MapStrAny, error)
	GetTeamDetail(ctx context.Context, teamID string) (maps.MapStrAny, error)
	TeamExists(ctx context.Context, teamID string) (bool, error)
	CreateTeam(ctx context.Context, teamData maps.MapStrAny) (string, error)
	UpdateTeam(ctx context.Context, teamID string, teamData maps.MapStrAny) error
	DeleteTeam(ctx context.Context, teamID string) error

	// Team List and Search
	GetTeams(ctx context.Context, param model.QueryParam) ([]maps.MapStr, error)
	PaginateTeams(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error)
	CountTeams(ctx context.Context, param model.QueryParam) (int64, error)

	// Team Query Methods
	GetTeamsByOwner(ctx context.Context, ownerID string) ([]maps.MapStr, error)
	GetTeamsByStatus(ctx context.Context, status string) ([]maps.MapStr, error)

	// Team Management
	UpdateTeamStatus(ctx context.Context, teamID string, status string) error
	VerifyTeam(ctx context.Context, teamID string, verifiedBy string) error
	UnverifyTeam(ctx context.Context, teamID string) error
	TransferTeamOwnership(ctx context.Context, teamID string, newOwnerID string) error

	// Team Permission Checks
	IsTeamOwner(ctx context.Context, teamID string, userID string) (bool, error)
	IsTeamMember(ctx context.Context, teamID string, userID string) (bool, error)
	CheckTeamAccess(ctx context.Context, teamID string, userID string) (isOwner bool, isMember bool, err error)

	// ============================================================================
	// Member Resource
	// ============================================================================

	// Member Basic Operations
	GetMember(ctx context.Context, teamID string, userID string) (maps.MapStrAny, error)
	GetMemberDetail(ctx context.Context, teamID string, userID string) (maps.MapStrAny, error)
	GetMemberByID(ctx context.Context, memberID int64) (maps.MapStrAny, error)
	GetMemberByInvitationID(ctx context.Context, invitationID string) (maps.MapStrAny, error)
	MemberExists(ctx context.Context, teamID string, userID string) (bool, error)
	CreateMember(ctx context.Context, memberData maps.MapStrAny) (int64, error)
	UpdateMember(ctx context.Context, teamID string, userID string, memberData maps.MapStrAny) error
	UpdateMemberByID(ctx context.Context, memberID int64, memberData maps.MapStrAny) error
	UpdateMemberByInvitationID(ctx context.Context, invitationID string, memberData maps.MapStrAny) error
	RemoveMember(ctx context.Context, teamID string, userID string) error
	RemoveMemberByInvitationID(ctx context.Context, invitationID string) error
	RemoveAllTeamMembers(ctx context.Context, teamID string) error

	// Member Invitation Management
	AddMember(ctx context.Context, teamID string, userID string, roleID string, invitedBy string) (int64, error)
	AcceptInvitation(ctx context.Context, invitationToken string) error

	// Robot Member Operations
	CreateRobotMember(ctx context.Context, teamID string, robotData maps.MapStrAny) (int64, error)
	UpdateRobotActivity(ctx context.Context, memberID int64, robotStatus string) error
	GetActiveRobotMembers(ctx context.Context) ([]maps.MapStr, error)

	// Member Query Methods
	GetTeamMembers(ctx context.Context, teamID string) ([]maps.MapStr, error)
	GetUserTeams(ctx context.Context, userID string) ([]maps.MapStr, error)
	GetTeamMembersByStatus(ctx context.Context, teamID string, status string) ([]maps.MapStr, error)
	GetTeamRobotMembers(ctx context.Context, teamID string) ([]maps.MapStr, error)

	// Member Management
	UpdateMemberRole(ctx context.Context, teamID string, userID string, roleID string) error
	UpdateMemberStatus(ctx context.Context, teamID string, userID string, status string) error
	UpdateMemberLastActivity(ctx context.Context, teamID string, userID string) error

	// Member List and Search
	PaginateMembers(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error)

	// ============================================================================
	// Utils
	// ============================================================================

	// GenerateUserID generates a new unique user_id for user creation
	GenerateUserID(ctx context.Context, safe ...bool) (string, error)

	// GetOAuthUserID quickly retrieves user_id by OAuth provider and subject
	GetOAuthUserID(ctx context.Context, provider string, subject string) (string, error)
}

// ClientProvider interface for OAuth client management and persistence
type ClientProvider interface {
	// GetClientByID retrieves client information using a client ID
	GetClientByID(ctx context.Context, clientID string) (*ClientInfo, error)

	// GetClientByCredentials retrieves and validates client using client credentials
	// Used for client authentication in token requests
	GetClientByCredentials(ctx context.Context, clientID string, clientSecret string) (*ClientInfo, error)

	// CreateClient creates a new OAuth client and returns the client information
	CreateClient(ctx context.Context, clientInfo *ClientInfo) (*ClientInfo, error)

	// UpdateClient updates an existing OAuth client configuration
	UpdateClient(ctx context.Context, clientID string, clientInfo *ClientInfo) (*ClientInfo, error)

	// DeleteClient removes an OAuth client from the system
	// This should also invalidate all associated tokens
	DeleteClient(ctx context.Context, clientID string) error

	// ValidateClient validates client information and configuration
	// Returns validation result with any errors or warnings
	ValidateClient(ctx context.Context, clientInfo *ClientInfo) (*ValidationResult, error)

	// ListClients retrieves a list of clients with optional filtering
	// Supports pagination and filtering by various criteria
	ListClients(ctx context.Context, filters map[string]interface{}, limit int, offset int) ([]*ClientInfo, int, error)

	// ValidateRedirectURI validates if a redirect URI is registered for the client
	ValidateRedirectURI(ctx context.Context, clientID string, redirectURI string) (*ValidationResult, error)

	// ValidateScope validates if the client is authorized to request specific scopes
	ValidateScope(ctx context.Context, clientID string, scopes []string) (*ValidationResult, error)

	// IsClientActive checks if a client is active and can be used for authentication
	IsClientActive(ctx context.Context, clientID string) (bool, error)
}
