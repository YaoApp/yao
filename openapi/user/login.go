package user

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/kb"
	kbapi "github.com/yaoapp/yao/kb/api"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/providers/user"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
	"github.com/yaoapp/yao/utils/captcha"
)

// kbCollectionCreating tracks collections currently being created to avoid duplicate creation
var kbCollectionCreating sync.Map

// getCaptcha is the handler for get captcha image for entry (login/register)
func getCaptcha(c *gin.Context) {
	var option captcha.Option = captcha.NewOption()

	err := c.ShouldBindQuery(&option)
	if err != nil {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		})
		return
	}

	// Set the type to image
	option.Type = "image"
	id, content := captcha.Generate(option)

	// Return in the format expected by the frontend
	response.RespondWithSuccess(c, http.StatusOK, gin.H{
		"captcha_id":    id,
		"captcha_image": content,
		"expires_in":    300, // 5 minutes
	})
}

// LoginThirdParty is the handler for third party login
func LoginThirdParty(providerID string, userinfo *oauthtypes.OIDCUserInfo, loginCtx *LoginContext, locale string) (*LoginResponse, error) {

	// Get provider
	provider, err := GetProvider(providerID)
	if err != nil {
		return nil, err
	}

	// Get entry configuration for role and type
	entryConfig := GetEntryConfig(locale)
	if entryConfig == nil {
		// If no entry config found, try to get default entry config
		log.Warn("Entry configuration not found for locale '%s', trying default locale 'en'", locale)
		entryConfig = GetEntryConfig("en")
		if entryConfig == nil {
			return nil, fmt.Errorf("entry configuration not found. Please create entry config files in openapi/user/entry/")
		}
	}

	// Check if user exists
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		return nil, err
	}

	// Auto register user if not exists
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var userID string

	// Auto register user if not exists
	if provider.Register != nil && provider.Register.Auto {
		userID, err = userProvider.GetOAuthUserID(ctx, providerID, userinfo.Sub)
		if err != nil && err.Error() == user.ErrOAuthAccountNotFound {

			// Determine initial status based on invite requirement
			status := "active"
			if entryConfig.InviteRequired {
				status = "pending_invite" // Waiting for invite code verification
			}

			userData := map[string]interface{}{
				"name":        userinfo.Name,
				"given_name":  userinfo.GivenName,
				"family_name": userinfo.FamilyName,
				"picture":     userinfo.Picture,
				"role_id":     entryConfig.Role,
				"type_id":     entryConfig.Type,
				"status":      status,
			}

			// Auto register user
			userID, err = userProvider.CreateUser(ctx, userData)
			if err != nil {
				return nil, err
			}

			// Create OAuth account
			userData = userinfo.Map()
			userData["provider"] = providerID
			_, err = userProvider.CreateOAuthAccount(ctx, userID, userData)
			if err != nil {
				return nil, err
			}
		}
	}

	// Get User ID from OAuth account
	userID, err = userProvider.GetOAuthUserID(ctx, providerID, userinfo.Sub)
	if err != nil {
		return nil, err
	}

	return LoginByUserID(userID, loginCtx)
}

