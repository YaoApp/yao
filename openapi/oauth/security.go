package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/yao/openapi/oauth/types"
)

// GenerateCodeChallenge generates a code challenge from a code verifier
// This is used for PKCE (Proof Key for Code Exchange) flow
func (s *Service) GenerateCodeChallenge(ctx context.Context, codeVerifier string, method string) (string, error) {
	switch method {
	case "S256":
		// SHA256 hash of the code verifier
		hash := sha256.Sum256([]byte(codeVerifier))
		return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:]), nil
	case "plain":
		// Plain text code verifier (not recommended for production)
		return codeVerifier, nil
	default:
		return "", fmt.Errorf("unsupported code challenge method: %s", method)
	}
}

// ValidateCodeChallenge validates a code verifier against a code challenge
// This verifies the PKCE code challenge during token exchange
func (s *Service) ValidateCodeChallenge(ctx context.Context, codeVerifier string, codeChallenge string, method string) error {
	expectedChallenge, err := s.GenerateCodeChallenge(ctx, codeVerifier, method)
	if err != nil {
		return err
	}

	if expectedChallenge != codeChallenge {
		return fmt.Errorf("code challenge verification failed")
	}

	return nil
}

// ValidateStateParameter validates OAuth state parameters
// This prevents CSRF attacks by verifying state parameters
func (s *Service) ValidateStateParameter(ctx context.Context, state string, clientID string) (*types.ValidationResult, error) {
	result := &types.ValidationResult{Valid: false}

	// Get state parameter from store
	stateKey := s.stateParameterKey(clientID, state)

	// Try cache first if available
	if s.cache != nil {
		if cached, ok := s.cache.Get(stateKey); ok {
			if stateParam, ok := cached.(*types.StateParameter); ok {
				// Check if state parameter is still valid
				if time.Now().Before(stateParam.ExpiresAt) {
					result.Valid = true
					return result, nil
				}
			}
		}
	}

	// Try store
	data, ok := s.store.Get(stateKey)
	if !ok {
		result.Errors = append(result.Errors, "State parameter not found")
		return result, nil
	}

	// Parse state parameter from store
	stateParam, ok := data.(*types.StateParameter)
	if !ok {
		result.Errors = append(result.Errors, "Invalid state parameter format")
		return result, nil
	}

	// Check if state parameter is still valid
	if time.Now().After(stateParam.ExpiresAt) {
		result.Errors = append(result.Errors, "State parameter has expired")
		return result, nil
	}

	// Validate that the state parameter belongs to the client
	if stateParam.ClientID != clientID {
		result.Errors = append(result.Errors, "State parameter does not belong to this client")
		return result, nil
	}

	result.Valid = true
	return result, nil
}

// GenerateStateParameter generates a secure state parameter
// This creates cryptographically secure state values for CSRF protection
func (s *Service) GenerateStateParameter(ctx context.Context, clientID string) (*types.StateParameter, error) {
	// Generate random state value
	length := s.config.Security.StateParameterLength
	if length == 0 {
		length = 32
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate state parameter: %w", err)
	}

	stateValue := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)

	// Create state parameter
	stateParam := &types.StateParameter{
		Value:     stateValue,
		ClientID:  clientID,
		ExpiresAt: time.Now().Add(s.config.Security.StateParameterLifetime),
	}

	// Store state parameter
	stateKey := s.stateParameterKey(clientID, stateValue)

	// Store in cache if available
	if s.cache != nil {
		s.cache.Set(stateKey, stateParam, s.config.Security.StateParameterLifetime)
	}

	// Store in persistent store
	if err := s.store.Set(stateKey, stateParam, s.config.Security.StateParameterLifetime); err != nil {
		return nil, fmt.Errorf("failed to store state parameter: %w", err)
	}

	return stateParam, nil
}

