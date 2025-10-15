package user

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/messenger"
	messengertypes "github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
	utilscaptcha "github.com/yaoapp/yao/utils/captcha"
	utilsotp "github.com/yaoapp/yao/utils/otp"
)

// getEntryConfig is the handler for get unified auth entry configuration
func getEntryConfig(c *gin.Context) {
	// Get locale from query parameter (optional)
	locale := c.Query("locale")

	// Get entry configuration for the specified locale
	config := GetEntryConfig(locale)

	// Set session id if not exists
	sid := utils.GetSessionID(c)
	if sid == "" {
		sid = generateSessionID()
		response.SendSessionCookie(c, sid)
	}

	// If no configuration found, return error
	if config == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "No entry configuration found for the requested locale",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Create public config without sensitive data
	publicConfig := *config
	publicConfig.ClientSecret = "" // Remove sensitive data

	// Remove captcha secret from public config
	if publicConfig.Form != nil && publicConfig.Form.Captcha != nil && publicConfig.Form.Captcha.Options != nil {
		// Create a copy of captcha options without the secret
		captchaOptions := make(map[string]interface{})
		for k, v := range publicConfig.Form.Captcha.Options {
			if k != "secret" {
				captchaOptions[k] = v
			}
		}
		publicConfig.Form.Captcha.Options = captchaOptions
	}

	// Return the entry configuration
	response.RespondWithSuccess(c, response.StatusOK, publicConfig)
}

// entry is the handler for unified auth entry (login/register)
// The backend determines whether this is a login or registration based on email existence
func entry(c *gin.Context) {
	// This is a placeholder - you may need to implement the actual login/register logic here
	// The logic should:
	// 1. Check if the email exists in the database
	// 2. If exists: proceed with login flow
	// 3. If not exists: proceed with registration flow
}

// GinEntryVerify is the handler for verifying entry (login/register)
// It checks if the username exists and sends verification code if needed
func GinEntryVerify(c *gin.Context) {
	// Parse request body
	var req EntryVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get locale from request or query parameter
	locale := req.Locale
	if locale == "" {
		locale = c.Query("locale")
	}
	if locale == "" {
		locale = "en" // Default locale
	}

	// Determine username type (email or mobile) - check this first before expensive operations
	usernameType := determineUsernameType(req.Username)
	if usernameType == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid username format: must be email or mobile number",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get entry configuration
	config := GetEntryConfig(locale)
	if config == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Entry configuration not found for locale: " + locale,
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Verify captcha
	if config.Form != nil && config.Form.Captcha != nil {
		err := verifyCaptcha(config.Form.Captcha, req.CaptchaID, req.Captcha)
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Captcha verification failed: " + err.Error(),
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
			return
		}
	}

	// Check if user exists
	userExists, userID, err := checkUserExists(c.Request.Context(), usernameType, req.Username)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to check user existence: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get Yao client config for token generation
	yaoClientConfig := GetYaoClientConfig()
	if yaoClientConfig == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Client configuration not found",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Generate temporary access token for entry verification (valid for 10 minutes)
	var tokenExpire int = 10 * 60 // 10 minutes

	// Create subject based on username (temporary subject for verification)
	tempSubject := fmt.Sprintf("entry:%s:%s", usernameType, req.Username)

	// Extra claims for the token
	extraClaims := map[string]interface{}{
		"username":      req.Username,
		"username_type": usernameType,
	}

	// If user exists, add user_id to claims
	if userExists && userID != "" {
		extraClaims["user_id"] = userID
	}

	accessToken, err := oauth.OAuth.MakeAccessToken(yaoClientConfig.ClientID, ScopeEntryVerification, tempSubject, tokenExpire, extraClaims)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to generate access token: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Prepare response
	verifyResp := EntryVerifyResponse{
		AccessToken: accessToken,
		ExpiresIn:   tokenExpire,
		TokenType:   "Bearer",
		Scope:       ScopeEntryVerification,
		UserExists:  userExists,
	}

	// If user exists: return login status
	if userExists {
		verifyResp.Status = "login"
		response.RespondWithSuccess(c, response.StatusOK, verifyResp)
		return
	}

	// User doesn't exist: send verification code and return register status
	verifyResp.Status = "register"

	// Send verification code asynchronously
	go func() {
		ctx := context.Background()
		err := sendEntryVerificationCode(ctx, config, usernameType, req.Username, locale)
		if err != nil {
			log.Error("Failed to send verification code to %s: %v", req.Username, err)
		} else {
			log.Info("Verification code sent to %s for registration", req.Username)
		}
	}()

	verifyResp.VerificationSent = true
	response.RespondWithSuccess(c, response.StatusOK, verifyResp)
}

// Helper Functions

