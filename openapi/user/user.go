package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func init() {
	// Register user process handlers
	process.RegisterGroup("user", map[string]process.Handler{
		"team.list":   ProcessTeamList,
		"team.get":    ProcessTeamGet,
		"team.create": ProcessTeamCreate,
		"team.update": ProcessTeamUpdate,
		"team.delete": ProcessTeamDelete,
	})
}

// Attach attaches the signin handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// User Authentication (migrated from /signin)
	group.GET("/login", getLoginConfig)             // Get login page config (public) - migrated from /signin
	group.POST("/login", login)                     // User login (public) - migrated from /signin
	group.GET("/login/captcha", getCaptcha)         // Get captcha for login (public)
	group.POST("/register", placeholder)            // User register (public)
	group.POST("/logout", oauth.Guard, placeholder) // User logout

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
	team := group.Group("/teams")
	team.Use(oauth.Guard)

	// Team CRUD
	team.GET("/", GinTeamList)              // Get user teams
	team.GET("/:team_id", GinTeamGet)       // Get user team details
	team.POST("/", GinTeamCreate)           // Create user team
	team.PUT("/:team_id", GinTeamUpdate)    // Update user team
	team.DELETE("/:team_id", GinTeamDelete) // Delete user team

	// Member Management
	team.GET("/:team_id/members", GinMemberList)                 // Get user team members
	team.GET("/:team_id/members/:member_id", GinMemberGet)       // Get user team member details
	team.POST("/:team_id/members/direct", GinMemberCreateDirect) // Add member directly (for bots/system)
	team.PUT("/:team_id/members/:member_id", GinMemberUpdate)    // Update user team member
	team.DELETE("/:team_id/members/:member_id", GinMemberDelete) // Remove user team member

	// Member Invitation Management
	team.POST("/:team_id/invitations", GinInvitationCreate)                      // Send team invitation
	team.GET("/:team_id/invitations", GinInvitationList)                         // Get team invitations
	team.GET("/:team_id/invitations/:invitation_id", GinInvitationGet)           // Get invitation details
	team.PUT("/:team_id/invitations/:invitation_id/resend", GinInvitationResend) // Resend invitation
	team.DELETE("/:team_id/invitations/:invitation_id", GinInvitationDelete)     // Cancel invitation
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

	profile.GET("/", placeholder) // Get user profile
	profile.PUT("/", placeholder) // Update user profile
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