// ValidateRedirectURI validates redirect URIs against registered URIs
func (s *Service) ValidateRedirectURI(ctx context.Context, redirectURI string, registeredURIs []string) (*types.ValidationResult, error) {
	// This method signature doesn't match our ClientProvider interface
	// For now, we'll do a basic validation since we don't have a clientID
	result := &types.ValidationResult{Valid: false}

	// If no registered URIs provided, cannot validate
	if len(registeredURIs) == 0 {
		result.Errors = append(result.Errors, "No registered URIs provided")
		return result, nil
	}

	// Check if redirect URI matches any registered URI
	for _, uri := range registeredURIs {
		if uri == redirectURI {
			result.Valid = true
			return result, nil
		}
	}

	result.Errors = append(result.Errors, "Redirect URI not found in registered URIs")
	return result, nil
}

// ValidateRedirectURIForClient validates redirect URIs for a specific client
func (s *Service) ValidateRedirectURIForClient(ctx context.Context, clientID string, redirectURI string) (*types.ValidationResult, error) {
	return s.clientProvider.ValidateRedirectURI(ctx, clientID, redirectURI)
}

// PushAuthorizationRequest processes a pushed authorization request
// This implements RFC 9126 for enhanced security
func (s *Service) PushAuthorizationRequest(ctx context.Context, request *types.PushedAuthorizationRequest) (*types.PushedAuthorizationResponse, error) {
	// Validate client
	_, err := s.clientProvider.GetClientByID(ctx, request.ClientID)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidClient,
			ErrorDescription: "Invalid client",
		}
	}

	// Validate redirect URI
	validationResult, err := s.clientProvider.ValidateRedirectURI(ctx, request.ClientID, request.RedirectURI)
	if err != nil {
		return nil, err
	}
	if !validationResult.Valid {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "Invalid redirect URI",
		}
	}

	// Validate scopes if provided
	if request.Scope != "" {
		scopes := strings.Fields(request.Scope)
		scopeValidation, err := s.clientProvider.ValidateScope(ctx, request.ClientID, scopes)
		if err != nil {
			return nil, err
		}
		if !scopeValidation.Valid {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorInvalidScope,
				ErrorDescription: "Invalid scope",
			}
		}
	}

	// Generate request URI
	requestURI := s.generateRequestURI()

	// Store the request
	requestKey := s.pushedAuthRequestKey(requestURI)
	expiresIn := 600 // 10 minutes

	if s.cache != nil {
		s.cache.Set(requestKey, request, time.Duration(expiresIn)*time.Second)
	}

	if err := s.store.Set(requestKey, request, time.Duration(expiresIn)*time.Second); err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to store pushed authorization request",
		}
	}

	response := &types.PushedAuthorizationResponse{
		RequestURI: requestURI,
		ExpiresIn:  expiresIn,
	}

	return response, nil
}

// GetPushedAuthorizationRequest retrieves a PAR record without consuming it.
// The authorize endpoint may pass through Yao's login page before issuing a
// code, so the record must remain available until authorization succeeds.
func (s *Service) GetPushedAuthorizationRequest(ctx context.Context, requestURI string) (*types.PushedAuthorizationRequest, error) {
	if strings.TrimSpace(requestURI) == "" {
		return nil, &types.ErrorResponse{Code: types.ErrorInvalidRequest, ErrorDescription: "request_uri is required"}
	}
	key := s.pushedAuthRequestKey(requestURI)
	var data interface{}
	var ok bool
	if s.cache != nil {
		data, ok = s.cache.Get(key)
	}
	if !ok {
		data, ok = s.store.Get(key)
	}
	if !ok {
		return nil, &types.ErrorResponse{Code: types.ErrorInvalidRequest, ErrorDescription: "Invalid or expired request_uri"}
	}

	switch value := data.(type) {
	case *types.PushedAuthorizationRequest:
		return value, nil
	case types.PushedAuthorizationRequest:
		return &value, nil
	case map[string]interface{}:
		request := &types.PushedAuthorizationRequest{}
		if v, ok := value["client_id"].(string); ok {
			request.ClientID = v
		}
		if v, ok := value["response_type"].(string); ok {
			request.ResponseType = v
		}
		if v, ok := value["redirect_uri"].(string); ok {
			request.RedirectURI = v
		}
		if v, ok := value["scope"].(string); ok {
			request.Scope = v
		}
		if v, ok := value["state"].(string); ok {
			request.State = v
		}
		if v, ok := value["code_challenge"].(string); ok {
			request.CodeChallenge = v
		}
		if v, ok := value["code_challenge_method"].(string); ok {
			request.CodeChallengeMethod = v
		}
		if v, ok := value["resource"].(string); ok {
			request.Resource = v
		}
		return request, nil
	default:
		return nil, &types.ErrorResponse{Code: types.ErrorServerError, ErrorDescription: "Invalid request_uri data"}
	}
}

