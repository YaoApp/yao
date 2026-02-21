package otp

import (
	"encoding/json"
	"fmt"
)

// Verify looks up an OTP code and returns its stored Payload.
// It does NOT consume the code — the code remains valid within its TTL.
func (s *Service) Verify(code string) (*Payload, error) {
	if code == "" {
		return nil, fmt.Errorf("code is required")
	}

	key := s.storeKey(code)
	val, ok := s.store.Get(key)
	if !ok || val == nil {
		return nil, fmt.Errorf("invalid or expired OTP code")
	}

	return coercePayload(val)
}

// coercePayload converts a store value into a *Payload.
// The store may return *Payload, map[string]interface{}, or raw JSON bytes.
func coercePayload(val interface{}) (*Payload, error) {
	switch v := val.(type) {
	case *Payload:
		return v, nil

	case Payload:
		return &v, nil

	case map[string]interface{}:
		p := &Payload{Consume: true}
		if s, ok := v["team_id"].(string); ok {
			p.TeamID = s
		}
		if s, ok := v["member_id"].(string); ok {
			p.MemberID = s
		}
		if s, ok := v["user_id"].(string); ok {
			p.UserID = s
		}
		if s, ok := v["redirect"].(string); ok {
			p.Redirect = s
		}
		if s, ok := v["scope"].(string); ok {
			p.Scope = s
		}
		switch te := v["token_expires_in"].(type) {
		case float64:
			p.TokenExpiresIn = int(te)
		case int:
			p.TokenExpiresIn = te
		case int64:
			p.TokenExpiresIn = int(te)
		}
		if b, ok := v["consume"].(bool); ok {
			p.Consume = b
		}
		return p, nil

	default:
		// JSON fallback for serialised store backends
		raw, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("unexpected OTP payload type: %T", val)
		}
		p := &Payload{Consume: true}
		if err := json.Unmarshal(raw, p); err != nil {
			return nil, fmt.Errorf("failed to decode OTP payload: %w", err)
		}
		return p, nil
	}
}