// LoginByUserID is the handler for login by user ID
func LoginByUserID(userid string, loginCtx *LoginContext) (*LoginResponse, error) {
	// Get User
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		return nil, err
	}

	// Get User
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	user, err := userProvider.GetUserWithScopes(ctx, userid)
	if err != nil {
		return nil, err
	}

	yaoClientConfig := GetYaoClientConfig()
	var scopes []string = yaoClientConfig.Scopes
	if v, ok := user["scopes"].([]string); ok {
		scopes = v
	}

	subject, err := oauth.OAuth.Subject(yaoClientConfig.ClientID, userid)
	if err != nil {
		log.Warn("Failed to store user fingerprint: %s", err.Error())
	}

	// Check user status first - handle all non-active statuses
	status, _ := user["status"].(string)
	switch status {
	case "pending":
		return nil, fmt.Errorf("account is pending activation. Please contact administrator")
	case "email_unverified":
		return nil, fmt.Errorf("email is not verified. Please verify your email address")
	case "disabled":
		return nil, fmt.Errorf("account is disabled. Please contact administrator")
	case "suspended":
		return nil, fmt.Errorf("account is suspended. Please contact administrator")
	case "locked":
		return nil, fmt.Errorf("account is locked. Please contact administrator")
	case "archived":
		return nil, fmt.Errorf("account is archived. Please contact administrator")
	case "password_expired":
		return nil, fmt.Errorf("password has expired. Please reset your password")
	case "pending_invite":
		// User needs to verify invitation code, generate temporary token
		var inviteExpire int = 10 * 60 // 10 minutes

		// Prepare extra claims to preserve Remember Me state
		extraClaims := make(map[string]interface{})
		if loginCtx != nil && loginCtx.RememberMe {
			extraClaims["remember_me"] = true
		}

		accessToken, err := oauth.OAuth.MakeAccessToken(yaoClientConfig.ClientID, ScopeEntryVerification, subject, inviteExpire, extraClaims)
		if err != nil {
			return nil, err
		}

		return &LoginResponse{
			UserID:      userid,
			AccessToken: accessToken,
			ExpiresIn:   inviteExpire,
			TokenType:   "Bearer",
			Scope:       ScopeEntryVerification,
			Status:      LoginStatusInviteVerification,
		}, nil
	case "active":
		// Continue with normal login flow
	default:
		return nil, fmt.Errorf("account status is invalid: %s", status)
	}

	// Get MFA enabled status from user data
	mfaEnabled := utils.ToBool(user["mfa_enabled"])

	// If MFA enabled, generate MFA token
	if mfaEnabled {
		// Sign temporary access token for MFA
		var mfaExpire int = 10 * 60 // 10 minutes

		// Prepare extra claims to preserve Remember Me state
		extraClaims := make(map[string]interface{})
		if loginCtx != nil && loginCtx.RememberMe {
			extraClaims["remember_me"] = true
		}

		accessToken, err := oauth.OAuth.MakeAccessToken(yaoClientConfig.ClientID, ScopeMFAVerification, subject, mfaExpire, extraClaims)
		if err != nil {
			return nil, err
		}

		return &LoginResponse{
			UserID:      userid,
			AccessToken: accessToken,
			ExpiresIn:   mfaExpire,
			MFAEnabled:  mfaEnabled,
			TokenType:   "Bearer",
			Scope:       ScopeMFAVerification,
			Status:      LoginStatusMFA,
		}, nil
	}

	// Update Last Login
	if loginCtx != nil {
		err = userProvider.UpdateUserLastLogin(ctx, userid, loginCtx)
		if err != nil {
			log.Warn("Failed to update last login: %s", err.Error())
		}
	}

	// Count User Teams
	numTeams, err := getUserTeamsCount(ctx, userid)
	if err != nil {
		return nil, err
	}

	// If user has teams, return team selection status with temporary access token
	if numTeams > 0 {
		// Sign temporary access token for Team Selection
		var teamSelectionExpire int = 10 * 60 // 10 minutes

		// Prepare extra claims to preserve Remember Me state
		extraClaims := make(map[string]interface{})
		if loginCtx != nil && loginCtx.RememberMe {
			extraClaims["remember_me"] = true
		}

		accessToken, err := oauth.OAuth.MakeAccessToken(yaoClientConfig.ClientID, ScopeTeamSelection, subject, teamSelectionExpire, extraClaims)
		if err != nil {
			return nil, err
		}

		return &LoginResponse{
			UserID:      userid,
			Subject:     subject,
			AccessToken: accessToken,
			ExpiresIn:   teamSelectionExpire,
			MFAEnabled:  mfaEnabled,
			TokenType:   "Bearer",
			Scope:       ScopeTeamSelection,
			Status:      LoginStatusTeamSelection,
		}, nil
	}

	// Issue tokens without team context
	resp, err := issueTokens(ctx, &IssueTokensParams{
		UserID:   userid,
		TeamID:   "",
		Team:     nil,
		Member:   nil,
		User:     user,
		Subject:  subject,
		Scopes:   scopes,
		LoginCtx: loginCtx,
	})
	if err != nil {
		return nil, err
	}

	// Initialize KB collection asynchronously after successful login
	locale := ""
	if loginCtx != nil {
		locale = loginCtx.Locale
	}
	go prepareUserKBCollection(userid, "", locale)

	return resp, nil
}