// ConsumePushedAuthorizationRequest retrieves a PAR record and deletes it so
// the request_uri cannot be replayed after a successful authorization.
func (s *Service) ConsumePushedAuthorizationRequest(ctx context.Context, requestURI string) (*types.PushedAuthorizationRequest, error) {
	request, err := s.GetPushedAuthorizationRequest(ctx, requestURI)
	if err != nil {
		return nil, err
	}
	key := s.pushedAuthRequestKey(requestURI)
	if s.cache != nil {
		s.cache.Del(key)
	}
	_ = s.store.Del(key)
	return request, nil
}

// Helper methods

// stateParameterKey generates a key for state parameter storage
func (s *Service) stateParameterKey(clientID string, state string) string {
	return fmt.Sprintf("%soauth:state:%s:%s", s.prefix, clientID, state)
}

// pushedAuthRequestKey generates a key for pushed authorization request storage
func (s *Service) pushedAuthRequestKey(requestURI string) string {
	return fmt.Sprintf("%soauth:par:%s", s.prefix, s.hashKey(requestURI))
}

// generateRequestURI generates a request URI for pushed authorization requests
func (s *Service) generateRequestURI() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return fmt.Sprintf("urn:ietf:params:oauth:request_uri:%s",
		base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes))
}

const consentTokenTTL = 10 * time.Minute

// ConsentRecord is the server-side state bound to a consent form token.
type ConsentRecord struct {
	Request        *types.AuthorizationRequest `json:"request"`
	UserID         string                      `json:"user_id"`
	Subject        string                      `json:"subject"`
	AuthSource     string                      `json:"auth_source,omitempty"`
	IdentitySource string                      `json:"identity_source,omitempty"`
}

func (s *Service) consentTokenKey(token string) string {
	return fmt.Sprintf("%soauth:consent:%s", s.prefix, s.hashKey(token))
}

// CreateConsentToken stores an authorization request for the consent page and
// returns a one-time CSRF token. The request must be confirmed by POSTing it
// back to the authorize endpoint.
func (s *Service) CreateConsentToken(ctx context.Context, record *ConsentRecord) (string, error) {
	if record == nil || record.Request == nil || record.UserID == "" {
		return "", fmt.Errorf("invalid consent record")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	data, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	key := s.consentTokenKey(token)
	if s.cache != nil {
		_ = s.cache.Set(key, string(data), consentTokenTTL)
	}
	if err := s.store.Set(key, string(data), consentTokenTTL); err != nil {
		return "", err
	}
	return token, nil
}

// ConsumeConsentToken retrieves and deletes a consent form token.
func (s *Service) ConsumeConsentToken(ctx context.Context, token string) (*ConsentRecord, error) {
	if strings.TrimSpace(token) == "" {
		return nil, &types.ErrorResponse{Code: types.ErrorInvalidRequest, ErrorDescription: "consent_token is required"}
	}
	key := s.consentTokenKey(token)
	var raw interface{}
	var ok bool
	if s.cache != nil {
		raw, ok = s.cache.Get(key)
	}
	if !ok {
		raw, ok = s.store.Get(key)
	}
	if !ok {
		return nil, &types.ErrorResponse{Code: types.ErrorInvalidRequest, ErrorDescription: "Invalid or expired consent token"}
	}
	if s.cache != nil {
		s.cache.Del(key)
	}
	_ = s.store.Del(key)
	text, ok := raw.(string)
	if !ok {
		return nil, &types.ErrorResponse{Code: types.ErrorInvalidRequest, ErrorDescription: "Invalid or expired consent token"}
	}
	var record ConsentRecord
	if err := json.Unmarshal([]byte(text), &record); err != nil || record.Request == nil || record.UserID == "" {
		return nil, &types.ErrorResponse{Code: types.ErrorInvalidRequest, ErrorDescription: "Invalid or expired consent token"}
	}
	return &record, nil
}
