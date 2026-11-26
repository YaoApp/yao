package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/openapi/oauth"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
)

const (
	taoModeActivate = "activate"
	taoModeLogin    = "login"
)

// taoGetUserProvider is replaceable in unit tests; production uses OAuth.
var taoGetUserProvider = func() (oauthtypes.UserProvider, error) {
	if oauth.OAuth == nil {
		return nil, fmt.Errorf("oauth service is not initialized")
	}
	return oauth.OAuth.GetUserProvider()
}

type taoRefreshCall struct {
	done  chan struct{}
	token *OAuthTokenResponse
	err   error
}

var (
	taoRefreshMu    sync.Mutex
	taoRefreshCalls = map[string]*taoRefreshCall{}
)

type taoActivationResponse struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Reused       bool     `json:"reused"`
	Scopes       []string `json:"scopes"`
	RedirectURIs []string `json:"redirect_uris"`
	DeploymentID string   `json:"deployment_id"`
}

// taoAPIError preserves Tao's machine-readable error code for Yao mapping.
type taoAPIError struct {
	Code        string
	Description string
	Status      int
}

func (e *taoAPIError) Error() string {
	if e.Description != "" {
		return e.Description
	}
	return e.Code
}

func taoProvider() (*Provider, error) {
	provider, err := GetProvider("tao")
	if err != nil || provider == nil {
		return nil, fmt.Errorf("tao provider is not registered")
	}
	if provider.Issuer == "" {
		return nil, fmt.Errorf("tao oauth issuer is not configured")
	}
	return provider, nil
}

func taoBaseURL(c *gin.Context) string {
	if base := strings.TrimRight(os.Getenv("YAO_SERVER_URL"), "/"); base != "" {
		return base
	}
	return fmt.Sprintf("%s://%s", getScheme(c), c.Request.Host)
}

func taoActivationCallbackURL(c *gin.Context) string {
	return taoBaseURL(c) + "/v1/user/oauth/tao/activate/callback"
}

func taoLoginCallbackURL(c *gin.Context) string {
	return taoBaseURL(c) + "/v1/user/oauth/tao/callback"
}

func taoLoginRedirectURIs(c *gin.Context) []string {
	result := []string{taoLoginCallbackURL(c)}
	for _, raw := range strings.Split(os.Getenv("TAO_OAUTH_LOGIN_REDIRECT_URIS"), ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		duplicate := false
		for _, existing := range result {
			if existing == raw {
				duplicate = true
				break
			}
		}
		if !duplicate {
			result = append(result, raw)
		}
	}
	return result
}

// taoReturnURL accepts only the same-origin OAuth authorize endpoint as a
// login continuation target. It prevents open redirects while allowing the
// browser to resume an authorization request after Tao login.
func taoReturnURL(c *gin.Context) string {
	raw := strings.TrimSpace(c.Query("return_to"))
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || !strings.HasSuffix(parsed.Path, "/oauth/authorize") {
		return ""
	}
	if parsed.Scheme == "" && parsed.Host == "" {
		if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
			return ""
		}
		return raw
	}
	base, err := url.Parse(taoBaseURL(c))
	if err != nil || base.Scheme != parsed.Scheme || base.Host != parsed.Host {
		return ""
	}
	return raw
}

