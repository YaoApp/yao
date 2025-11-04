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

	// Create public config without sensitive data (deep copy to avoid modifying global config)
	publicConfig := createPublicEntryConfig(config)

	// Return the entry configuration
	response.RespondWithSuccess(c, response.StatusOK, publicConfig)
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

	// Get entry configuration (GetEntryConfig has default fallback logic)
	config := GetEntryConfig(locale)
	if config == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Entry configuration not found",
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
		verifyResp.Status = EntryVerificationStatusLogin
		response.RespondWithSuccess(c, response.StatusOK, verifyResp)
		return
	}

	// User doesn't exist: generate OTP and send verification code
	verifyResp.Status = EntryVerificationStatusRegister

	// Generate OTP first
	otpID, verificationCode := generateEntryOTP()
	verifyResp.OtpID = otpID
	verifyResp.VerificationSent = true

	// Send verification message asynchronously
	go func() {
		ctx := context.Background()
		err := sendVerificationMessage(ctx, config, usernameType, req.Username, verificationCode, locale)
		if err != nil {
			log.Error("Failed to send verification code to %s: %v", req.Username, err)
			return
		}
		log.Info("Verification code sent to %s for registration (OTP ID: %s)", req.Username, otpID)
	}()

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

// generateEntryOTP generates an OTP code for entry verification
// Returns OTP ID and verification code
func generateEntryOTP() (string, string) {
	otpOption := utilsotp.NewOption()
	otpOption.Length = 6
	otpOption.Type = "numeric"
	otpOption.Expiration = 600 // 10 minutes

	return utilsotp.Generate(otpOption)
}

// sendVerificationMessage sends a verification code message via email or SMS
func sendVerificationMessage(ctx context.Context, config *EntryConfig, usernameType, username, verificationCode, locale string) error {
	// Check if messenger is available
	if messenger.Instance == nil {
		return fmt.Errorf("messenger service not available")
	}

	// Check messenger configuration
	if config.Messenger == nil {
		return fmt.Errorf("messenger configuration not found in entry config")
	}

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
		"to":         username,
		"code":       verificationCode,
		"expires_in": "10", // 10 minutes
		"locale":     locale,
	}

	// Send verification code
	err := messenger.Instance.SendT(ctx, channel, template, templateData, messageType)
	if err != nil {
		return fmt.Errorf("failed to send verification code: %w", err)
	}

	return nil
}

