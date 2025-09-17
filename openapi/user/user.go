package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Attach attaches the signin handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// User Authentication
	group.GET("/login", placeholder)                // Get login page config (public)
	group.POST("/login", placeholder)               // User login (public)
	group.POST("/register", placeholder)            // User register (public)
	group.POST("/logout", oauth.Guard, placeholder) // User logout

	// Logined User Settings
	attachProfile(group, oauth)      // User profile management
	attachPreferences(group, oauth)  // User preferences management
	attachAccount(group, oauth)      // Account settings
	attachThirdParty(group, oauth)   // Third party login
	attachMFA(group, oauth)          // MFA settings
	attachBalance(group, oauth)      // User balance management
	attachSubscription(group, oauth) // User subscription management
	attachAPIKeys(group, oauth)      // User API keys management
	attachUsage(group, oauth)        // User usage management
	attachBilling(group, oauth)      // User billing management
	attachInvite(group, oauth)       // User invite management
	attachTeam(group, oauth)         // User team management
	attachPrivacy(group, oauth)      // User privacy management

	// User Management
	attachUsers(group, oauth)
}

// User Team Management
func attachTeam(group *gin.RouterGroup, oauth types.OAuth) {
	team := group.Group("/teams")
	team.Use(oauth.Guard)
	team.GET("/", placeholder)            // Get user teams
	team.GET("/:team_id", placeholder)    // Get user team details
	team.POST("/", placeholder)           // Create user team
	team.PUT("/:team_id", placeholder)    // Update user team
	team.DELETE("/:team_id", placeholder) // Delete user team

	// Member Management
	team.GET("/:team_id/members", placeholder)               // Get user team members
	team.GET("/:team_id/members/:member_id", placeholder)    // Get user team member details
	team.POST("/:team_id/members/:type", placeholder)        // Create user team member
	team.PUT("/:team_id/members/:member_id", placeholder)    // Update user team member
	team.DELETE("/:team_id/members/:member_id", placeholder) // Remove user team member
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
	billing.PUT("/history", placeholder) // Update user billing history
}

// Invite Management
func attachInvite(group *gin.RouterGroup, oauth types.OAuth) {
	invite := group.Group("/invite")
	invite.Use(oauth.Guard)

	invite.GET("/code", placeholder)        // Get user invite code
	invite.GET("/statistics", placeholder)  // Get user invite statistics
	invite.GET("/history", placeholder)     // Get user invite history
	invite.GET("/commissions", placeholder) // Get user invite commissions
}

// User Balance Management
func attachBalance(group *gin.RouterGroup, oauth types.OAuth) {
	balance := group.Group("/balance")
	balance.Use(oauth.Guard)

	balance.GET("/", placeholder)        // Get user balance info
	balance.GET("/history", placeholder) // Get balance change history

	// Top-up Management
	topup := balance.Group("/topup")
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
	mfa := group.Group("/2fa")
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

	thirdParty.GET("/providers/available", placeholder)             // Get available OAuth providers
	thirdParty.GET("/:provider/authorize", placeholder)             // Get OAuth authorization URL
	thirdParty.POST("/:provider/connect", oauth.Guard, placeholder) // Connect OAuth provider
	thirdParty.POST("/:provider/authorize/prepare", placeholder)    // Get OAuth authorization URL
	thirdParty.POST("/:provider/callback", placeholder)             // Handle OAuth callback

}

func placeholder(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
}
