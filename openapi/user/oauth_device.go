package user

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/http"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
)

// deviceAuthorize initiates Device Flow (RFC 8628) with a third-party IdP.
// POST /user/oauth/:provider/device/authorize
func deviceAuthorize(c *gin.Context) {
	providerID := c.Param("provider")

	provider, err := GetProvider(providerID)
	if err != nil || provider == nil {
		response.RespondWithError(c, response.StatusNotFound, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("OAuth provider '%s' not found", providerID),
		})
		return
	}

	if provider.Endpoints == nil || provider.Endpoints.DeviceAuthorization == "" {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Provider '%s' does not support Device Flow", providerID),
		})
		return
	}

	// Use DeviceClientID if available, fallback to ClientID
	clientID := provider.ClientID
	if provider.DeviceClientID != "" {
		clientID = provider.DeviceClientID
	}

	params := map[string]string{
		"client_id": clientID,
		"scope":     strings.Join(provider.Scopes, " "),
	}

	req := http.New(provider.Endpoints.DeviceAuthorization).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "Yao-OAuth-Client/1.0")

	resp := req.Post(params)
	if resp == nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to contact IdP device authorization endpoint",
		})
		return
	}

	if resp.Code != 200 {
		errMsg := fmt.Sprintf("IdP device authorization failed with status %d", resp.Code)
		if resp.Data != nil {
			if data, ok := resp.Data.(map[string]interface{}); ok {
				if desc, ok := data["error_description"]; ok {
					errMsg = fmt.Sprintf("%v", desc)
				} else if e, ok := data["error"]; ok {
					errMsg = fmt.Sprintf("%v", e)
				}
			}
		}
		response.RespondWithError(c, response.StatusBadGateway, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: errMsg,
		})
		return
	}

	var deviceResp DeviceAuthResponse
	if err := parseResponseData(resp.Data, &deviceResp); err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Failed to parse IdP response: %v", err),
		})
		return
	}

	if deviceResp.DeviceCode == "" || deviceResp.UserCode == "" {
		response.RespondWithError(c, response.StatusBadGateway, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "IdP returned incomplete device authorization response",
		})
		return
	}

	// Normalize: Google returns verification_url, RFC uses verification_uri
	if deviceResp.VerificationURI == "" && deviceResp.VerificationURL != "" {
		deviceResp.VerificationURI = deviceResp.VerificationURL
	}

	// Default interval to 5 seconds if not provided
	if deviceResp.Interval == 0 {
		deviceResp.Interval = 5
	}

	response.RespondWithSuccess(c, response.StatusOK, deviceResp)
}