// createPublicEntryConfig creates a deep copy of EntryConfig without sensitive data
// This prevents modifying the global config when removing secrets
func createPublicEntryConfig(config *EntryConfig) *EntryConfig {
	if config == nil {
		return nil
	}

	// Create a new config instance
	publicConfig := &EntryConfig{
		Title:          config.Title,
		Description:    config.Description,
		Default:        config.Default,
		SuccessURL:     config.SuccessURL,
		FailureURL:     config.FailureURL,
		LogoutRedirect: config.LogoutRedirect,
		ClientID:       config.ClientID,
		ClientSecret:   "", // Remove sensitive data
		AutoLogin:      config.AutoLogin,
		Role:           config.Role,
		Type:           config.Type,
		InviteRequired: config.InviteRequired,
	}

	// Deep copy Form config
	if config.Form != nil {
		publicConfig.Form = &FormConfig{
			ForgotPasswordLink: config.Form.ForgotPasswordLink,
			RememberMe:         config.Form.RememberMe,
			RegisterLink:       config.Form.RegisterLink,
			LoginLink:          config.Form.LoginLink,
			TermsOfServiceLink: config.Form.TermsOfServiceLink,
			PrivacyPolicyLink:  config.Form.PrivacyPolicyLink,
		}

		// Deep copy Username config
		if config.Form.Username != nil {
			publicConfig.Form.Username = &UsernameConfig{
				Placeholder: config.Form.Username.Placeholder,
			}
			if config.Form.Username.Fields != nil {
				publicConfig.Form.Username.Fields = make([]string, len(config.Form.Username.Fields))
				copy(publicConfig.Form.Username.Fields, config.Form.Username.Fields)
			}
		}

		// Deep copy Password config
		if config.Form.Password != nil {
			publicConfig.Form.Password = &PasswordConfig{
				Placeholder: config.Form.Password.Placeholder,
			}
		}

		// Deep copy ConfirmPassword config
		if config.Form.ConfirmPassword != nil {
			publicConfig.Form.ConfirmPassword = &PasswordConfig{
				Placeholder: config.Form.ConfirmPassword.Placeholder,
			}
		}

		// Deep copy Captcha config (WITHOUT secret)
		if config.Form.Captcha != nil {
			publicConfig.Form.Captcha = &CaptchaConfig{
				Type: config.Form.Captcha.Type,
			}

			// Deep copy Options, excluding "secret"
			if config.Form.Captcha.Options != nil {
				publicConfig.Form.Captcha.Options = make(map[string]interface{})
				for k, v := range config.Form.Captcha.Options {
					if k != "secret" {
						publicConfig.Form.Captcha.Options[k] = v
					}
				}
			}
		}
	}

	// Deep copy Token config
	if config.Token != nil {
		publicConfig.Token = &TokenConfig{
			ExpiresIn:                       config.Token.ExpiresIn,
			RefreshTokenExpiresIn:           config.Token.RefreshTokenExpiresIn,
			RememberMeExpiresIn:             config.Token.RememberMeExpiresIn,
			RememberMeRefreshTokenExpiresIn: config.Token.RememberMeRefreshTokenExpiresIn,
		}
	}

	// Note: Messenger config is intentionally not copied to public config (backend only)

	// Deep copy ThirdParty config
	if config.ThirdParty != nil {
		publicConfig.ThirdParty = &ThirdParty{}

		if config.ThirdParty.Providers != nil {
			publicConfig.ThirdParty.Providers = make([]*Provider, len(config.ThirdParty.Providers))
			for i, provider := range config.ThirdParty.Providers {
				if provider != nil {
					// Create a copy of provider without sensitive data
					publicConfig.ThirdParty.Providers[i] = &Provider{
						ID:           provider.ID,
						Label:        provider.Label,
						Title:        provider.Title,
						Logo:         provider.Logo,
						Color:        provider.Color,
						TextColor:    provider.TextColor,
						ClientID:     provider.ClientID,
						ResponseMode: provider.ResponseMode,
						// ClientSecret is intentionally omitted for security
						// ClientSecretGenerator is intentionally omitted for security
					}

					// Copy scopes if present
					if provider.Scopes != nil {
						publicConfig.ThirdParty.Providers[i].Scopes = make([]string, len(provider.Scopes))
						copy(publicConfig.ThirdParty.Providers[i].Scopes, provider.Scopes)
					}
				}
			}
		}
	}

	// Deep copy Invite config
	if config.Invite != nil {
		publicConfig.Invite = &InvitePageConfig{
			Title:       config.Invite.Title,
			Description: config.Invite.Description,
			Placeholder: config.Invite.Placeholder,
			ApplyLink:   config.Invite.ApplyLink,
			ApplyPrompt: config.Invite.ApplyPrompt,
			ApplyText:   config.Invite.ApplyText,
		}
	}

	return publicConfig
}

// validatePassword validates password format (8+ characters, must contain letters and numbers, can have special characters)
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasLetter {
		return fmt.Errorf("password must contain at least one letter")
	}

	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}

	return nil
}

// GinSendOTP handles resending OTP verification code
func GinSendOTP(c *gin.Context) {
	// Get authorized info from the temporary token
	authInfo := oauth.GetAuthorizedInfo(c)
	if authInfo == nil || authInfo.Scope != ScopeEntryVerification {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Invalid or missing entry verification token",
		}
		response.RespondWithError(c, response.StatusUnauthorized, errorResp)
		return
	}

	// Get username and username_type from token claims
	username, _ := c.Get("__username")
	usernameType, _ := c.Get("__username_type")

	usernameStr, ok := username.(string)
	if !ok || usernameStr == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Username not found in token",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	usernameTypeStr, ok := usernameType.(string)
	if !ok || usernameTypeStr == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Username type not found in token",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get locale from query parameter
	locale := c.Query("locale")

	// Get entry configuration
	config := GetEntryConfig(locale)
	if config == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Entry configuration not found",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Generate new OTP
	otpID, verificationCode := generateEntryOTP()

	// Send verification message asynchronously
	go func() {
		ctx := context.Background()
		err := sendVerificationMessage(ctx, config, usernameTypeStr, usernameStr, verificationCode, locale)
		if err != nil {
			log.Error("Failed to resend verification code to %s: %v", usernameStr, err)
			return
		}
		log.Info("Verification code resent to %s (OTP ID: %s)", usernameStr, otpID)
	}()

	// Prepare response
	otpResponse := EntrySendOTPResponse{
		OtpID:     otpID,
		ExpiresIn: 600, // 10 minutes
	}

	response.RespondWithSuccess(c, response.StatusOK, otpResponse)
}

