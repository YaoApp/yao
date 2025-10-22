package user

import (
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// LoginStatus represents the login status
type LoginStatus string

// EntryVerificationStatus represents the entry verification status
type EntryVerificationStatus string

const (
	// LoginStatusSuccess is the success status
	LoginStatusSuccess LoginStatus = "ok"
	// LoginStatusMFA is the MFA status
	LoginStatusMFA LoginStatus = "mfa_required"
	// LoginStatusTeamSelection is the team selection status
	LoginStatusTeamSelection LoginStatus = "team_selection_required"
	// LoginStatusInviteRequired is the invite required status (for registration response)
	LoginStatusInviteRequired LoginStatus = "invite_required"
	// LoginStatusInviteVerification is the invite verification status (for login response)
	LoginStatusInviteVerification LoginStatus = "invite_verification_required"
)

const (
	// EntryVerificationStatusLogin is the login status
	EntryVerificationStatusLogin EntryVerificationStatus = "login"
	// EntryVerificationStatusRegister is the register status
	EntryVerificationStatusRegister EntryVerificationStatus = "register"
	// EntryVerificationStatusInviteRequired is the invite required status (user registered but needs invite code)
	EntryVerificationStatusInviteRequired EntryVerificationStatus = "invite_required"
)

const (
	// ScopeMFAVerification is the MFA verification scope for temporary access token
	ScopeMFAVerification = "builtin:mfa:verification"
	// ScopeTeamSelection is the team selection scope for temporary access token
	ScopeTeamSelection = "builtin:teams:selection"
	// ScopeInviteVerification is the invite verification scope for temporary access token
	ScopeInviteVerification = "builtin:invite:verification"
	// ScopeEntryVerification is the entry verification scope for temporary access token (login or register)
	ScopeEntryVerification = "builtin:entry:verification"
)

// FormConfig represents the form configuration
type FormConfig struct {
	Username           *UsernameConfig `json:"username,omitempty"`
	Password           *PasswordConfig `json:"password,omitempty"`
	ConfirmPassword    *PasswordConfig `json:"confirm_password,omitempty"`
	Captcha            *CaptchaConfig  `json:"captcha,omitempty"`
	ForgotPasswordLink bool            `json:"forgot_password_link,omitempty"`
	RememberMe         bool            `json:"remember_me,omitempty"`
	RegisterLink       string          `json:"register_link,omitempty"`
	LoginLink          string          `json:"login_link,omitempty"`
	TermsOfServiceLink string          `json:"terms_of_service_link,omitempty"`
	PrivacyPolicyLink  string          `json:"privacy_policy_link,omitempty"`
}

// UsernameConfig represents the username field configuration
type UsernameConfig struct {
	Placeholder string   `json:"placeholder,omitempty"`
	Fields      []string `json:"fields,omitempty"`
}

// PasswordConfig represents the password field configuration
type PasswordConfig struct {
	Placeholder string `json:"placeholder,omitempty"`
}

// CaptchaConfig represents the captcha configuration
type CaptchaConfig struct {
	Type    string                 `json:"type,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// TokenConfig represents the token configuration
type TokenConfig struct {
	ExpiresIn                       string `json:"expires_in,omitempty"`
	RefreshTokenExpiresIn           string `json:"refresh_token_expires_in,omitempty"`
	RememberMeExpiresIn             string `json:"remember_me_expires_in,omitempty"`
	RememberMeRefreshTokenExpiresIn string `json:"remember_me_refresh_token_expires_in,omitempty"`
}

// ThirdParty represents the third party login configuration
type ThirdParty struct {
	Providers []*Provider `json:"providers,omitempty"`
}

// ProviderRegisterConfig represents the auto register configuration in provider
type ProviderRegisterConfig struct {
	Auto bool `json:"auto,omitempty"`
}

// EntryConfig represents the unified auth entry configuration (login + register)
// This merges signin and register configurations into a single entry point
type EntryConfig struct {
	Title          string            `json:"title,omitempty"`
	Description    string            `json:"description,omitempty"`
	Default        bool              `json:"default,omitempty"`
	SuccessURL     string            `json:"success_url,omitempty"`
	FailureURL     string            `json:"failure_url,omitempty"`
	LogoutRedirect string            `json:"logout_redirect,omitempty"` // From signin config
	ClientID       string            `json:"client_id,omitempty"`       // From signin config
	ClientSecret   string            `json:"client_secret,omitempty"`   // From signin config (not exposed to frontend)
	AutoLogin      bool              `json:"auto_login,omitempty"`      // From register config
	Role           string            `json:"role,omitempty"`            // From register config
	Type           string            `json:"type,omitempty"`            // From register config - User type id
	Form           *FormConfig       `json:"form,omitempty"`
	Token          *TokenConfig      `json:"token,omitempty"`           // From signin config
	Messenger      *MessengerConfig  `json:"messenger,omitempty"`       // From register config
	InviteRequired bool              `json:"invite_required,omitempty"` // From register config
	Invite         *InvitePageConfig `json:"invite,omitempty"`          // Invite code page configuration
	ThirdParty     *ThirdParty       `json:"third_party,omitempty"`
}

// MessengerConfig represents the messenger configuration for user registration
type MessengerConfig struct {
	Mail *MessengerChannelConfig `json:"mail,omitempty"` // Email verification config
	SMS  *MessengerChannelConfig `json:"sms,omitempty"`  // SMS verification config
}

// MessengerChannelConfig represents a single messenger channel configuration
type MessengerChannelConfig struct {
	Channel  string `json:"channel,omitempty"`  // Messenger channel name (e.g., "default", "aws_ses")
	Template string `json:"template,omitempty"` // Template name for this channel
}

// InvitePageConfig represents the invitation code page configuration
type InvitePageConfig struct {
	Title       string `json:"title,omitempty"`        // Page title for invite code verification
	Description string `json:"description,omitempty"`  // Description text for invite code page
	Placeholder string `json:"placeholder,omitempty"`  // Placeholder text for invite code input
	ApplyLink   string `json:"apply_link,omitempty"`   // Optional link to apply for invitation code
	ApplyPrompt string `json:"apply_prompt,omitempty"` // Prompt text before apply link (e.g., "Don't have an invitation code?")
	ApplyText   string `json:"apply_text,omitempty"`   // Text for apply link (e.g., "Apply for invitation code")
}

// YaoClientConfig represents the Yao OpenAPI Client config
type YaoClientConfig struct {
	ClientID              string   `json:"client_id,omitempty"`
	ClientSecret          string   `json:"client_secret,omitempty"`
	Scopes                []string `json:"scopes,omitempty"`                   // Default scopes if not set in the provider config
	ExpiresIn             int      `json:"expires_in,omitempty"`               // Default expires in for the access token (optional) in seconds
	RefreshTokenExpiresIn int      `json:"refresh_token_expires_in,omitempty"` // Default expires in for the refresh token (optional) in seconds
}

// Provider represents a third party login provider
type Provider struct {
	ID                    string                  `json:"id,omitempty"`
	Label                 string                  `json:"label,omitempty"`
	Title                 string                  `json:"title,omitempty"`
	Logo                  string                  `json:"logo,omitempty"`
	Color                 string                  `json:"color,omitempty"`
	TextColor             string                  `json:"text_color,omitempty"`
	ClientID              string                  `json:"client_id,omitempty"`
	ClientSecret          string                  `json:"client_secret,omitempty"`
	ClientSecretGenerator *SecretGenerator        `json:"client_secret_generator,omitempty"`
	Scopes                []string                `json:"scopes,omitempty"`
	ResponseMode          string                  `json:"response_mode,omitempty"`
	UserInfoSource        string                  `json:"user_info_source,omitempty"` // "endpoint" (default) | "id_token" | "access_token"
	Endpoints             *Endpoints              `json:"endpoints,omitempty"`
	Mapping               interface{}             `json:"mapping,omitempty"` // string (preset) | map[string]string (custom) | nil (generic)
	Register              *ProviderRegisterConfig `json:"register,omitempty"`
}

// SecretGenerator represents the client secret generator configuration
type SecretGenerator struct {
	Type       string                 `json:"type,omitempty"`
	ExpiresIn  string                 `json:"expires_in,omitempty"`
	PrivateKey string                 `json:"private_key,omitempty"`
	Header     map[string]interface{} `json:"header,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
}

// Endpoints represents the OAuth endpoints
type Endpoints struct {
	Authorization string `json:"authorization,omitempty"`
	Token         string `json:"token,omitempty"`
	UserInfo      string `json:"user_info,omitempty"`
	JWKS          string `json:"jwks,omitempty"` // JSON Web Key Set endpoint for token verification
}

// ==== API Types ====

// OAuthAuthorizationURLResponse represents the response for OAuth authorization URL
type OAuthAuthorizationURLResponse struct {
	AuthorizationURL string   `json:"authorization_url"`
	State            string   `json:"state"`
	Warnings         []string `json:"warnings,omitempty"` // Optional warnings about state format or other issues
}

// OAuthCallbackResponse represents the response for OAuth callback
type OAuthCallbackResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// OAuthAuthbackRequest represents the request for OAuth callback
type OAuthAuthbackRequest struct {
	Locale   string `json:"locale" form:"locale"`
	Code     string `json:"code" form:"code"`
	State    string `json:"state" form:"state"`
	Provider string `json:"provider" form:"provider"`
	Scope    string `json:"scope,omitempty" form:"scope,omitempty"`
}

// OAuthTokenResponse represents the response from OAuth token endpoint
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token,omitempty"` // JWT token containing user info (Apple, etc.)
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// OAuthTokenRequest represents the request to OAuth token endpoint
type OAuthTokenRequest struct {
	GrantType    string `json:"grant_type" form:"grant_type"`
	Code         string `json:"code" form:"code"`
	ClientID     string `json:"client_id" form:"client_id"`
	ClientSecret string `json:"client_secret" form:"client_secret"`
	RedirectURI  string `json:"redirect_uri,omitempty" form:"redirect_uri,omitempty"`
}

// OAuthUserInfoResponse is an alias for OIDC standard user information type
type OAuthUserInfoResponse = oauthtypes.OIDCUserInfo

// OIDCAddress is an alias for OIDC standard address claim type
type OIDCAddress = oauthtypes.OIDCAddress

// LoginResponse represents the response for login
type LoginResponse struct {
	UserID                string      `json:"user_id,omitempty"`
	Subject               string      `json:"subject,omitempty"`
	AccessToken           string      `json:"access_token"`
	IDToken               string      `json:"id_token,omitempty"`
	RefreshToken          string      `json:"refresh_token,omitempty"`
	ExpiresIn             int         `json:"expires_in,omitempty"`
	RefreshTokenExpiresIn int         `json:"refresh_token_expires_in,omitempty"`
	TokenType             string      `json:"token_type,omitempty"`
	MFAEnabled            bool        `json:"mfa_enabled,omitempty"`
	Scope                 string      `json:"scope,omitempty"`
	Status                LoginStatus `json:"status,omitempty"`
}

// LoginSuccessResponse represents the response for login success
type LoginSuccessResponse struct {
	UserID                string      `json:"user_id,omitempty"` // User ID (optional, for registration)
	Message               string      `json:"message,omitempty"` // Success message (optional, for registration)
	IDToken               string      `json:"id_token,omitempty"`
	AccessToken           string      `json:"access_token,omitempty"`
	SessionID             string      `json:"session_id,omitempty"`
	RefreshToken          string      `json:"refresh_token,omitempty"`
	ExpiresIn             int         `json:"expires_in,omitempty"`
	MFAEnabled            bool        `json:"mfa_enabled"`
	RefreshTokenExpiresIn int         `json:"refresh_token_expires_in,omitempty"`
	Status                LoginStatus `json:"status,omitempty"`
}

// LoginContext is an alias for the oauth types LoginContext
type LoginContext = oauthtypes.LoginContext

// ==== Entry Verification Types ====

// EntryVerifyRequest represents the request to verify entry (login/register)
type EntryVerifyRequest struct {
	Username  string `json:"username" binding:"required"` // Email or mobile
	CaptchaID string `json:"captcha_id,omitempty"`        // Captcha ID (for image captcha)
	Captcha   string `json:"captcha,omitempty"`           // Captcha answer or token
	Locale    string `json:"locale,omitempty"`            // Locale for localized responses
}

// EntryVerifyResponse represents the response for entry verification
type EntryVerifyResponse struct {
	Status           EntryVerificationStatus `json:"status"`                      // "login" or "register" or "invite_required"
	AccessToken      string                  `json:"access_token"`                // Temporary token for next step
	ExpiresIn        int                     `json:"expires_in"`                  // Token expiration in seconds
	TokenType        string                  `json:"token_type"`                  // Token type (Bearer)
	Scope            string                  `json:"scope"`                       // Token scope
	UserExists       bool                    `json:"user_exists"`                 // Whether user exists
	VerificationSent bool                    `json:"verification_sent,omitempty"` // Whether verification code was sent (for register)
	OtpID            string                  `json:"otp_id,omitempty"`            // OTP ID for verification code (for register)
}

// EntryRegisterRequest represents the request to register a new user
type EntryRegisterRequest struct {
	Name             string `json:"name,omitempty"` // User's display name (optional)
	Password         string `json:"password" binding:"required"`
	ConfirmPassword  string `json:"confirm_password,omitempty"`
	OtpID            string `json:"otp_id,omitempty"`            // OTP ID from entry verify response
	VerificationCode string `json:"verification_code,omitempty"` // Verification code from email/SMS
	Locale           string `json:"locale,omitempty"`
}

// EntryLoginRequest represents the request to login with username and password
type EntryLoginRequest struct {
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"remember_me,omitempty"`
	Locale     string `json:"locale,omitempty"`
}

// EntrySendOTPResponse represents the response for sending OTP verification code
type EntrySendOTPResponse struct {
	OtpID     string `json:"otp_id"`               // OTP ID for verification
	ExpiresIn int    `json:"expires_in,omitempty"` // OTP expiration in seconds
}

// Built-in preset mapping types
const (
	MappingGoogle    = "google"
	MappingGitHub    = "github"
	MappingMicrosoft = "microsoft"
	MappingApple     = "apple"
	MappingWeChat    = "wechat"
	MappingGeneric   = "generic"
)

// User info source types
const (
	UserInfoSourceEndpoint    = "endpoint"     // Default: Get user info from dedicated endpoint
	UserInfoSourceIDToken     = "id_token"     // Extract user info from ID token (JWT)
	UserInfoSourceAccessToken = "access_token" // Extract user info from access token response
)

// ==== Settings Types ====

// TeamSettings represents team-specific settings
type TeamSettings struct {
	Theme      string `json:"theme,omitempty"`      // Team UI theme (e.g., "light", "dark")
	Visibility string `json:"visibility,omitempty"` // Team visibility (e.g., "public", "private")
}

// MemberSettings represents member-specific settings
type MemberSettings struct {
	Notifications bool     `json:"notifications,omitempty"` // Whether to receive notifications
	Permissions   []string `json:"permissions,omitempty"`   // Custom permissions (e.g., ["read", "write"])
}

// InvitationSettings represents invitation-specific settings
type InvitationSettings struct {
	SendEmail bool   `json:"send_email,omitempty"` // Whether to send invitation email
	Locale    string `json:"locale,omitempty"`     // Locale for email template
}

// ==== Team API Types ====

// TeamResponse represents a team in API responses
type TeamResponse struct {
	ID          int64  `json:"id"`
	TeamID      string `json:"team_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	OwnerID     string `json:"owner_id"`
	Status      string `json:"status"`
	IsVerified  bool   `json:"is_verified"`
	VerifiedBy  string `json:"verified_by,omitempty"`
	VerifiedAt  string `json:"verified_at,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// TeamDetailResponse represents detailed team information
type TeamDetailResponse struct {
	TeamResponse
	// Add additional fields that are only included in detailed responses
	Settings *TeamSettings `json:"settings,omitempty"`
}

// CreateTeamRequest represents the request to create a team
type CreateTeamRequest struct {
	Name        string        `json:"name" binding:"required"`
	Description string        `json:"description,omitempty"`
	Settings    *TeamSettings `json:"settings,omitempty"`
}

// UpdateTeamRequest represents the request to update a team
type UpdateTeamRequest struct {
	Name        string        `json:"name,omitempty"`
	Description string        `json:"description,omitempty"`
	Settings    *TeamSettings `json:"settings,omitempty"`
}

// TeamSelectionRequest represents the request to select a team
type TeamSelectionRequest struct {
	TeamID string `json:"team_id" binding:"required"`
}

// ==== Member API Types ====

// MemberResponse represents a team member in API responses
type MemberResponse struct {
	ID           int64           `json:"id"`
	TeamID       string          `json:"team_id"`
	UserID       string          `json:"user_id"`
	MemberType   string          `json:"member_type"`
	RoleID       string          `json:"role_id"`
	Status       string          `json:"status"`
	InvitedBy    string          `json:"invited_by,omitempty"`
	InvitedAt    string          `json:"invited_at,omitempty"`
	JoinedAt     string          `json:"joined_at,omitempty"`
	LastActivity string          `json:"last_activity,omitempty"`
	Settings     *MemberSettings `json:"settings,omitempty"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

// MemberDetailResponse represents detailed member information
type MemberDetailResponse struct {
	MemberResponse
	// Add additional fields that are only included in detailed responses
	UserInfo map[string]interface{} `json:"user_info,omitempty"`
}

// CreateMemberRequest represents the request to add a member directly
type CreateMemberRequest struct {
	UserID     string          `json:"user_id" binding:"required"`
	MemberType string          `json:"member_type,omitempty"` // "user" or "robot"
	RoleID     string          `json:"role_id" binding:"required"`
	Settings   *MemberSettings `json:"settings,omitempty"`
}

// UpdateMemberRequest represents the request to update a member
type UpdateMemberRequest struct {
	RoleID       string          `json:"role_id,omitempty"`
	Status       string          `json:"status,omitempty"`
	Settings     *MemberSettings `json:"settings,omitempty"`
	LastActivity string          `json:"last_activity,omitempty"`
}

// ==== Invitation API Types ====

// InvitationResponse represents a team invitation in API responses
type InvitationResponse struct {
	ID                  int64               `json:"id"`
	InvitationID        string              `json:"invitation_id"`
	TeamID              string              `json:"team_id"`
	UserID              string              `json:"user_id"`
	MemberType          string              `json:"member_type"`
	RoleID              string              `json:"role_id"`
	Status              string              `json:"status"`
	InvitedBy           string              `json:"invited_by"`
	InvitedAt           string              `json:"invited_at"`
	InvitationToken     string              `json:"invitation_token,omitempty"`
	InvitationLink      string              `json:"invitation_link,omitempty"` // Full invitation link
	InvitationExpiresAt string              `json:"invitation_expires_at,omitempty"`
	Message             string              `json:"message,omitempty"`
	Settings            *InvitationSettings `json:"settings,omitempty"`
	CreatedAt           string              `json:"created_at"`
	UpdatedAt           string              `json:"updated_at"`
}

// InvitationDetailResponse represents detailed invitation information
type InvitationDetailResponse struct {
	InvitationResponse
	// Add additional fields that are only included in detailed responses
	UserInfo map[string]interface{} `json:"user_info,omitempty"`
	TeamInfo map[string]interface{} `json:"team_info,omitempty"`
}

// PublicInvitationResponse represents a public team invitation (for invitation recipients)
// This type excludes sensitive information like tokens, database IDs, and timestamps
type PublicInvitationResponse struct {
	InvitationID        string       `json:"invitation_id"`
	TeamName            string       `json:"team_name"`
	TeamLogo            string       `json:"team_logo"`        // Always return, empty string if not set
	TeamDescription     string       `json:"team_description"` // Always return, empty string if not set
	RoleLabel           string       `json:"role_label,omitempty"`
	Status              string       `json:"status"`
	InvitedAt           string       `json:"invited_at"`
	InvitationExpiresAt string       `json:"invitation_expires_at,omitempty"`
	Message             string       `json:"message,omitempty"`
	InviterInfo         *InviterInfo `json:"inviter_info,omitempty"` // Inviter's public info
}

// InviterInfo represents public information about the person who sent the invitation
type InviterInfo struct {
	UserID  string `json:"user_id"` // Inviter's user ID
	Name    string `json:"name,omitempty"`
	Picture string `json:"picture"` // Always return, empty string if not set
}

// CreateInvitationRequest represents the request to send a team invitation
type CreateInvitationRequest struct {
	UserID     string              `json:"user_id,omitempty"`     // Optional for unregistered users
	Email      string              `json:"email,omitempty"`       // Email address (if not provided, will be read from user profile when user_id is provided)
	MemberType string              `json:"member_type,omitempty"` // "user" or "robot"
	RoleID     string              `json:"role_id" binding:"required"`
	Message    string              `json:"message,omitempty"`
	Expiry     string              `json:"expiry,omitempty"`     // Custom expiry duration (e.g., "1d", "8h"), defaults to team config
	SendEmail  *bool               `json:"send_email,omitempty"` // Whether to send email (defaults to false)
	Locale     string              `json:"locale,omitempty"`     // Language code for email template (e.g., "zh-CN", "en")
	Settings   *InvitationSettings `json:"settings,omitempty"`
}

// ==== Team Configuration Types ====

// TeamConfig represents the team configuration loaded from DSL files
type TeamConfig struct {
	Roles  []*TeamRole   `json:"roles,omitempty"`
	Invite *InviteConfig `json:"invite,omitempty"`
	Type   string        `json:"type,omitempty"` // Default subscription type for new teams
	Role   string        `json:"role,omitempty"` // Default user role for team creator
}

// TeamRole represents a team role configuration
type TeamRole struct {
	RoleID      string `json:"role_id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Default     bool   `json:"default"`  // Whether this role is the default role
	Hidden      bool   `json:"hidden"`   // Whether this role is hidden from UI
	IsOwner     bool   `json:"is_owner"` // Whether this role represents team owner (deprecated, use config.Role instead)
}

// InviteConfig represents the invitation configuration
type InviteConfig struct {
	Channel   string            `json:"channel,omitempty"`
	Expiry    string            `json:"expiry,omitempty"`
	BaseURL   string            `json:"base_url,omitempty"` // Base URL for invitation links
	Templates map[string]string `json:"templates,omitempty"`
}
