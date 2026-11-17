package oauth

import (
	"context"
	"fmt"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const userCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// DeviceAuthorization initiates the device authorization flow (RFC 8628).
func (s *Service) DeviceAuthorization(ctx context.Context, clientID string, scope string) (*types.DeviceAuthorizationResponse, error) {
	if !s.config.Features.DeviceFlowEnabled {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorUnsupportedGrantType,
			ErrorDescription: "Device flow is not enabled",
		}
	}

	client, err := s.clientProvider.GetClientByID(ctx, clientID)
	if err != nil || client == nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidClient,
			ErrorDescription: "Invalid client",
		}
	}

	if !clientSupportsGrantType(client, types.GrantTypeDeviceCode) {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorUnauthorizedClient,
			ErrorDescription: "Client does not support device code grant",
		}
	}

	deviceCode, err := s.generateToken("dc", clientID)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate device code",
		}
	}

	userCode, err := s.generateUserCode()
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate user code",
		}
	}

	if err := s.storeDeviceCode(deviceCode, userCode, clientID, scope); err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to store device code",
		}
	}

	verificationURI := fmt.Sprintf("%s/auth/device", s.config.IssuerURL)
	verificationURIComplete := fmt.Sprintf("%s?user_code=%s", verificationURI, userCode)

	return &types.DeviceAuthorizationResponse{
		DeviceCode:              deviceCode,
		UserCode:                userCode,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURIComplete,
		ExpiresIn:               int(s.config.Token.DeviceCodeLifetime.Seconds()),
		Interval:                int(s.config.Token.DeviceCodeInterval.Seconds()),
	}, nil
}

// AuthorizeDevice allows an authenticated user to authorize a device code via user_code.
func (s *Service) AuthorizeDevice(ctx context.Context, userCode string, subject string, extraClaims ...map[string]interface{}) error {
	if !s.config.Features.DeviceFlowEnabled {
		return &types.ErrorResponse{
			Code:             types.ErrorUnsupportedGrantType,
			ErrorDescription: "Device flow is not enabled",
		}
	}

	normalized := strings.ToUpper(strings.ReplaceAll(userCode, "-", ""))
	formatted := normalized
	if len(normalized) == 8 {
		formatted = normalized[:4] + "-" + normalized[4:]
	}

	var claims map[string]interface{}
	if len(extraClaims) > 0 {
		claims = extraClaims[0]
	}
	return s.authorizeDeviceCode(formatted, subject, claims)
}

// AuthorizeDeviceForUser authorizes a device code for a Yao user. The token
// subject is scoped to the requesting device client, not the first-party
// session client, and no Team/tenant claims are propagated.
func (s *Service) AuthorizeDeviceForUser(ctx context.Context, userCode, userID string, extraClaims ...map[string]interface{}) error {
	if !s.config.Features.DeviceFlowEnabled {
		return &types.ErrorResponse{
			Code:             types.ErrorUnsupportedGrantType,
			ErrorDescription: "Device flow is not enabled",
		}
	}
	formatted, ok := s.normalizeUserCode(userCode)
	if !ok {
		return &types.ErrorResponse{Code: types.ErrorInvalidGrant, ErrorDescription: "Invalid or expired user code"}
	}
	reverseData, exists := s.store.Get(s.userCodeKey(formatted))
	if !exists {
		return &types.ErrorResponse{Code: types.ErrorInvalidGrant, ErrorDescription: "Invalid or expired user code"}
	}
	var deviceCode string
	switch v := reverseData.(type) {
	case map[string]interface{}:
		deviceCode, _ = v["device_code"].(string)
	case primitive.M:
		deviceCode, _ = v["device_code"].(string)
	}
	if deviceCode == "" {
		return &types.ErrorResponse{Code: types.ErrorServerError, ErrorDescription: "Invalid user code mapping"}
	}
	codeData, err := s.getDeviceCodeData(deviceCode)
	if err != nil {
		return err
	}
	clientID, _ := codeData["client_id"].(string)
	if clientID == "" {
		return &types.ErrorResponse{Code: types.ErrorServerError, ErrorDescription: "Invalid device code client"}
	}
	subject, err := s.Subject(clientID, userID)
	if err != nil {
		return err
	}
	var claims map[string]interface{}
	if len(extraClaims) > 0 {
		claims = extraClaims[0]
	}
	return s.authorizeDeviceCode(formatted, subject, claims)
}

func (s *Service) normalizeUserCode(userCode string) (string, bool) {
	normalized := strings.ToUpper(strings.ReplaceAll(userCode, "-", ""))
	if normalized == "" {
		return "", false
	}
	if len(normalized) == 8 {
		return normalized[:4] + "-" + normalized[4:], true
	}
	return normalized, true
}

// generateUserCode generates a user-friendly code formatted as XXXX-XXXX.
func (s *Service) generateUserCode() (string, error) {
	length := s.config.Token.UserCodeLength
	if length <= 0 {
		length = 8
	}
	raw, err := gonanoid.Generate(userCodeAlphabet, length)
	if err != nil {
		return "", err
	}
	if len(raw) == 8 {
		return raw[:4] + "-" + raw[4:], nil
	}
	return raw, nil
}

func clientSupportsGrantType(client *types.ClientInfo, grantType string) bool {
	if client == nil || len(client.GrantTypes) == 0 {
		return false
	}
	for _, gt := range client.GrantTypes {
		if gt == grantType {
			return true
		}
	}
	return false
}
