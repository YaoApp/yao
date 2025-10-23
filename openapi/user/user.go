package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func init() {
	// Register builtin scopes for temporary tokens (before ACL initialization)
	// These scopes grant limited access to specific endpoints for special purposes
	acl.Register(
		// MFA verification scope - allows users to complete MFA setup during login
		&acl.ScopeDefinition{
			Name:        ScopeMFAVerification,
			Description: "MFA verification - temporary access for completing MFA challenge",
			Endpoints: []string{
				"POST /user/mfa/totp/verify",
				"POST /user/mfa/sms/verify",
				"GET /user/mfa/totp",
			},
		},
		// Team selection scope - allows users to select a team and issue new tokens
		&acl.ScopeDefinition{
			Name:        ScopeTeamSelection,
			Description: "Team selection - temporary access for selecting a team after login",
			Endpoints: []string{
				"POST /user/teams/select",
				"GET /user/teams/config",
			},
		},
		// Invite verification scope - allows users to accept team invitations
		&acl.ScopeDefinition{
			Name:        ScopeInviteVerification,
			Description: "Invite verification - temporary access for accepting team invitations",
			Endpoints: []string{
				"POST /user/teams/invitations/:invitation_id/accept",
				"GET /user/teams/invitations/:invitation_id",
			},
		},
		// Entry verification scope - allows users to complete registration or login verification
		&acl.ScopeDefinition{
			Name:        ScopeEntryVerification,
			Description: "Entry verification - temporary access for completing registration or login verification",
			Endpoints: []string{
				"POST /user/entry/register",
				"POST /user/entry/login",
				"POST /user/entry/invite/verify",
				"POST /user/entry/otp",
			},
		},
	)

	// Register user process handlers
	process.RegisterGroup("user", map[string]process.Handler{
		"team.list":              ProcessTeamList,
		"team.get":               ProcessTeamGet,
		"team.create":            ProcessTeamCreate,
		"team.update":            ProcessTeamUpdate,
		"team.delete":            ProcessTeamDelete,
		"team.invitation.list":   ProcessTeamInvitationList,
		"team.invitation.get":    ProcessTeamInvitationGet,
		"team.invitation.create": ProcessTeamInvitationCreate,
		"team.invitation.resend": ProcessTeamInvitationResend,
		"team.invitation.delete": ProcessTeamInvitationDelete,
	})
}

// Attach attaches the signin handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// User Authentication
	group.GET("/entry", getEntryConfig)         // Get unified auth entry config (public)
	group.GET("/entry/captcha", getCaptcha)     // Get captcha for login/register (public)
	group.POST("/entry/verify", GinEntryVerify) // Verify login/register email or mobile (public)

	// Register a new user
	group.POST("/entry/register", oauth.Guard, GinEntryRegister)     // Register a new user
	group.POST("/entry/login", oauth.Guard, GinEntryLogin)           // Login a user
	group.POST("/entry/invite/verify", oauth.Guard, GinVerifyInvite) // Verify invitation code (redeem)
	group.POST("/entry/otp", oauth.Guard, GinSendOTP)                // Send OTP
	group.POST("/logout", oauth.Guard, GinLogout)                    // User logout

	// Logined User Settings
	attachProfile(group, oauth)      // User profile management
	attachPreferences(group, oauth)  // User preferences management
	attachAccount(group, oauth)      // Account settings
	attachThirdParty(group, oauth)   // Third party login
	attachMFA(group, oauth)          // MFA settings
	attachCredits(group, oauth)      // User credits management
	attachSubscription(group, oauth) // User subscription management
	attachAPIKeys(group, oauth)      // User API keys management
	attachUsage(group, oauth)        // User usage management
	attachBilling(group, oauth)      // User billing management
	attachReferral(group, oauth)     // User referral management
	attachTeam(group, oauth)         // User team management
	attachInvitations(group, oauth)  // Invitation response management
	attachPrivacy(group, oauth)      // User privacy management

	// User Management
	attachUsers(group, oauth)
}

