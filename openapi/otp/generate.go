package otp

import (
	"fmt"

	nanoid "github.com/matoous/go-nanoid/v2"
)

// Create generates a new OTP code, stores the payload, and returns the code.
// It validates required fields and retries on NanoID collision.
func (s *Service) Create(params *GenerateParams) (string, error) {
	if err := validateCreateParams(params); err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"redirect": params.Redirect,
		"consume":  params.Consume,
	}
	if params.TeamID != "" {
		data["team_id"] = params.TeamID
	}
	if params.MemberID != "" {
		data["member_id"] = params.MemberID
	}
	if params.UserID != "" {
		data["user_id"] = params.UserID
	}
	if params.Scope != "" {
		data["scope"] = params.Scope
	}
	if params.TokenExpiresIn != 0 {
		data["token_expires_in"] = params.TokenExpiresIn
	}

	for i := 0; i < maxCollisionRetry; i++ {
		code, err := nanoid.Generate(codeAlphabet, codeLength)
		if err != nil {
			return "", fmt.Errorf("failed to generate OTP code: %w", err)
		}

		key := s.storeKey(code)
		if s.store.Has(key) {
			continue
		}

		if err := s.store.Set(key, data, ttl(params.ExpiresIn)); err != nil {
			return "", fmt.Errorf("failed to store OTP code: %w", err)
		}
		return code, nil
	}

	return "", fmt.Errorf("failed to generate unique OTP code after %d attempts", maxCollisionRetry)
}

func validateCreateParams(p *GenerateParams) error {
	if p == nil {
		return fmt.Errorf("params is required")
	}
	if p.UserID == "" && p.MemberID == "" {
		return fmt.Errorf("user_id or member_id is required")
	}
	if p.MemberID != "" && p.TeamID == "" {
		return fmt.Errorf("team_id is required when member_id is set")
	}
	if p.Redirect == "" {
		return fmt.Errorf("redirect is required")
	}
	return nil
}