// LoginByTeamID is the handler for login by team ID (after team selection)
func LoginByTeamID(userid string, teamID string, loginCtx *LoginContext) (*LoginResponse, error) {
	// Get User
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get user data with scopes
	user, err := userProvider.GetUserWithScopes(ctx, userid)
	if err != nil {
		return nil, err
	}

	yaoClientConfig := GetYaoClientConfig()
	var scopes []string = yaoClientConfig.Scopes
	if v, ok := user["scopes"].([]string); ok {
		scopes = v
	}

	// Get or create subject
	subject, err := oauth.OAuth.Subject(yaoClientConfig.ClientID, userid)
	if err != nil {
		log.Warn("Failed to store user fingerprint: %s", err.Error())
	}

	// Handle personal account (no team)
	if teamID == "" || teamID == "personal" {
		resp, err := issueTokens(ctx, &IssueTokensParams{
			UserID:   userid,
			TeamID:   "",
			Team:     nil,
			Member:   nil,
			User:     user,
			Subject:  subject,
			Scopes:   scopes,
			LoginCtx: loginCtx,
		})
		if err != nil {
			return nil, err
		}

		// Initialize KB collection asynchronously after successful login
		locale := ""
		if loginCtx != nil {
			locale = loginCtx.Locale
		}
		go prepareUserKBCollection(userid, "", locale)

		return resp, nil
	}

	// Verify user is a member of the team and get team details
	team, err := userProvider.GetTeamByMember(ctx, teamID, userid)
	if err != nil {
		return nil, fmt.Errorf("access denied: you are not a member of this team")
	}

	// Get member profile information for team context
	member, err := userProvider.GetMember(ctx, teamID, userid)
	if err != nil {
		log.Warn("Failed to get member profile: %s", err.Error())
		// Continue without member profile if it fails
		member = nil
	}

	// Update Last Login
	if loginCtx != nil {
		err = userProvider.UpdateUserLastLogin(ctx, userid, loginCtx)
		if err != nil {
			log.Warn("Failed to update last login: %s", err.Error())
		}
	}

	// Issue tokens with team context and member profile
	resp, err := issueTokens(ctx, &IssueTokensParams{
		UserID:   userid,
		TeamID:   teamID,
		Team:     team,
		Member:   member,
		User:     user,
		Subject:  subject,
		Scopes:   scopes,
		LoginCtx: loginCtx,
	})
	if err != nil {
		return nil, err
	}

	// Initialize KB collection asynchronously after successful login
	locale := ""
	if loginCtx != nil {
		locale = loginCtx.Locale
	}
	go prepareUserKBCollection(userid, teamID, locale)

	return resp, nil
}