// User Team Management
func attachTeam(group *gin.RouterGroup, oauth types.OAuth) {
	// Public endpoint for viewing team invitations (no auth required)
	// Must be registered BEFORE the team group with auth guard
	group.GET("/teams/invitations/:invitation_id", GinTeamInvitationGetPublic)                   // GET /user/teams/invitations/:invitation_id - Get invitation details (public)
	group.POST("/teams/invitations/:invitation_id/accept", oauth.Guard, GinTeamInvitationAccept) // POST /user/teams/invitations/:invitation_id/accept - Accept invitation and login

	team := group.Group("/teams")

	// Protected endpoints (authentication required)
	team.Use(oauth.Guard)

	// Team Configuration
	team.GET("/config", GinTeamConfig) // Get team configuration (requires authentication)

	// Team Selection
	team.POST("/select", GinTeamSelection) // POST /teams/select - Select a team and issue tokens with team_id (requires authentication)

	// Team CRUD - Standard REST endpoints
	team.GET("/", GinTeamList)         // GET /teams - List user teams
	team.POST("/", GinTeamCreate)      // POST /teams - Create new team
	team.GET("/:id", GinTeamGet)       // GET /teams/:id - Get team details
	team.PUT("/:id", GinTeamUpdate)    // PUT /teams/:id - Update team
	team.DELETE("/:id", GinTeamDelete) // DELETE /teams/:id - Delete team

	// Get Current Team
	team.GET("/current", GinTeamCurrent)

	// Team Members - Nested resource endpoints
	team.GET("/:id/members", GinMemberList)                 // GET /teams/:id/members - List team members
	team.POST("/:id/members", GinMemberCreateDirect)        // POST /teams/:id/members - Add team member
	team.GET("/:id/members/:member_id", GinMemberGet)       // GET /teams/:id/members/:member_id - Get member details
	team.PUT("/:id/members/:member_id", GinMemberUpdate)    // PUT /teams/:id/members/:member_id - Update member
	team.DELETE("/:id/members/:member_id", GinMemberDelete) // DELETE /teams/:id/members/:member_id - Remove member

	// Team Invitations - Nested resource endpoints
	team.GET("/:id/invitations", GinTeamInvitationList)                         // GET /teams/:id/invitations - List invitations
	team.POST("/:id/invitations", GinTeamInvitationCreate)                      // POST /teams/:id/invitations - Send invitation
	team.GET("/:id/invitations/:invitation_id", GinTeamInvitationGet)           // GET /teams/:id/invitations/:invitation_id - Get invitation (admin)
	team.PUT("/:id/invitations/:invitation_id/resend", GinTeamInvitationResend) // PUT /teams/:id/invitations/:invitation_id/resend - Resend invitation
	team.DELETE("/:id/invitations/:invitation_id", GinTeamInvitationDelete)     // DELETE /teams/:id/invitations/:invitation_id - Cancel invitation
}

// Invitation Response Management (Cross-module invitation handling)
func attachInvitations(group *gin.RouterGroup, oauth types.OAuth) {
	// Public endpoints for invitation recipients
	group.GET("/invitations/:token", placeholder)                      // Get invitation info by token (public)
	group.POST("/invitations/:token/accept", oauth.Guard, placeholder) // Accept invitation (requires login)
	group.POST("/invitations/:token/decline", placeholder)             // Decline invitation (public)
}

// User Privacy
func attachPrivacy(group *gin.RouterGroup, oauth types.OAuth) {
	privacy := group.Group("/privacy")
	privacy.Use(oauth.Guard)
	privacy.GET("/", placeholder)       // Get user privacy
	privacy.GET("/schema", placeholder) // Get user privacy schema
	privacy.PUT("/", placeholder)       // Update user privacy
}

// User Preferences
func attachPreferences(group *gin.RouterGroup, oauth types.OAuth) {
	preferences := group.Group("/preferences")
	preferences.Use(oauth.Guard)

	preferences.GET("/", placeholder)       // Get user preferences
	preferences.GET("/schema", placeholder) // Get user preferences schema
	preferences.PUT("/", placeholder)       // Update user preferences
}

// User Billing Management
func attachBilling(group *gin.RouterGroup, oauth types.OAuth) {
	billing := group.Group("/billing")
	billing.Use(oauth.Guard)
	billing.GET("/history", placeholder)  // Get user billing history
	billing.GET("/invoices", placeholder) // Get user invoices list
}

// Referral Management
func attachReferral(group *gin.RouterGroup, oauth types.OAuth) {
	referral := group.Group("/referral")
	referral.Use(oauth.Guard)

	referral.GET("/code", placeholder)        // Get user referral code
	referral.GET("/statistics", placeholder)  // Get user referral statistics
	referral.GET("/history", placeholder)     // Get user referral history
	referral.GET("/commissions", placeholder) // Get user referral commissions
}

// User Credits Management
func attachCredits(group *gin.RouterGroup, oauth types.OAuth) {
	credits := group.Group("/credits")
	credits.Use(oauth.Guard)

	credits.GET("/", placeholder)        // Get user credits info
	credits.GET("/history", placeholder) // Get credits change history

	// Top-up Management
	topup := credits.Group("/topup")
	topup.GET("/", placeholder)           // Get topup records
	topup.POST("/", placeholder)          // Create topup order
	topup.GET("/:order_id", placeholder)  // Get topup order status
	topup.POST("/card-code", placeholder) // Redeem card code
}

// Usage Management
func attachUsage(group *gin.RouterGroup, oauth types.OAuth) {
	usage := group.Group("/usage")
	usage.Use(oauth.Guard)
	usage.GET("/statistics", placeholder) // Get user usage statistics
	usage.GET("/history", placeholder)    // Get user usage history
}