func taoAppendQuery(raw, key, value string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := parsed.Query()
	q.Set(key, value)
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func taoRandom(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func taoPKCE() (verifier, challenge string, err error) {
	verifier, err = taoRandom(64)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func taoStartLogin(c *gin.Context) (authURL, state string, err error) {
	provider, err := taoProvider()
	if err != nil {
		return "", "", err
	}
	creds, err := loadTaoClientCredentials()
	if err != nil {
		return "", "", err
	}
	state, err = taoRandom(32)
	if err != nil {
		return "", "", err
	}
	nonce, err := taoRandom(32)
	if err != nil {
		return "", "", err
	}
	locale := strings.TrimSpace(c.Query("locale"))
	if locale == "" {
		locale = "en"
	}
	prompt := strings.TrimSpace(c.Query("prompt"))
	loginHint := strings.TrimSpace(c.Query("login_hint"))
	maxAge := strings.TrimSpace(c.Query("max_age"))
	verifier, challenge, err := taoPKCE()
	if err != nil {
		return "", "", err
	}
	sid := utils.GetSessionID(c)
	if sid == "" {
		sid = generateSessionID()
		response.SendSessionCookie(c, sid)
	}
	mode := taoModeLogin
	if !creds.complete() {
		mode = taoModeActivate
	}
	redirectURI := taoLoginCallbackURL(c)
	var authorizationURL string
	if mode == taoModeActivate {
		deployment, err := loadTaoDeployment()
		if err != nil {
			return "", "", err
		}
		redirectURI = taoActivationCallbackURL(c)
		params := url.Values{}
		params.Set("deployment_id", deployment.ID)
		params.Set("deployment_public_key", deployment.PublicKey)
		params.Set("redirect_uri", redirectURI)
		for _, loginURI := range taoLoginRedirectURIs(c) {
			params.Add("login_redirect_uris", loginURI)
		}
		params.Set("scope", strings.Join(provider.Scopes, " "))
		params.Set("state", state)
		params.Set("code_challenge", challenge)
		params.Set("code_challenge_method", "S256")
		authorizationURL = strings.TrimRight(provider.Issuer, "/") + "/oauth/activate?" + params.Encode()
	} else {
		params := url.Values{}
		params.Set("response_type", "code")
		params.Set("client_id", creds.ClientID)
		params.Set("redirect_uri", redirectURI)
		params.Set("scope", strings.Join(provider.Scopes, " "))
		params.Set("state", state)
		params.Set("nonce", nonce)
		if prompt != "" {
			params.Set("prompt", prompt)
		}
		if loginHint != "" {
			params.Set("login_hint", loginHint)
		}
		if maxAge != "" {
			params.Set("max_age", maxAge)
		}
		params.Set("code_challenge", challenge)
		params.Set("code_challenge_method", "S256")
		authorizationURL = strings.TrimRight(provider.Issuer, "/") + "/oauth/authorize?" + params.Encode()
	}
	txn := &taoLoginTxn{
		State: state, Nonce: nonce, Verifier: verifier, Challenge: challenge,
		RedirectURI: redirectURI, SID: sid, Mode: mode, Locale: locale,
		Prompt: prompt, LoginHint: loginHint, MaxAge: maxAge,
		ReturnURL: taoReturnURL(c),
	}
	if err := saveTaoTxn(txn); err != nil {
		return "", "", err
	}
	return authorizationURL, state, nil
}

func getTaoAuthorizationURL(c *gin.Context) {
	authorizationURL, state, err := taoStartLogin(c)
	if err != nil {
		response.RespondWithError(c, http.StatusServiceUnavailable, &response.ErrorResponse{
			Code:             "tao_not_configured",
			ErrorDescription: err.Error(),
		})
		return
	}
	if c.Query("redirect") == "1" {
		c.Redirect(http.StatusFound, authorizationURL)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, &OAuthAuthorizationURLResponse{
		AuthorizationURL: authorizationURL,
		State:            state,
	})
}

// taoTxnSessionMatches binds a login transaction to the initiating browser
// only when both sides have a session id. Safari/WebKit does not send the
// SameSite=Lax session cookie on the cross-site redirect from an external IdP
// back to Yao, so a missing cookie must not invalidate an otherwise one-time
// state + PKCE transaction. If a cookie is present it must still match.
func taoTxnSessionMatches(c *gin.Context, txn *taoLoginTxn) bool {
	if txn == nil || txn.SID == "" {
		return true
	}
	current := utils.GetSessionID(c)
	return current == "" || current == txn.SID
}

func taoActivateCallback(c *gin.Context) {
	state := c.Query("state")
	activeCode := c.Query("activation_code")
	if activeCode == "" {
		activeCode = c.Query("code")
	}
	txn, err := loadTaoTxn(state)
	if err != nil || txn.Mode != taoModeActivate {
		taoCallbackFailure(c, "invalid_state")
		return
	}
	if !taoTxnSessionMatches(c, txn) {
		taoCallbackFailure(c, "invalid_state")
		return
	}
	// Preserve the caller's locale through the activation continuation.
	q := c.Request.URL.Query()
	q.Set("locale", txn.Locale)
	if txn.ReturnURL != "" {
		q.Set("return_to", txn.ReturnURL)
	}
	if txn.Prompt != "" {
		q.Set("prompt", txn.Prompt)
	}
	if txn.LoginHint != "" {
		q.Set("login_hint", txn.LoginHint)
	}
	if txn.MaxAge != "" {
		q.Set("max_age", txn.MaxAge)
	}
	c.Request.URL.RawQuery = q.Encode()
	deleteTaoTxn(state)
	if activeCode == "" {
		taoCallbackFailure(c, "tao_activation_failed")
		return
	}
	deployment, err := loadTaoDeployment()
	if err != nil {
		taoCallbackFailure(c, "tao_activation_failed")
		return
	}
	act, err := taoActivationExchange(c.Request.Context(), activeCode, txn.Verifier, deployment)
	if err != nil {
		taoCallbackFailure(c, taoActivationErrorCode(err))
		return
	}
	if act.ClientID == "" || act.ClientSecret == "" {
		taoCallbackFailure(c, "tao_activation_failed")
		return
	}
	if err := taoValidateActivationRedirects(act.RedirectURIs, taoActivationCallbackURL(c), taoLoginRedirectURIs(c)); err != nil {
		taoCallbackFailure(c, "tao_activation_failed")
		return
	}
	if err := saveTaoClientCredentials(act.ClientID, act.ClientSecret); err != nil {
		taoCallbackFailure(c, "tao_activation_failed")
		return
	}
	if provider, perr := taoProvider(); perr == nil {
		provider.ClientID = act.ClientID
		provider.ClientSecret = act.ClientSecret
	}
	authorizationURL, _, err := taoStartLogin(c)
	if err != nil {
		taoCallbackFailure(c, "tao_not_configured")
		return
	}
	c.Redirect(http.StatusFound, authorizationURL)
}

func taoLoginCallback(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")
	txn, err := loadTaoTxn(state)
	if err != nil || txn.Mode != taoModeLogin {
		taoCallbackFailure(c, "invalid_state")
		return
	}
	if !taoTxnSessionMatches(c, txn) {
		taoCallbackFailure(c, "invalid_state")
		return
	}
	deleteTaoTxn(state)
	if code == "" {
		taoCallbackFailure(c, "access_denied")
		return
	}
	provider, err := taoProvider()
	if err != nil {
		taoCallbackFailure(c, "tao_not_configured")
		return
	}
	tokenResponse, err := taoExchangeCode(c.Request.Context(), code, txn.RedirectURI, txn.Verifier)
	if err != nil {
		log.Warn("Tao login token exchange failed: %v", err)
		taoCallbackFailure(c, "server_error")
		return
	}
	creds, _ := loadTaoClientCredentials()
	clientID := ""
	if creds != nil {
		clientID = creds.ClientID
	}
	if err := taoValidateAccessToken(provider, tokenResponse.AccessToken, clientID); err != nil {
		log.Warn("Tao access token validation failed: %v", err)
		taoCallbackFailure(c, "server_error")
		return
	}
	userInfo, err := taoUserInfoFromToken(provider, tokenResponse, txn.Nonce)
	if err != nil {
		log.Warn("Tao userinfo failed: %v", err)
		taoCallbackFailure(c, "server_error")
		return
	}
	loginCtx := makeLoginContext(c)
	loginCtx.AuthSource = "tao"
	loginCtx.RememberMe = true
	sid := utils.GetSessionID(c)
	if sid == "" {
		sid = generateSessionID()
		response.SendSessionCookie(c, sid)
	}
	locale := txn.Locale
	if locale == "" {
		locale = "en"
	}
	loginResponse, err := LoginThirdParty("tao", userInfo, loginCtx, locale)
	if err != nil {
		log.Warn("Tao third-party login failed: %v", err)
		taoCallbackFailure(c, "server_error")
		return
	}
	if err := taoSaveRefreshToken(c.Request.Context(), userInfo.Sub, tokenResponse); err != nil {
		log.Warn("failed to save Tao refresh token: %v", err)
	}
	if loginResponse.AccessToken == "" {
		log.Warn("Tao login did not return an access token; cannot provision managed key")
		taoCallbackFailure(c, "server_error")
		return
	}
	userID, userIDErr := taoUserID(c.Request.Context(), userInfo.Sub)
	if userIDErr != nil {
		log.Warn("failed to resolve Tao user for managed key: %v", userIDErr)
		taoCallbackFailure(c, "server_error")
		return
	}
	if keyErr := taoEnsureManagedKey(c.Request.Context(), userID, tokenResponse.AccessToken); keyErr != nil {
		log.Warn("failed to provision Tao managed key: %v", keyErr)
		taoCallbackFailure(c, "server_error")
		return
	}
	SendLoginCookies(c, loginResponse, sid)
	target := txn.ReturnURL
	if target == "" {
		target = taoLoginSuccessURL(txn.Locale)
	} else if handoffID, err := saveTaoLoginHandoff(loginResponse.AccessToken, loginResponse.RefreshToken, sid); err == nil {
		// The redirect from Tao to /v1/oauth/authorize is cross-site. Access
		// token cookies are SameSite=Strict and may be dropped on that hop, so
		// let authorize consume a one-time server-side handoff instead.
		target = taoAppendQuery(target, "login_handoff", handoffID)
	}
	c.Redirect(http.StatusFound, target)
}

// taoLoginSuccessURL returns the configured post-login destination. The entry
// configuration is part of the OpenAPI app config and may point at the
// frontend's real entry route instead of the site root.
func taoLoginSuccessURL(locale string) string {
	return taoLoginSuccessURLFromConfig(GetEntryConfig(locale))
}

func taoLoginSuccessURLFromConfig(cfg *EntryConfig) string {
	if cfg != nil {
		if target := configuredEntryURL(cfg.SuccessURL); target != "" {
			return target
		}
	}
	return "/auth/entry/success"
}

// configuredEntryURL rejects unresolved env placeholders so a shared app config
// can opt in to an environment-specific frontend URL without breaking tests
// or deployments that do not define it.
func configuredEntryURL(raw string) string {
	target := strings.TrimSpace(raw)
	if target == "" || strings.HasPrefix(target, "$ENV.") || strings.HasPrefix(target, "${") {
		return ""
	}
	return target
}

// taoLoginFailureURL returns the configured failure destination with the
// OAuth error code and a human-readable message.
func taoLoginFailureURL(locale, code string) string {
	return taoLoginFailureURLFromConfig(GetEntryConfig(locale), code)
}

func taoLoginFailureURLFromConfig(cfg *EntryConfig, code string) string {
	target := "/auth/entry/failure"
	if cfg != nil {
		if configured := configuredEntryURL(cfg.FailureURL); configured != "" {
			target = configured
		}
	}
	separator := "?"
	if strings.Contains(target, "?") {
		separator = "&"
	}
	values := url.Values{}
	values.Set("code", code)
	values.Set("message", taoLoginFailureMessage(code))
	return target + separator + values.Encode()
}

func taoLoginFailureMessage(code string) string {
	switch code {
	case "invalid_state":
		return "Invalid or expired login state"
	case "access_denied":
		return "Login was cancelled or denied"
	case "tao_not_configured":
		return "Tao login is not configured"
	case "tao_activation_failed":
		return "Tao activation failed"
	case "tao_activation_conflict":
		return "Tao activation conflicts with an existing deployment"
	case "tao_activation_rate_limited":
		return "Tao activation is rate limited"
	default:
		return "Login failed"
	}
}

func taoUserID(ctx context.Context, sub string) (string, error) {
	provider, err := taoGetUserProvider()
	if err != nil {
		return "", err
	}
	return provider.GetOAuthUserID(ctx, "tao", sub)
}

func taoSaveRefreshToken(ctx context.Context, sub string, token *OAuthTokenResponse) error {
	if token == nil || token.RefreshToken == "" {
		return nil
	}
	provider, err := taoGetUserProvider()
	if err != nil {
		return err
	}
	data := maps.MapStrAny{
		"refresh_token":    token.RefreshToken,
		"token_expires_at": time.Now().Add(time.Duration(token.ExpiresIn) * time.Second),
		"token_scope":      token.Scope,
		"token_updated_at": time.Now(),
	}
	return provider.UpdateOAuthAccount(ctx, "tao", sub, data)
}

func taoExchangeCode(ctx context.Context, code, redirectURI, verifier string) (*OAuthTokenResponse, error) {
	provider, err := taoProvider()
	if err != nil {
		return nil, err
	}
	creds, err := loadTaoClientCredentials()
	if err != nil {
		return nil, err
	}
	if !creds.complete() {
		return nil, fmt.Errorf("tao client is not available")
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(provider.Issuer, "/")+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(creds.ClientID, creds.ClientSecret)
	resp, err := taoHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var token OAuthTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("decode tao token: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if token.ErrorDesc != "" {
			return nil, fmt.Errorf("tao token: %s", token.ErrorDesc)
		}
		return nil, fmt.Errorf("tao token status %d", resp.StatusCode)
	}
	return &token, nil
}

func taoActivationExchange(ctx context.Context, activationCode, verifier string, deployment *taoDeployment) (*taoActivationResponse, error) {
	provider, err := taoProvider()
	if err != nil {
		return nil, err
	}
	payload := map[string]string{
		"activation_code":       activationCode,
		"code_verifier":         verifier,
		"deployment_id":         deployment.ID,
		"deployment_public_key": deployment.PublicKey,
	}
	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(provider.Issuer, "/")+"/oauth/activate/exchange", strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := taoHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result taoActivationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode activation: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		_ = json.Unmarshal(body, &apiErr)
		if apiErr.Error != "" {
			return nil, &taoAPIError{Code: apiErr.Error, Description: apiErr.Description, Status: resp.StatusCode}
		}
		return nil, fmt.Errorf("activation status %d", resp.StatusCode)
	}
	return &result, nil
}

func taoValidateAccessToken(provider *Provider, accessToken, clientID string) error {
	if accessToken == "" {
		return fmt.Errorf("access token is required")
	}
	parser := &jwt.Parser{SkipClaimsValidation: true}
	token, err := parser.Parse(accessToken, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != "RS256" {
			return nil, fmt.Errorf("unexpected access token alg: %s", t.Method.Alg())
		}
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("missing access token kid")
		}
		return provider.getJWKSPublicKey(kid)
	})
	if err != nil || !token.Valid {
		return fmt.Errorf("invalid access token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid access token claims")
	}
	if iss, _ := claims["iss"].(string); iss == "" || iss != provider.Issuer {
		return fmt.Errorf("invalid access token issuer")
	}
	if exp, ok := claims["exp"].(float64); !ok || time.Now().Unix() > int64(exp) {
		return fmt.Errorf("access token expired")
	}
	if aud, _ := claims["aud"].(string); aud == "" {
		return fmt.Errorf("missing access token audience")
	}
	if sub, _ := claims["sub"].(string); sub == "" {
		return fmt.Errorf("missing access token subject")
	}
	if clientID != "" {
		claimClient := ""
		if v, _ := claims["client_id"].(string); v != "" {
			claimClient = v
		} else if v, _ := claims["azp"].(string); v != "" {
			claimClient = v
		}
		if claimClient != "" && claimClient != clientID {
			return fmt.Errorf("invalid access token client")
		}
	}
	return nil
}

