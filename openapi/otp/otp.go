package otp

import (
	"fmt"
	"time"

	"github.com/yaoapp/gou/store"
)

// OTP is the global OTP service instance, initialized by NewService.
var OTP *Service

const (
	defaultExpiresIn  = 24 * 60 * 60 // 24 hours in seconds
	storeKeyInfix     = "oauth:otp:"
	codeAlphabet      = "23456789abcdefghjkmnpqrstuvwxyz"
	codeLength        = 12
	maxCollisionRetry = 5
)

// Service manages OTP code lifecycle.
type Service struct {
	store  store.Store
	prefix string // key prefix, e.g. "yao_:"
}

// Payload is the data stored alongside an OTP code.
type Payload struct {
	TeamID         string `json:"team_id,omitempty"`
	MemberID       string `json:"member_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
	Redirect       string `json:"redirect"`
	Scope          string `json:"scope,omitempty"`
	TokenExpiresIn int    `json:"token_expires_in,omitempty"`
	Consume        bool   `json:"consume"`
}

// GenerateParams holds the input for creating an OTP code.
type GenerateParams struct {
	TeamID         string `json:"team_id,omitempty"`
	MemberID       string `json:"member_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
	ExpiresIn      int    `json:"expires_in,omitempty"` // seconds; 0 means default (24h)
	Redirect       string `json:"redirect"`
	Scope          string `json:"scope,omitempty"`
	TokenExpiresIn int    `json:"token_expires_in,omitempty"` // access_token lifetime override (seconds); 0 means system default
	Consume        bool   `json:"consume"`                    // revoke code after login; default true
}

// LoginResult wraps user.LoginResponse with the OTP redirect path.
type LoginResult struct {
	UserID                string `json:"user_id,omitempty"`
	Subject               string `json:"subject,omitempty"`
	AccessToken           string `json:"access_token"`
	IDToken               string `json:"id_token,omitempty"`
	RefreshToken          string `json:"refresh_token,omitempty"`
	ExpiresIn             int    `json:"expires_in,omitempty"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in,omitempty"`
	TokenType             string `json:"token_type,omitempty"`
	Scope                 string `json:"scope,omitempty"`
	Redirect              string `json:"redirect"`
}

// NewService creates and registers a global OTP service.
func NewService(s store.Store, prefix string) *Service {
	OTP = &Service{store: s, prefix: prefix}
	return OTP
}

// storeKey builds a namespaced store key for the given OTP code.
func (s *Service) storeKey(code string) string {
	return fmt.Sprintf("%s%s%s", s.prefix, storeKeyInfix, code)
}

// ttl returns the effective TTL as a time.Duration.
func ttl(expiresIn int) time.Duration {
	if expiresIn <= 0 {
		expiresIn = defaultExpiresIn
	}
	return time.Duration(expiresIn) * time.Second
}