// User API Keys Management
func attachAPIKeys(group *gin.RouterGroup, oauth types.OAuth) {
	apiKeys := group.Group("/api-keys")
	apiKeys.Use(oauth.Guard)

	apiKeys.GET("/", placeholder)                    // Get all user API keys
	apiKeys.POST("/", placeholder)                   // Create new API key
	apiKeys.GET("/:key_id", placeholder)             // Get specific API key details
	apiKeys.PUT("/:key_id", placeholder)             // Update API key (name, permissions)
	apiKeys.DELETE("/:key_id", placeholder)          // Delete API key
	apiKeys.POST("/:key_id/regenerate", placeholder) // Regenerate API key
}

// User Subscription Management
func attachSubscription(group *gin.RouterGroup, oauth types.OAuth) {
	subscription := group.Group("/subscription")
	subscription.Use(oauth.Guard)
	subscription.GET("/", placeholder) // Get user subscription
	subscription.PUT("/", placeholder) // Update user subscription
}

// User profile management
func attachProfile(group *gin.RouterGroup, oauth types.OAuth) {
	profile := group.Group("/profile")
	profile.Use(oauth.Guard)

	profile.GET("/", GinProfileGet) // Get user profile
	profile.PUT("/", placeholder)   // Update user profile
}

// User management (CRUD)
func attachUsers(group *gin.RouterGroup, oauth types.OAuth) {
	users := group.Group("/users")
	users.Use(oauth.Guard)

	users.GET("/", placeholder)            // Get users
	users.GET("/:user_id", placeholder)    // Get user details
	users.POST("/", placeholder)           // Create user
	users.PUT("/:user_id", placeholder)    // Update user
	users.DELETE("/:user_id", placeholder) // Delete user
}

// Account settings
func attachAccount(group *gin.RouterGroup, oauth types.OAuth) {
	account := group.Group("/account")
	account.Use(oauth.Guard)

	// Password Management
	account.PUT("/password", placeholder)                // Change password (requires current password or 2FA)
	account.POST("/password/reset/request", placeholder) // Request password reset (public, rate-limited)
	account.POST("/password/reset/verify", placeholder)  // Verify reset token and set new password (public)

	// Email Management
	account.GET("/email", placeholder)                    // Get current email info
	account.POST("/email/change/request", placeholder)    // Request email change (sends code to current email)
	account.POST("/email/change/verify", placeholder)     // Verify email change with code
	account.POST("/email/verification-code", placeholder) // Send verification code to current email
	account.POST("/email/verify", placeholder)            // Verify current email

	// Mobile Management
	account.GET("/mobile", placeholder)                    // Get current mobile info
	account.POST("/mobile/change/request", placeholder)    // Request mobile change
	account.POST("/mobile/change/verify", placeholder)     // Verify mobile change with code
	account.POST("/mobile/verification-code", placeholder) // Send verification code to mobile
	account.POST("/mobile/verify", placeholder)            // Verify current mobile
}

// MFA settings
func attachMFA(group *gin.RouterGroup, oauth types.OAuth) {
	mfa := group.Group("/mfa")
	mfa.Use(oauth.Guard)

	// TOTP Management
	mfa.GET("/totp", placeholder)                            // Get TOTP QR code and setup info
	mfa.POST("/totp/enable", placeholder)                    // Enable TOTP with verification
	mfa.POST("/totp/disable", placeholder)                   // Disable TOTP with verification
	mfa.POST("/totp/verify", placeholder)                    // Verify TOTP code
	mfa.GET("/totp/recovery-codes", placeholder)             // Get TOTP recovery codes
	mfa.POST("/totp/recovery-codes/regenerate", placeholder) // Regenerate recovery codes
	mfa.POST("/totp/reset", placeholder)                     // Reset TOTP (requires email verification)

	// SMS MFA Management
	mfa.GET("/sms", placeholder)                    // Get SMS MFA status
	mfa.POST("/sms/enable", placeholder)            // Enable SMS MFA
	mfa.POST("/sms/disable", placeholder)           // Disable SMS MFA
	mfa.POST("/sms/verification-code", placeholder) // Send SMS verification code
	mfa.POST("/sms/verify", placeholder)            // Verify SMS code
}

// Third party login (OAuth)
func attachThirdParty(group *gin.RouterGroup, oauth types.OAuth) {

	thirdParty := group.Group("/oauth")                       // OAuth
	thirdParty.GET("/providers", oauth.Guard, placeholder)    // Get linked OAuth providers
	thirdParty.DELETE("/:provider", oauth.Guard, placeholder) // Unlink OAuth provider

	thirdParty.GET("/providers/available", placeholder)              // Get available OAuth providers
	thirdParty.GET("/:provider/authorize", getOAuthAuthorizationURL) // Get OAuth authorization URL - migrated from /signin/oauth/:provider/authorize
	thirdParty.POST("/:provider/connect", oauth.Guard, placeholder)  // Connect OAuth provider
	thirdParty.POST("/:provider/authorize/prepare", authbackPrepare) // OAuth authorization prepare - migrated from /signin/oauth/:provider/authorize/prepare
	thirdParty.POST("/:provider/callback", authback)                 // Handle OAuth callback - migrated from /signin/oauth/:provider/authback

}

func placeholder(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
}
