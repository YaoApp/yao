package user

import (
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Error messages
const (
	ErrUserNotFound             = "user not found"
	ErrRoleNotFound             = "role not found"
	ErrTypeNotFound             = "type not found"
	ErrOAuthAccountNotFound     = "oauth account not found"
	ErrInvalidIdentifierType    = "invalid identifier type: %s"
	ErrNoPasswordHash           = "no password hash found"
	ErrFailedToGenerateUserID   = "failed to generate user_id: %w"
	ErrFailedToGeneratePassword = "failed to generate password: %w"
	ErrInvalidUserIDInOAuth     = "invalid user_id in oauth account"

	ErrFailedToGetUser         = "failed to get user: %w"
	ErrFailedToGetRole         = "failed to get role: %w"
	ErrFailedToGetType         = "failed to get type: %w"
	ErrFailedToGetOAuthAccount = "failed to get oauth account: %w"
	ErrFailedToCreateUser      = "failed to create user: %w"
	ErrFailedToCreateRole      = "failed to create role: %w"
	ErrFailedToCreateType      = "failed to create type: %w"
	ErrFailedToCreateOAuth     = "failed to create oauth account: %w"
	ErrFailedToUpdateUser      = "failed to update user: %w"
	ErrFailedToUpdateRole      = "failed to update role: %w"
	ErrFailedToUpdateType      = "failed to update type: %w"
	ErrFailedToUpdateOAuth     = "failed to update oauth account: %w"
	ErrFailedToDeleteUser      = "failed to delete user: %w"
	ErrFailedToDeleteRole      = "failed to delete role: %w"
	ErrFailedToDeleteType      = "failed to delete type: %w"
	ErrFailedToDeleteOAuth     = "failed to delete oauth account: %w"

	// MFA related errors
	ErrMFANotEnabled             = "MFA is not enabled for this user"
	ErrMFAAlreadyEnabled         = "MFA is already enabled for this user"
	ErrInvalidMFACode            = "invalid MFA code"
	ErrInvalidRecoveryCode       = "invalid recovery code"
	ErrFailedToGenerateMFASecret = "failed to generate MFA secret: %w"
	ErrFailedToGenerateQRCode    = "failed to generate QR code: %w"
	ErrFailedToVerifyMFACode     = "failed to verify MFA code: %w"
	ErrFailedToUpdateMFAStatus   = "failed to update MFA status: %w"
	ErrRecoveryCodeNotFound      = "recovery code not found or already used"
)

// Default field lists - used when not configured
var (
	// DefaultPublicUserFields contains fields that can be safely returned to users
	DefaultPublicUserFields = []interface{}{
		"id", "user_id", "preferred_username", "email", "email_verified", "name", "given_name", "family_name",
		"middle_name", "nickname", "profile", "picture", "website", "gender", "birthdate", "zoneinfo", "locale",
		"phone_number", "phone_number_verified", "address", "theme", "status", "role_id", "type_id",
		"mfa_enabled", "last_login_at", "metadata", "created_at", "updated_at",
	}

	// DefaultBasicUserFields contains minimal fields for basic user info
	DefaultBasicUserFields = []interface{}{
		"id", "user_id", "preferred_username", "email", "email_verified", "name", "given_name", "family_name",
		"picture", "status", "role_id", "type_id",
	}

	// DefaultAuthUserFields contains fields needed for authentication
	DefaultAuthUserFields = []interface{}{
		"id", "user_id", "preferred_username", "email", "password_hash", "status", "role_id", "type_id",
		"email_verified", "phone_number_verified", "mfa_enabled", "last_login_at",
	}

	// DefaultMFAUserFields contains fields needed for MFA authentication
	DefaultMFAUserFields = []interface{}{
		"id", "user_id", "mfa_enabled", "mfa_secret", "mfa_issuer", "mfa_algorithm",
		"mfa_digits", "mfa_period", "mfa_recovery_hash", "mfa_enabled_at",
	}

	// DefaultOAuthAccountFields contains basic OAuth account fields
	DefaultOAuthAccountFields = []interface{}{
		"id", "user_id", "provider", "sub", "preferred_username", "email", "email_verified",
		"name", "given_name", "family_name", "picture", "last_login_at", "is_active",
		"created_at", "updated_at",
	}

	// DefaultOAuthAccountDetailFields contains all OAuth account fields including OIDC claims
	DefaultOAuthAccountDetailFields = []interface{}{
		"id", "user_id", "provider", "sub", "preferred_username", "email", "email_verified",
		"name", "given_name", "family_name", "middle_name", "nickname", "profile", "picture",
		"website", "gender", "birthdate", "zoneinfo", "locale", "phone_number", "phone_number_verified",
		"address", "raw", "last_login_at", "is_active", "created_at", "updated_at",
	}

	// DefaultRoleFields contains basic role fields
	DefaultRoleFields = []interface{}{
		"id", "role_id", "name", "description", "is_active", "is_default", "is_system",
		"level", "sort_order", "color", "icon", "created_at", "updated_at",
	}

	// DefaultRoleDetailFields contains all role fields including permissions and metadata
	DefaultRoleDetailFields = []interface{}{
		"id", "role_id", "name", "description", "permissions", "restricted_permissions",
		"parent_role_id", "level", "is_active", "is_default", "is_system", "sort_order",
		"color", "icon", "max_users", "requires_approval", "auto_revoke_days",
		"metadata", "conditions", "created_at", "updated_at",
	}

	// DefaultTypeFields contains basic type fields
	DefaultTypeFields = []interface{}{
		"id", "type_id", "name", "description", "is_active", "is_default", "sort_order",
		"default_role_id", "max_sessions", "session_timeout", "created_at", "updated_at",
	}

	// DefaultTypeDetailFields contains all type fields including configuration and metadata
	DefaultTypeDetailFields = []interface{}{
		"id", "type_id", "name", "description", "default_role_id", "schema", "metadata",
		"is_active", "is_default", "sort_order", "max_sessions", "session_timeout",
		"password_policy", "features", "limits", "created_at", "updated_at",
	}

	// DefaultMFAOptions contains default MFA configuration
	DefaultMFAOptions = &types.MFAOptions{
		Issuer:         "Yao App Engine",
		Algorithm:      "SHA256",
		Digits:         6,
		Period:         30,
		SecretSize:     32,
		RecoveryCount:  16, // 16 codes (~960 bytes, under 1024 char limit)
		RecoveryLength: 12, // 12-character codes for better security
	}
)

// DefaultUser provides a default implementation of UserProvider
type DefaultUser struct {
	prefix            string
	model             string
	roleModel         string
	typeModel         string
	oauthAccountModel string
	cache             store.Store

	// ID Generation Configuration
	idStrategy IDStrategy
	idPrefix   string

	// Field lists
	publicUserFields []interface{} // configurable
	basicUserFields  []interface{} // configurable
	authUserFields   []interface{} // fixed for security
	mfaUserFields    []interface{} // fixed for security

	// OAuth Account Field lists
	oauthAccountFields       []interface{} // configurable
	oauthAccountDetailFields []interface{} // configurable

	// Role Field lists
	roleFields       []interface{} // configurable
	roleDetailFields []interface{} // configurable

	// Type Field lists
	typeFields       []interface{} // configurable
	typeDetailFields []interface{} // configurable

	// MFA Configuration
	mfaOptions *types.MFAOptions // configurable MFA settings
}

// IDStrategy defines the strategy for generating user IDs
type IDStrategy string

// Available ID generation strategies
const (
	NanoIDStrategy  IDStrategy = "nanoid"  // Short, URL-safe, readable (e.g., "Kx9mP2aQ7nR3")
	UUIDStrategy    IDStrategy = "uuid"    // Traditional UUID (for compatibility)
	NumericStrategy IDStrategy = "numeric" // Numeric ID (for compatibility)
)

// DefaultUserOptions provides options for the DefaultUser
type DefaultUserOptions struct {
	Prefix            string
	Model             string // bind to a specific user model
	RoleModel         string // bind to a specific role model
	TypeModel         string // bind to a specific type model
	OAuthAccountModel string // bind to a specific oauth account model
	Cache             store.Store

	// ID Generation Strategy
	IDStrategy IDStrategy // strategy for generating user IDs (default: NanoIDStrategy)
	IDPrefix   string     // prefix for generated IDs (e.g., "user", "member", default: "")

	// Configurable field lists (use defaults if not specified)
	PublicUserFields []interface{} // fields returned in public APIs
	BasicUserFields  []interface{} // minimal fields for basic user info
	// Note: AuthUserFields and MFAUserFields are fixed for security reasons

	// OAuth Account field lists (use defaults if not specified)
	OAuthAccountFields       []interface{} // basic OAuth account fields
	OAuthAccountDetailFields []interface{} // detailed OAuth account fields with OIDC claims

	// Role field lists (use defaults if not specified)
	RoleFields       []interface{} // basic role fields
	RoleDetailFields []interface{} // detailed role fields including permissions and metadata

	// Type field lists (use defaults if not specified)
	TypeFields       []interface{} // basic type fields
	TypeDetailFields []interface{} // detailed type fields including configuration and metadata

	// MFA configuration (use defaults if not specified)
	MFAOptions *types.MFAOptions // MFA settings
}

// NewDefaultUser creates a new DefaultUser
func NewDefaultUser(options *DefaultUserOptions) *DefaultUser {
	// Set default model names if not specified
	model := options.Model
	if model == "" {
		model = "__yao.user"
	}

	roleModel := options.RoleModel
	if roleModel == "" {
		roleModel = "__yao.user_role"
	}

	typeModel := options.TypeModel
	if typeModel == "" {
		typeModel = "__yao.user_type"
	}

	oauthAccountModel := options.OAuthAccountModel
	if oauthAccountModel == "" {
		oauthAccountModel = "__yao.user_oauth_account"
	}

	// Set ID generation strategy with defaults
	idStrategy := options.IDStrategy
	if idStrategy == "" {
		idStrategy = NumericStrategy // Default to Numeric for better UX
	}

	// Set ID prefix (default is empty string)
	idPrefix := options.IDPrefix

	// Set configurable field lists with defaults if not specified
	publicUserFields := options.PublicUserFields
	if publicUserFields == nil {
		publicUserFields = DefaultPublicUserFields
	}

	basicUserFields := options.BasicUserFields
	if basicUserFields == nil {
		basicUserFields = DefaultBasicUserFields
	}

	// Set OAuth account field lists with defaults if not specified
	oauthAccountFields := options.OAuthAccountFields
	if oauthAccountFields == nil {
		oauthAccountFields = DefaultOAuthAccountFields
	}

	oauthAccountDetailFields := options.OAuthAccountDetailFields
	if oauthAccountDetailFields == nil {
		oauthAccountDetailFields = DefaultOAuthAccountDetailFields
	}

	// Set role field lists with defaults if not specified
	roleFields := options.RoleFields
	if roleFields == nil {
		roleFields = DefaultRoleFields
	}

	roleDetailFields := options.RoleDetailFields
	if roleDetailFields == nil {
		roleDetailFields = DefaultRoleDetailFields
	}

	// Set type field lists with defaults if not specified
	typeFields := options.TypeFields
	if typeFields == nil {
		typeFields = DefaultTypeFields
	}

	typeDetailFields := options.TypeDetailFields
	if typeDetailFields == nil {
		typeDetailFields = DefaultTypeDetailFields
	}

	// Set MFA options with defaults if not specified
	mfaOptions := options.MFAOptions
	if mfaOptions == nil {
		mfaOptions = DefaultMFAOptions
	}

	return &DefaultUser{
		prefix:            options.Prefix,
		model:             model,
		roleModel:         roleModel,
		typeModel:         typeModel,
		oauthAccountModel: oauthAccountModel,
		cache:             options.Cache,
		idStrategy:        idStrategy,
		idPrefix:          idPrefix,
		publicUserFields:  publicUserFields,
		basicUserFields:   basicUserFields,
		authUserFields:    DefaultAuthUserFields, // fixed for security
		mfaUserFields:     DefaultMFAUserFields,  // fixed for security

		// OAuth Account field lists
		oauthAccountFields:       oauthAccountFields,
		oauthAccountDetailFields: oauthAccountDetailFields,

		// Role field lists
		roleFields:       roleFields,
		roleDetailFields: roleDetailFields,

		// Type field lists
		typeFields:       typeFields,
		typeDetailFields: typeDetailFields,

		// MFA Configuration
		mfaOptions: mfaOptions,
	}
}