func taoUserInfoFromToken(provider *Provider, token *OAuthTokenResponse, nonce string) (*oauthtypes.OIDCUserInfo, error) {
	if token == nil {
		return nil, fmt.Errorf("token response is required")
	}
	if token.IDToken != "" {
		claims, err := provider.verifyIDTokenAndGetClaims(token.IDToken)
		if err != nil {
			return nil, err
		}
		creds, _ := loadTaoClientCredentials()
		clientID := ""
		if creds != nil {
			clientID = creds.ClientID
		}
		if err := taoValidateClaims(claims, provider.Issuer, nonce, clientID); err != nil {
			return nil, err
		}
		return provider.mapUserInfoResponse(claims), nil
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}
	return provider.GetUserInfo(token.AccessToken, token.TokenType)
}

func taoValidateClaims(claims map[string]interface{}, issuer, nonce, clientID string) error {
	if issuer == "" {
		return fmt.Errorf("issuer is required")
	}
	if got, _ := claims["iss"].(string); got != issuer {
		return fmt.Errorf("invalid id_token issuer")
	}
	if nonce != "" {
		if got, _ := claims["nonce"].(string); got != nonce {
			return fmt.Errorf("invalid id_token nonce")
		}
	}
	if clientID != "" {
		switch aud := claims["aud"].(type) {
		case string:
			if aud != clientID {
				return fmt.Errorf("invalid id_token audience")
			}
		case []interface{}:
			found := false
			for _, item := range aud {
				if s, _ := item.(string); s == clientID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("invalid id_token audience")
			}
		default:
			return fmt.Errorf("invalid id_token audience")
		}
	}
	return nil
}