// GinEntryRegister handles user registration
func GinEntryRegister(c *gin.Context) {
	// Get authorized info from the temporary token
	authInfo := oauth.GetAuthorizedInfo(c)
	if authInfo == nil || authInfo.Scope != ScopeEntryVerification {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Invalid or missing entry verification token",
		}
		response.RespondWithError(c, response.StatusUnauthorized, errorResp)
		return
	}

	// Parse request body
	var req EntryRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get entry configuration (GetEntryConfig has default fallback logic)
	config := GetEntryConfig(req.Locale)
	if config == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Entry configuration not found",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Validate password format
	if err := validatePassword(req.Password); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Validate confirm password if provided
	if req.ConfirmPassword != "" && req.Password != req.ConfirmPassword {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Password and confirm password do not match",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get username and username_type from token claims
	// These were stored in the temporary token by GinEntryVerify
	username, _ := c.Get("__username")
	usernameType, _ := c.Get("__username_type")

	usernameStr, ok := username.(string)
	if !ok || usernameStr == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Username not found in token",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	usernameTypeStr, ok := usernameType.(string)
	if !ok || usernameTypeStr == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Username type not found in token",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Validate password strength FIRST (pure format check, no external queries)
	if err := validatePassword(req.Password); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Verify verification code (before database queries)
	// This prevents malicious users from using this endpoint to detect existing users
	if config.Messenger != nil {
		// Check if messenger is configured for this username type
		requiresVerification := false
		if usernameTypeStr == "email" && config.Messenger.Mail != nil {
			requiresVerification = true
		} else if usernameTypeStr == "mobile" && config.Messenger.SMS != nil {
			requiresVerification = true
		}

		if requiresVerification {
			if req.OtpID == "" || req.VerificationCode == "" {
				errorResp := &response.ErrorResponse{
					Code:             response.ErrInvalidRequest.Code,
					ErrorDescription: "OTP ID and verification code are required",
				}
				response.RespondWithError(c, response.StatusBadRequest, errorResp)
				return
			}

			// Validate OTP code
			if !utilsotp.Validate(req.OtpID, req.VerificationCode, true) {
				errorResp := &response.ErrorResponse{
					Code:             response.ErrInvalidRequest.Code,
					ErrorDescription: "Invalid or expired verification code",
				}
				response.RespondWithError(c, response.StatusBadRequest, errorResp)
				return
			}
		}
	}

	ctx := c.Request.Context()

	// Check if user already exists (only after OTP verification)
	userExists, _, err := checkUserExists(ctx, usernameTypeStr, usernameStr)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to check user existence: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	if userExists {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "User already exists",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get user provider
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get user provider: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Generate name if not provided
	name := req.Name
	if name == "" {
		switch usernameTypeStr {
		case "email":
			// Extract name from email (part before @)
			if idx := strings.Index(usernameStr, "@"); idx > 0 {
				name = usernameStr[:idx]
			} else {
				name = usernameStr
			}
		case "mobile":
			// Use last 4 digits of phone number
			if len(usernameStr) >= 4 {
				name = "User" + usernameStr[len(usernameStr)-4:]
			} else {
				name = "User" + usernameStr
			}
		}
	}

	// Prepare user data
	userData := map[string]interface{}{
		"name":     name,
		"password": req.Password, // Yao will auto-hash this
		"role_id":  config.Role,
		"type_id":  config.Type,
	}

	// Set email or mobile
	switch usernameTypeStr {
	case "email":
		userData["email"] = usernameStr
		userData["email_verified"] = true // Verified via code
	case "mobile":
		userData["phone_number"] = usernameStr
		userData["phone_number_verified"] = true // Verified via code
	}

	// Determine initial status
	if config.InviteRequired {
		userData["status"] = "pending_invite" // Waiting for invite code verification
	} else {
		userData["status"] = "active"
	}

	// Create user
	userID, err := userProvider.CreateUser(ctx, userData)
	if err != nil {
		log.Error("Failed to create user: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to create user: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	log.Info("User registered successfully: %s (user_id: %s)", usernameStr, userID)

	// If auto_login is false and invite not required, return success without tokens
	if !config.AutoLogin && !config.InviteRequired {
		resp := LoginSuccessResponse{
			UserID:  userID,
			Status:  LoginStatusSuccess,
			Message: "Registration successful. You can now login.",
		}
		response.RespondWithSuccess(c, response.StatusOK, resp)
		return
	}

	// Auto-login or invite_required: Generate tokens using LoginByUserID
	// For invite_required, LoginByUserID will detect pending_invite status and return temporary token
	loginCtx := makeLoginContext(c)
	loginResponse, err := LoginByUserID(userID, loginCtx)
	if err != nil {
		log.Error("Failed to auto-login after registration: %v", err)
		// Still return success for registration, but without tokens
		resp := LoginSuccessResponse{
			UserID:  userID,
			Status:  LoginStatusSuccess,
			Message: "Registration successful, but auto-login failed. Please login manually.",
		}
		response.RespondWithSuccess(c, response.StatusOK, resp)
		return
	}

	// Get session ID
	sid := utils.GetSessionID(c)
	if sid == "" {
		sid = generateSessionID()
	}

	// Handle different login statuses
	switch loginResponse.Status {
	case LoginStatusInviteVerification, LoginStatusMFA, LoginStatusTeamSelection:
		// Return temporary token for next step verification (don't send cookies yet)
		response.RespondWithSuccess(c, response.StatusOK, LoginSuccessResponse{
			UserID:      userID,
			SessionID:   sid,
			Status:      loginResponse.Status,
			AccessToken: loginResponse.AccessToken,
			ExpiresIn:   loginResponse.ExpiresIn,
			MFAEnabled:  loginResponse.MFAEnabled,
			Message:     "Registration successful. Please complete the verification process.",
		})
	case LoginStatusSuccess:
		// Success - send cookies and return full token set
		SendLoginCookies(c, loginResponse, sid)
		response.RespondWithSuccess(c, response.StatusOK, LoginSuccessResponse{
			UserID:                userID,
			SessionID:             sid,
			IDToken:               loginResponse.IDToken,
			AccessToken:           loginResponse.AccessToken,
			RefreshToken:          loginResponse.RefreshToken,
			ExpiresIn:             loginResponse.ExpiresIn,
			RefreshTokenExpiresIn: loginResponse.RefreshTokenExpiresIn,
			MFAEnabled:            loginResponse.MFAEnabled,
			Status:                loginResponse.Status,
			Message:               "Registration and login successful.",
		})
	}
}

// GinEntryLogin handles user login with username and password
func GinEntryLogin(c *gin.Context) {
	// Get authorized info from the temporary token
	authInfo := oauth.GetAuthorizedInfo(c)
	if authInfo == nil || authInfo.Scope != ScopeEntryVerification {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Invalid or missing entry verification token",
		}
		response.RespondWithError(c, response.StatusUnauthorized, errorResp)
		return
	}

	// Parse request body
	var req EntryLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get username and username_type from token claims
	username, _ := c.Get("__username")
	usernameType, _ := c.Get("__username_type")
	userIDFromToken, _ := c.Get("__user_id")

	usernameStr, ok := username.(string)
	if !ok || usernameStr == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Username not found in token",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	usernameTypeStr, ok := usernameType.(string)
	if !ok || usernameTypeStr == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Username type not found in token",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	ctx := c.Request.Context()

	// Get user provider
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get user provider: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get user ID from token or query database
	var userID string
	if userIDFromToken != nil {
		if id, ok := userIDFromToken.(string); ok && id != "" {
			userID = id
		}
	}

	// If user ID not in token, get it from database
	if userID == "" {
		_, userID, err = checkUserExists(ctx, usernameTypeStr, usernameStr)
		if err != nil || userID == "" {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Invalid username or password",
			}
			response.RespondWithError(c, response.StatusUnauthorized, errorResp)
			return
		}
	}

	// Get user auth data (includes password_hash)
	user, err := userProvider.GetUserForAuth(ctx, userID, "user_id")
	if err != nil {
		log.Warn("Failed to get user for auth: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid username or password",
		}
		response.RespondWithError(c, response.StatusUnauthorized, errorResp)
		return
	}

	// Get password hash
	passwordHash, ok := user["password_hash"].(string)
	if !ok || passwordHash == "" {
		log.Warn("User %s has no password hash", userID)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid username or password",
		}
		response.RespondWithError(c, response.StatusUnauthorized, errorResp)
		return
	}

	// Verify password
	valid, err := userProvider.VerifyPassword(ctx, req.Password, passwordHash)
	if err != nil || !valid {
		log.Warn("Password verification failed for user %s", userID)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid username or password",
		}
		response.RespondWithError(c, response.StatusUnauthorized, errorResp)
		return
	}

	// Login using LoginByUserID (all status checks are handled inside)
	loginCtx := makeLoginContext(c)
	loginCtx.RememberMe = req.RememberMe // Set Remember Me from request
	loginResponse, err := LoginByUserID(userID, loginCtx)
	if err != nil {
		log.Error("Failed to login user %s: %v", userID, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Get or generate session ID
	sid := utils.GetSessionID(c)
	if sid == "" {
		sid = generateSessionID()
	}

	// Send login cookies
	SendLoginCookies(c, loginResponse, sid)

	// Handle different login statuses
	switch loginResponse.Status {
	case LoginStatusInviteVerification, LoginStatusMFA, LoginStatusTeamSelection:
		// Return temporary token for next step verification
		response.RespondWithSuccess(c, response.StatusOK, LoginSuccessResponse{
			SessionID:   sid,
			Status:      loginResponse.Status,
			AccessToken: loginResponse.AccessToken,
			ExpiresIn:   loginResponse.ExpiresIn,
			MFAEnabled:  loginResponse.MFAEnabled,
		})
	case LoginStatusSuccess:
		// Success - return full token set
		response.RespondWithSuccess(c, response.StatusOK, LoginSuccessResponse{
			SessionID:             sid,
			IDToken:               loginResponse.IDToken,
			AccessToken:           loginResponse.AccessToken,
			RefreshToken:          loginResponse.RefreshToken,
			ExpiresIn:             loginResponse.ExpiresIn,
			RefreshTokenExpiresIn: loginResponse.RefreshTokenExpiresIn,
			MFAEnabled:            loginResponse.MFAEnabled,
			Status:                loginResponse.Status,
		})
	}
}

// GinVerifyInvite is the handler for verifying and redeeming invitation code
// This endpoint is called with the temporary access token (scope: invite_verification)
// after user registration when invite is required
func GinVerifyInvite(c *gin.Context) {
	// Parse request body
	var req struct {
		InvitationCode string `json:"invitation_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get authorized info from the temporary token
	authInfo := oauth.GetAuthorizedInfo(c)
	if authInfo == nil || authInfo.Scope != ScopeEntryVerification {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInsufficientScope.Code,
			ErrorDescription: "Invalid or missing token scope. Expected entry_verification scope",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Get user ID from auth info
	userID := authInfo.UserID
	if userID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidToken.Code,
			ErrorDescription: "User ID not found in token",
		}
		response.RespondWithError(c, response.StatusUnauthorized, errorResp)
		return
	}

	// Get user provider
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		log.Error("Failed to get user provider: %s", err.Error())
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Internal server error",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Redeem invitation code
	err = userProvider.UseInvitationCode(ctx, req.InvitationCode, userID)
	if err != nil {
		log.Error("Failed to use invitation code: %s", err.Error())
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Failed to verify invitation code: %s", err.Error()),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Update user status to active
	err = userProvider.UpdateUserStatus(ctx, userID, "active")
	if err != nil {
		log.Error("Failed to update user status: %s", err.Error())
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to activate user account",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Generate login context
	loginCtx := makeLoginContext(c)

	// Preserve Remember Me state from temporary token (authInfo is already available from above)
	loginCtx.RememberMe = authInfo.RememberMe

	// Generate full login token
	loginResponse, err := LoginByUserID(userID, loginCtx)
	if err != nil {
		log.Error("Failed to generate login token: %s", err.Error())
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to generate login credentials",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get or create session ID
	sid := utils.GetSessionID(c)
	if sid == "" {
		sid = generateSessionID()
	}

	// Send login cookies
	SendLoginCookies(c, loginResponse, sid)

	// Handle different login statuses (in case MFA is enabled or team selection needed)
	switch loginResponse.Status {
	case LoginStatusMFA, LoginStatusTeamSelection:
		// Return temporary token for next step verification
		response.RespondWithSuccess(c, response.StatusOK, LoginSuccessResponse{
			SessionID:   sid,
			AccessToken: loginResponse.AccessToken,
			ExpiresIn:   loginResponse.ExpiresIn,
			MFAEnabled:  loginResponse.MFAEnabled,
			Status:      loginResponse.Status,
		})
	default:
		// Success - return full token set
		response.RespondWithSuccess(c, response.StatusOK, LoginSuccessResponse{
			SessionID:             sid,
			IDToken:               loginResponse.IDToken,
			AccessToken:           loginResponse.AccessToken,
			RefreshToken:          loginResponse.RefreshToken,
			ExpiresIn:             loginResponse.ExpiresIn,
			RefreshTokenExpiresIn: loginResponse.RefreshTokenExpiresIn,
			MFAEnabled:            loginResponse.MFAEnabled,
			Status:                loginResponse.Status,
		})
	}
}