// verifyCaptcha verifies the captcha based on type (image or turnstile)
func verifyCaptcha(captchaConfig *CaptchaConfig, captchaID, captcha string) error {
	if captchaConfig == nil {
		return nil // No captcha required
	}

	switch captchaConfig.Type {
	case "image":
		// Verify image captcha
		if captchaID == "" || captcha == "" {
			return fmt.Errorf("captcha_id and captcha are required for image captcha")
		}

		valid := utilscaptcha.Validate(captchaID, captcha)
		if !valid {
			return fmt.Errorf("invalid captcha")
		}
		return nil

	case "turnstile":
		// Verify Cloudflare Turnstile
		if captcha == "" {
			return fmt.Errorf("captcha token is required for Turnstile")
		}

		// Get secret from options
		secret := ""
		if captchaConfig.Options != nil {
			if s, ok := captchaConfig.Options["secret"].(string); ok {
				secret = s
			}
		}

		if secret == "" {
			return fmt.Errorf("Turnstile secret not configured")
		}

		// Verify Turnstile token using captcha function
		valid := utilscaptcha.ValidateCloudflare(captcha, secret)
		if !valid {
			return fmt.Errorf("invalid Turnstile token")
		}
		return nil

	default:
		return fmt.Errorf("unsupported captcha type: %s", captchaConfig.Type)
	}
}

// determineUsernameType determines if the username is email or mobile
func determineUsernameType(username string) string {
	// Check if it's an email
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if emailRegex.MatchString(username) {
		return "email"
	}

	// Check if it's a mobile number (international format)
	// Support formats like: +86123456789, 86123456789, 123456789
	mobileRegex := regexp.MustCompile(`^\+?[0-9]{10,15}$`)
	if mobileRegex.MatchString(username) {
		return "mobile"
	}

	return ""
}

// checkUserExists checks if a user exists with the given email or mobile
// Returns: (userExists bool, userID string, error)
func checkUserExists(ctx context.Context, usernameType, username string) (bool, string, error) {
	// Get user provider
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		return false, "", fmt.Errorf("failed to get user provider: %w", err)
	}

	// Query user by email or mobile
	var user map[string]interface{}
	switch usernameType {
	case "email":
		user, err = userProvider.GetUserByEmail(ctx, username)
	case "mobile":
		// For mobile, use GetUserForAuth with phone_number identifier type
		user, err = userProvider.GetUserForAuth(ctx, username, "phone_number")
	default:
		return false, "", fmt.Errorf("invalid username type: %s", usernameType)
	}

	if err != nil {
		// If user not found, return false without error
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "User not found") {
			return false, "", nil
		}
		return false, "", fmt.Errorf("failed to query user: %w", err)
	}

	// Extract user_id from the returned map
	userID := ""
	if user != nil {
		if id, ok := user["user_id"].(string); ok {
			userID = id
		} else if id, ok := user["id"].(string); ok {
			userID = id
		}
	}

	if userID == "" {
		return false, "", nil
	}

	return true, userID, nil
}

// sendEntryVerificationCode sends a verification code to the user's email or mobile
func sendEntryVerificationCode(ctx context.Context, config *EntryConfig, usernameType, username, locale string) error {
	// Check if messenger is available
	if messenger.Instance == nil {
		return fmt.Errorf("messenger service not available")
	}

	// Check messenger configuration
	if config.Messenger == nil {
		return fmt.Errorf("messenger configuration not found in entry config")
	}

	// Generate verification code using OTP (6-digit number, 10 minutes expiry)
	otpOption := utilsotp.NewOption()
	otpOption.Length = 6
	otpOption.Type = "numeric"
	otpOption.Expiration = 600 // 10 minutes

	otpID, verificationCode := utilsotp.Generate(otpOption)

	// Store OTP ID in context for later verification
	// The OTP code is automatically stored in memory with expiration
	log.Debug("Generated OTP for %s: ID=%s", username, otpID)

	var channel string
	var template string
	var messageType messengertypes.MessageType

	// Determine channel and template based on username type
	switch usernameType {
	case "email":
		if config.Messenger.Mail == nil {
			return fmt.Errorf("email messenger configuration not found")
		}
		channel = config.Messenger.Mail.Channel
		template = config.Messenger.Mail.Template
		messageType = messengertypes.MessageTypeEmail

		// Default channel if not specified
		if channel == "" {
			channel = "default"
		}

	case "mobile":
		if config.Messenger.SMS == nil {
			return fmt.Errorf("SMS messenger configuration not found")
		}
		channel = config.Messenger.SMS.Channel
		template = config.Messenger.SMS.Template
		messageType = messengertypes.MessageTypeSMS

		// Default channel if not specified
		if channel == "" {
			channel = "default"
		}

	default:
		return fmt.Errorf("unsupported username type: %s", usernameType)
	}

	if template == "" {
		return fmt.Errorf("template not configured for %s verification", usernameType)
	}

	// Prepare template data
	templateData := messengertypes.TemplateData{
		"to":                username,
		"verification_code": verificationCode,
		"expires_in":        "10", // 10 minutes
		"locale":            locale,
	}

	// Send verification code
	err := messenger.Instance.SendT(ctx, channel, template, templateData, messageType)
	if err != nil {
		return fmt.Errorf("failed to send verification code: %w", err)
	}

	return nil
}
