package oauth

import (
	"context"
	"fmt"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/yao/openapi/oauth/types"
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