// deviceToken polls the IdP token endpoint during Device Flow.
// On success, completes the full login flow (GetUserInfo + LoginThirdParty + SendLoginCookies).
// POST /user/oauth/:provider/device/token
func deviceToken(c *gin.Context) {
	providerID := c.Param("provider")
	sid := utils.GetSessionID(c)

	var params DeviceTokenRequest
	if err := c.ShouldBind(&params); err != nil {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "device_code is required",
		})
		return
	}

	provider, err := GetProvider(providerID)
	if err != nil || provider == nil {
		response.RespondWithError(c, response.StatusNotFound, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("OAuth provider '%s' not found", providerID),
		})
		return
	}

	if provider.Endpoints == nil || provider.Endpoints.Token == "" {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Provider token endpoint not configured",
		})
		return
	}

	clientID := provider.ClientID
	if provider.DeviceClientID != "" {
		clientID = provider.DeviceClientID
	}

	tokenParams := map[string]string{
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		"device_code": params.DeviceCode,
		"client_id":   clientID,
	}
	if provider.DeviceClientID != "" && provider.DeviceClientSecret != "" {
		tokenParams["client_secret"] = provider.DeviceClientSecret
	} else if provider.ClientSecret != "" {
		tokenParams["client_secret"] = provider.ClientSecret
	}

	req := http.New(provider.Endpoints.Token).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "Yao-OAuth-Client/1.0")

	resp := req.Post(tokenParams)
	if resp == nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to contact IdP token endpoint",
		})
		return
	}

	// Parse IdP response to check for pending/error states
	var idpResp map[string]interface{}
	if err := parseResponseData(resp.Data, &idpResp); err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Failed to parse IdP token response: %v", err),
		})
		return
	}

	// Check for Device Flow specific error responses (HTTP 400 with error field)
	if errStr, ok := idpResp["error"].(string); ok && errStr != "" {
		switch errStr {
		case "authorization_pending":
			response.RespondWithSuccess(c, response.StatusOK, DeviceTokenResponse{Status: "pending"})
			return
		case "slow_down":
			response.RespondWithSuccess(c, response.StatusOK, DeviceTokenResponse{Status: "slow_down"})
			return
		case "expired_token":
			response.RespondWithSuccess(c, response.StatusOK, DeviceTokenResponse{Status: "expired"})
			return
		case "access_denied":
			response.RespondWithSuccess(c, response.StatusOK, DeviceTokenResponse{Status: "denied"})
			return
		default:
			desc := ""
			if d, ok := idpResp["error_description"].(string); ok {
				desc = d
			}
			log.With(log.F{"provider": providerID, "error": errStr, "desc": desc}).Error("Device Flow token error")
			response.RespondWithError(c, response.StatusBadGateway, &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: fmt.Sprintf("IdP error: %s", errStr),
			})
			return
		}
	}

	// Success: IdP returned access_token. Parse into OAuthTokenResponse.
	var tokenResponse OAuthTokenResponse
	if err := parseResponseData(resp.Data, &tokenResponse); err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Failed to parse token response: %v", err),
		})
		return
	}

	if tokenResponse.AccessToken == "" {
		response.RespondWithError(c, response.StatusBadGateway, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "IdP returned empty access token",
		})
		return
	}

	// --- Login flow (mirrors authback L156-222, independent implementation) ---

	// Get user info based on provider configuration
	var userInfo *OAuthUserInfoResponse
	if provider.UserInfoSource == UserInfoSourceIDToken {
		userInfo, err = provider.GetUserInfoFromTokenResponse(&tokenResponse)
	} else {
		userInfo, err = provider.GetUserInfo(tokenResponse.AccessToken, tokenResponse.TokenType)
	}

	if err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Failed to get user info: %v", err),
		})
		return
	}

	loginCtx := makeLoginContext(c)
	loginCtx.AuthSource = providerID
	loginCtx.RememberMe = true

	locale := params.Locale
	if locale == "" {
		locale = "en"
	}

	loginResponse, err := LoginThirdParty(providerID, userInfo, loginCtx, locale)
	if err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to login: " + err.Error(),
		})
		return
	}

	SendLoginCookies(c, loginResponse, sid)

	switch loginResponse.Status {
	case LoginStatusInviteVerification, LoginStatusMFA, LoginStatusTeamSelection:
		response.RespondWithSuccess(c, response.StatusOK, DeviceTokenResponse{
			Status:      "success",
			SessionID:   sid,
			AccessToken: loginResponse.AccessToken,
			ExpiresIn:   loginResponse.ExpiresIn,
			MFAEnabled:  loginResponse.MFAEnabled,
		})
	case LoginStatusSuccess:
		response.RespondWithSuccess(c, response.StatusOK, DeviceTokenResponse{
			Status:                "success",
			SessionID:             sid,
			IDToken:               loginResponse.IDToken,
			AccessToken:           loginResponse.AccessToken,
			RefreshToken:          loginResponse.RefreshToken,
			ExpiresIn:             loginResponse.ExpiresIn,
			RefreshTokenExpiresIn: loginResponse.RefreshTokenExpiresIn,
			MFAEnabled:            loginResponse.MFAEnabled,
		})
	default:
		response.RespondWithSuccess(c, response.StatusOK, DeviceTokenResponse{
			Status:      "success",
			SessionID:   sid,
			IDToken:     loginResponse.IDToken,
			AccessToken: loginResponse.AccessToken,
			ExpiresIn:   loginResponse.ExpiresIn,
		})
	}
}

// parseResponseData converts gou/http response data into a target struct.
func parseResponseData(data interface{}, target interface{}) error {
	switch d := data.(type) {
	case map[string]interface{}:
		jsonBytes, err := json.Marshal(d)
		if err != nil {
			return fmt.Errorf("failed to marshal: %w", err)
		}
		return json.Unmarshal(jsonBytes, target)
	case []byte:
		return json.Unmarshal(d, target)
	case string:
		return json.Unmarshal([]byte(d), target)
	default:
		return fmt.Errorf("unexpected data type: %T", data)
	}
}