func taoActivationErrorCode(err error) string {
	var apiErr *taoAPIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case "deployment_conflict", "client_unavailable":
			return "tao_activation_conflict"
		case "rate_limited", "client_quota_exceeded":
			return "tao_activation_rate_limited"
		}
	}
	return "tao_activation_failed"
}

func taoCallbackFailure(c *gin.Context, code string) {
	c.Redirect(http.StatusFound, taoLoginFailureURL(c.Query("locale"), code))
}

func taoHTTPClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func taoValidateActivationRedirects(registered []string, activation string, loginURIs []string) error {
	set := make(map[string]bool, len(registered))
	for _, uri := range registered {
		set[uri] = true
	}
	if !set[activation] {
		return fmt.Errorf("activation callback is not registered")
	}
	for _, uri := range loginURIs {
		if !set[uri] {
			return fmt.Errorf("login callback is not registered: %s", uri)
		}
	}
	return nil
}

// taoRefreshAccessToken exchanges a stored Tao refresh token. The caller must
// persist the returned token pair; Tao uses rotating refresh tokens.
func taoRefreshAccessToken(ctx context.Context, refreshToken string) (*OAuthTokenResponse, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}
	provider, err := taoProvider()
	if err != nil {
		return nil, err
	}
	creds, err := loadTaoClientCredentials()
	if err != nil {
		return nil, err
	}
	if !creds.complete() {
		return nil, fmt.Errorf("tao client is not available")
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(provider.Issuer, "/")+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(creds.ClientID, creds.ClientSecret)
	resp, err := taoHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var token OAuthTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("decode tao refresh: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if token.ErrorDesc != "" {
			return nil, fmt.Errorf("tao refresh: %s", token.ErrorDesc)
		}
		return nil, fmt.Errorf("tao refresh status %d", resp.StatusCode)
	}
	return &token, nil
}

