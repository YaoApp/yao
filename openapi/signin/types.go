package signin

// Config represents the signin page configuration
type Config struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	SuccessURL  string       `json:"success_url,omitempty"`
	FailureURL  string       `json:"failure_url,omitempty"`
	Form        *FormConfig  `json:"form,omitempty"`
	Token       *TokenConfig `json:"token,omitempty"`
	ThirdParty  *ThirdParty  `json:"third_party,omitempty"`
}

// FormConfig represents the form configuration
type FormConfig struct {
	Username           *UsernameConfig `json:"username,omitempty"`
	Password           *PasswordConfig `json:"password,omitempty"`
	Captcha            *CaptchaConfig  `json:"captcha,omitempty"`
	ForgotPasswordLink bool            `json:"forgot_password_link,omitempty"`
	RememberMe         bool            `json:"remember_me,omitempty"`
	RegisterLink       string          `json:"register_link,omitempty"`
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
	ExpiresIn           string `json:"expires_in,omitempty"`
	RememberMeExpiresIn string `json:"remember_me_expires_in,omitempty"`
}

// ThirdParty represents the third party login configuration
type ThirdParty struct {
	Register  *RegisterConfig `json:"register,omitempty"`
	Providers []*Provider     `json:"providers,omitempty"`
}

// RegisterConfig represents the auto register configuration
type RegisterConfig struct {
	Auto bool   `json:"auto,omitempty"`
	Role string `json:"role,omitempty"`
}

// Provider represents a third party login provider
type Provider struct {
	ID                    string            `json:"id,omitempty"`
	Title                 string            `json:"title,omitempty"`
	Logo                  string            `json:"logo,omitempty"`
	Color                 string            `json:"color,omitempty"`
	TextColor             string            `json:"text_color,omitempty"`
	ClientID              string            `json:"client_id,omitempty"`
	ClientSecret          string            `json:"client_secret,omitempty"`
	ClientSecretGenerator *SecretGenerator  `json:"client_secret_generator,omitempty"`
	Scopes                []string          `json:"scopes,omitempty"`
	ResponseMode          string            `json:"response_mode,omitempty"`
	Endpoints             *Endpoints        `json:"endpoints,omitempty"`
	Mapping               map[string]string `json:"mapping,omitempty"`
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
}

// ==== API Types ====

// OAuthAuthorizationURLResponse represents the response for OAuth authorization URL
type OAuthAuthorizationURLResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	State            string `json:"state"`
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

// OAuthUserInfoResponse represents the user information from OAuth provider
type OAuthUserInfoResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	Username string `json:"username"`
}