// issueTokens is the core function that issues all necessary tokens (ID token, access token, refresh token)
func issueTokens(ctx context.Context, params *IssueTokensParams) (*LoginResponse, error) {
	yaoClientConfig := GetYaoClientConfig()

	// Determine token expiration times based on Remember Me setting
	var expiresIn, refreshTokenExpiresIn int

	// Try to get token config from entry config first
	locale := ""
	entryConfig := GetEntryConfig(locale)

	if params.LoginCtx != nil && params.LoginCtx.RememberMe {
		// Remember Me mode: use extended token durations
		if entryConfig != nil && entryConfig.Token != nil {
			// Parse Remember Me access token expires_in
			if entryConfig.Token.RememberMeExpiresIn != "" {
				normalized, err := normalizeDuration(entryConfig.Token.RememberMeExpiresIn)
				if err != nil {
					log.Warn("Failed to parse remember_me_expires_in: %s, using default", err.Error())
				} else {
					duration, err := time.ParseDuration(normalized)
					if err == nil {
						expiresIn = int(duration.Seconds())
					}
				}
			}

			// Parse Remember Me refresh token expires_in
			if entryConfig.Token.RememberMeRefreshTokenExpiresIn != "" {
				normalized, err := normalizeDuration(entryConfig.Token.RememberMeRefreshTokenExpiresIn)
				if err != nil {
					log.Warn("Failed to parse remember_me_refresh_token_expires_in: %s, using default", err.Error())
				} else {
					duration, err := time.ParseDuration(normalized)
					if err == nil {
						refreshTokenExpiresIn = int(duration.Seconds())
					}
				}
			}

			// If refresh token not configured, default to 2x the access token duration
			if refreshTokenExpiresIn == 0 && expiresIn > 0 {
				refreshTokenExpiresIn = expiresIn * 2
			}
		}
	} else {
		// Normal login: use standard token durations from entry config
		if entryConfig != nil && entryConfig.Token != nil {
			// Parse access token expires_in
			if entryConfig.Token.ExpiresIn != "" {
				normalized, err := normalizeDuration(entryConfig.Token.ExpiresIn)
				if err != nil {
					log.Warn("Failed to parse expires_in: %s, using default", err.Error())
				} else {
					duration, err := time.ParseDuration(normalized)
					if err == nil {
						expiresIn = int(duration.Seconds())
					}
				}
			}

			// Parse refresh token expires_in
			if entryConfig.Token.RefreshTokenExpiresIn != "" {
				normalized, err := normalizeDuration(entryConfig.Token.RefreshTokenExpiresIn)
				if err != nil {
					log.Warn("Failed to parse refresh_token_expires_in: %s, using default", err.Error())
				} else {
					duration, err := time.ParseDuration(normalized)
					if err == nil {
						refreshTokenExpiresIn = int(duration.Seconds())
					}
				}
			}

			// If refresh token not configured, default to 24x the access token duration
			if refreshTokenExpiresIn == 0 && expiresIn > 0 {
				refreshTokenExpiresIn = expiresIn * 24
			}
		}
	}

	// Fall back to YaoClientConfig defaults if not set from entry config
	if expiresIn == 0 {
		expiresIn = yaoClientConfig.ExpiresIn
	}
	if refreshTokenExpiresIn == 0 {
		refreshTokenExpiresIn = yaoClientConfig.RefreshTokenExpiresIn
	}

	// Prepare OIDC user info
	oidcUserInfo := oauthtypes.MakeOIDCUserInfo(params.User)
	oidcUserInfo.Sub = params.Subject
	oidcUserInfo.YaoUserID = params.UserID // Add original user ID

	// Prepare extra claims for access token
	extraClaims := make(map[string]interface{})

	// Add team context if available
	if params.TeamID != "" && params.Team != nil {
		extraClaims["team_id"] = params.TeamID

		// Add tenant_id if available from the team
		if tenantID := utils.ToString(params.Team["tenant_id"]); tenantID != "" {
			extraClaims["tenant_id"] = tenantID
			oidcUserInfo.YaoTenantID = tenantID
		}

		// Add team info to OIDC user info
		oidcUserInfo.YaoTeamID = params.TeamID
		teamInfo := &oauthtypes.OIDCTeamInfo{}
		if teamIDVal := utils.ToString(params.Team["team_id"]); teamIDVal != "" {
			teamInfo.TeamID = teamIDVal
		}
		if logo := utils.ToString(params.Team["logo"]); logo != "" {
			teamInfo.Logo = logo
		}
		if name := utils.ToString(params.Team["name"]); name != "" {
			teamInfo.Name = name
		}
		if description := utils.ToString(params.Team["description"]); description != "" {
			teamInfo.Description = description
		}

		// Add owner_id if available from the team (only check once)
		if ownerID := utils.ToString(params.Team["owner_id"]); ownerID != "" {
			extraClaims["owner_id"] = ownerID
			teamInfo.OwnerID = ownerID

			// Check if user is owner
			if ownerID == params.UserID {
				isOwner := true
				oidcUserInfo.YaoIsOwner = &isOwner
			}
		}

		oidcUserInfo.YaoTeam = teamInfo

		// Add member profile information if available
		if params.Member != nil {
			memberInfo := &oauthtypes.OIDCMemberInfo{}
			if memberID := utils.ToString(params.Member["member_id"]); memberID != "" {
				memberInfo.MemberID = memberID
			}
			if displayName := utils.ToString(params.Member["display_name"]); displayName != "" {
				memberInfo.DisplayName = displayName
			}
			if bio := utils.ToString(params.Member["bio"]); bio != "" {
				memberInfo.Bio = bio
			}
			if avatar := utils.ToString(params.Member["avatar"]); avatar != "" {
				memberInfo.Avatar = avatar
			}
			if email := utils.ToString(params.Member["email"]); email != "" {
				memberInfo.Email = email
			}
			oidcUserInfo.YaoMember = memberInfo
		}
	}

	// Add type information (use team type if in team context, otherwise use user type)
	var typeID string
	if params.TeamID != "" && params.Team != nil {
		// Team context - use team's type
		typeID = utils.ToString(params.Team["type_id"])
	} else {
		// Personal context - use user's type
		typeID = utils.ToString(params.User["type_id"])
	}

	if typeID != "" {
		// Add type_id to extra claims for access token
		extraClaims["type_id"] = typeID
		oidcUserInfo.YaoTypeID = typeID

		// Get type details
		userProvider, err := oauth.OAuth.GetUserProvider()
		if err == nil {
			typeInfo, err := userProvider.GetType(ctx, typeID)
			if err == nil && typeInfo != nil {
				// Add type info to OIDC user info
				typeDetails := &oauthtypes.OIDCTypeInfo{}
				if typeIDVal := utils.ToString(typeInfo["type_id"]); typeIDVal != "" {
					typeDetails.TypeID = typeIDVal
				}
				if name := utils.ToString(typeInfo["name"]); name != "" {
					typeDetails.Name = name
				}
				if locale := utils.ToString(typeInfo["locale"]); locale != "" {
					typeDetails.Locale = locale
				}
				oidcUserInfo.YaoType = typeDetails
			}
		}
	}

	// Sign OIDC Token
	var oidcToken string
	var err error
	if len(extraClaims) > 0 {
		oidcToken, err = oauth.OAuth.SignIDToken(yaoClientConfig.ClientID, strings.Join(params.Scopes, " "), expiresIn, oidcUserInfo, extraClaims)
	} else {
		oidcToken, err = oauth.OAuth.SignIDToken(yaoClientConfig.ClientID, strings.Join(params.Scopes, " "), expiresIn, oidcUserInfo)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to sign OIDC token: %w", err)
	}

	// Sign Access Token
	var accessToken string
	if len(extraClaims) > 0 {
		accessToken, err = oauth.OAuth.MakeAccessToken(yaoClientConfig.ClientID, strings.Join(params.Scopes, " "), params.Subject, expiresIn, extraClaims)
	} else {
		accessToken, err = oauth.OAuth.MakeAccessToken(yaoClientConfig.ClientID, strings.Join(params.Scopes, " "), params.Subject, expiresIn)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Sign Refresh Token
	var refreshToken string
	if len(extraClaims) > 0 {
		refreshToken, err = oauth.OAuth.MakeRefreshToken(yaoClientConfig.ClientID, strings.Join(params.Scopes, " "), params.Subject, refreshTokenExpiresIn, extraClaims)
	} else {
		refreshToken, err = oauth.OAuth.MakeRefreshToken(yaoClientConfig.ClientID, strings.Join(params.Scopes, " "), params.Subject, refreshTokenExpiresIn)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &LoginResponse{
		UserID:                params.UserID,
		Subject:               params.Subject,
		AccessToken:           accessToken,
		IDToken:               oidcToken,
		RefreshToken:          refreshToken,
		ExpiresIn:             expiresIn,
		RefreshTokenExpiresIn: refreshTokenExpiresIn,
		TokenType:             "Bearer",
		MFAEnabled:            utils.ToBool(params.User["mfa_enabled"]),
		Scope:                 strings.Join(params.Scopes, " "),
		Status:                LoginStatusSuccess,
	}, nil
}

// prepareUserKBCollection prepares KB collection for user (called asynchronously after login)
func prepareUserKBCollection(userID, teamID, locale string) {
	// Get global KB setting
	kbSetting := assistant.GetGlobalKBSetting()
	if kbSetting == nil || kbSetting.Chat == nil {
		return // No KB configuration for chat, skip
	}

	// Check if KB API is initialized
	if kb.API == nil {
		log.Warn("KB API not initialized, skipping KB collection preparation")
		return
	}

	chatKB := kbSetting.Chat

	// Get KB collection ID for this user
	// Same team + user always produces the same ID (idempotent)
	collectionID := assistant.GetChatKBID(teamID, userID)

	// Check if this collection is currently being created by another goroutine
	if _, isCreating := kbCollectionCreating.LoadOrStore(collectionID, true); isCreating {
		return
	}
	// Ensure cleanup even if panic occurs
	defer kbCollectionCreating.Delete(collectionID)

	// Check if collection already exists
	ctx := context.Background()
	existsResult, err := kb.API.CollectionExists(ctx, collectionID)
	if err != nil {
		// If check fails, log and continue to create (let create handle conflicts)
		log.Warn("failed to check collection existence: %v, will attempt to create", err)
	} else if existsResult != nil && existsResult.Exists {
		// Collection exists, no need to create
		return
	}

	// Build metadata
	metadata := make(map[string]interface{})
	for k, v := range chatKB.Metadata {
		metadata[k] = v
	}
	metadata["team_id"] = teamID
	metadata["user_id"] = userID

	// Ensure name and description are set (required fields)
	// Use user's locale from login context to determine language
	isZh := strings.HasPrefix(strings.ToLower(locale), "zh")
	if _, exists := metadata["name"]; !exists {
		if isZh {
			metadata["name"] = "对话知识库"
		} else {
			metadata["name"] = "Chat Knowledge Base"
		}
	}
	if _, exists := metadata["description"]; !exists {
		if isZh {
			metadata["description"] = "用户对话知识库"
		} else {
			metadata["description"] = "User chat knowledge base"
		}
	}

	// Build auth scope (use __yao_ prefix for permission fields)
	// Only set __yao_created_by for create operations (consistent with WithCreateScope)
	authScope := make(map[string]interface{})
	if teamID != "" {
		authScope["__yao_team_id"] = teamID
	}
	authScope["__yao_created_by"] = userID

	// Create new collection for this user
	createParams := &kbapi.CreateCollectionParams{
		ID:                  collectionID,
		EmbeddingProviderID: chatKB.EmbeddingProviderID,
		EmbeddingOptionID:   chatKB.EmbeddingOptionID,
		Locale:              chatKB.Locale,
		Config:              chatKB.Config,
		Metadata:            metadata,
		AuthScope:           authScope,
	}

	_, err = kb.API.CreateCollection(ctx, createParams)
	if err != nil {
		log.Warn("failed to create KB collection for user %s: %v", userID, err)
		return
	}

	log.Info("Created KB collection: %s for team=%s, user=%s", collectionID, teamID, userID)
}

// generateSessionID generates a session ID
func generateSessionID() string {
	return session.ID()
}

// GinLogout handles user logout
func GinLogout(c *gin.Context) {
	ctx := c.Request.Context()

	// Get access token and refresh token from cookies or headers
	// These methods already handle Bearer prefix removal and cookie prefixes
	accessToken := oauth.OAuth.GetAccessToken(c)
	refreshToken := oauth.OAuth.GetRefreshToken(c)

	// Revoke access token if present
	if accessToken != "" {
		err := oauth.OAuth.Revoke(ctx, accessToken, "access_token")
		if err != nil {
			log.Warn("Failed to revoke access token during logout: %v", err)
		}
	}

	// Revoke refresh token if present
	if refreshToken != "" {
		err := oauth.OAuth.Revoke(ctx, refreshToken, "refresh_token")
		if err != nil {
			log.Warn("Failed to revoke refresh token during logout: %v", err)
		}
	}

	// Clear all authentication cookies
	response.DeleteAllAuthCookies(c)

	// Return success response
	response.RespondWithSuccess(c, http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

// SendLoginCookies sends all necessary cookies for a successful login
// This includes access token, refresh token, and optionally session ID cookies with appropriate security settings
func SendLoginCookies(c *gin.Context, loginResponse *LoginResponse, sessionID string) {

	// Send session ID cookie only if sessionID is provided
	if sessionID != "" {
		expires := time.Now().Add(time.Duration(yaoClientConfig.ExpiresIn) * time.Second)
		options := response.NewSecureCookieOptions().
			WithExpires(expires).
			WithSameSite("Strict")
		response.SendSecureCookieWithOptions(c, "session_id", sessionID, options)
	}

	// MFA Temporary Access Token
	if loginResponse.Status == LoginStatusMFA {
		mfaToken := fmt.Sprintf("Bearer %s", loginResponse.AccessToken)
		expires := time.Now().Add(time.Duration(loginResponse.ExpiresIn) * time.Second)
		response.SendAccessTokenCookieWithExpiry(c, mfaToken, expires)
		return
	}

	// Normal Access Token
	accessToken := fmt.Sprintf("%s %s", loginResponse.TokenType, loginResponse.AccessToken)
	refreshToken := fmt.Sprintf("%s %s", loginResponse.TokenType, loginResponse.RefreshToken)

	// Calculate expiration times
	refreshExpires := time.Now().Add(time.Duration(loginResponse.RefreshTokenExpiresIn) * time.Second)

	// Send access token cookie
	response.SendAccessTokenCookieWithExpiry(c, accessToken, time.Now().Add(time.Duration(loginResponse.ExpiresIn)*time.Second))

	// Send refresh token cookie
	response.SendRefreshTokenCookieWithExpiry(c, refreshToken, refreshExpires)
}