// taoRefreshUserToken refreshes and persists the Tao token pair for a Yao user.
// Concurrent calls for the same user share one rotation.
func taoRefreshUserToken(ctx context.Context, userID string) (*OAuthTokenResponse, error) {
	taoRefreshMu.Lock()
	if call, ok := taoRefreshCalls[userID]; ok {
		taoRefreshMu.Unlock()
		select {
		case <-call.done:
			return call.token, call.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	call := &taoRefreshCall{done: make(chan struct{})}
	taoRefreshCalls[userID] = call
	taoRefreshMu.Unlock()

	call.token, call.err = taoRefreshUserTokenOnce(ctx, userID)
	taoRefreshMu.Lock()
	delete(taoRefreshCalls, userID)
	close(call.done)
	taoRefreshMu.Unlock()
	return call.token, call.err
}

func taoRefreshUserTokenOnce(ctx context.Context, userID string) (*OAuthTokenResponse, error) {
	provider, err := taoGetUserProvider()
	if err != nil {
		return nil, err
	}
	accounts, err := provider.GetUserOAuthAccounts(ctx, userID)
	if err != nil {
		return nil, err
	}
	refreshToken := ""
	subject := ""
	for _, account := range accounts {
		if fmt.Sprint(account["provider"]) == "tao" {
			refreshToken = fmt.Sprint(account["refresh_token"])
			subject = fmt.Sprint(account["sub"])
			break
		}
	}
	if refreshToken == "" {
		return nil, fmt.Errorf("tao refresh token is not available")
	}
	token, err := taoRefreshAccessToken(ctx, refreshToken)
	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") {
			clearTaoRefreshTokenForUser(ctx, userID)
		}
		return nil, err
	}
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}
	if err := taoSaveRefreshToken(ctx, subject, token); err != nil {
		return nil, err
	}
	return token, nil
}
