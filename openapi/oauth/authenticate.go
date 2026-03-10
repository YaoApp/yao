package oauth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// AuthInput contains the raw tokens extracted from the transport layer
// (HTTP headers/cookies or gRPC metadata). No framework dependency.
type AuthInput struct {
	AccessToken  string
	RefreshToken string
	SessionID    string
}

// AuthResult holds the outcome of a successful authentication.
type AuthResult struct {
	Claims          *types.TokenClaims
	Info            *types.AuthorizedInfo
	NewAccessToken  string // non-empty when token refresh occurred
	NewRefreshToken string // non-empty when token refresh occurred
}

// AuthenticateToken performs token verification and optional refresh
// without any gin/HTTP dependency. The caller is responsible for
// extracting tokens from the transport and delivering refreshed tokens
// back to the client.
func (s *Service) AuthenticateToken(input AuthInput) (*AuthResult, error) {
	token := input.AccessToken

	// API Key resolution (same as getAccessToken in guard.go)
	if s.isAPIKey(token) {
		token = s.getAccessTokenFromAPIKey(token)
	}
	token = strings.TrimPrefix(token, "Bearer ")

	if token == "" {
		return nil, fmt.Errorf("%s", types.ErrTokenMissing.Error())
	}

	var newAccessToken, newRefreshToken string

	claims, err := s.VerifyToken(token)
	if err != nil {
		expiredClaims, expErr := s.VerifyTokenAllowExpired(token)
		if expErr != nil || expiredClaims == nil {
			return nil, fmt.Errorf("%s", types.ErrInvalidToken.Error())
		}

		if !expiredClaims.ExpiresAt.IsZero() && expiredClaims.ExpiresAt.Before(time.Now()) {
			newClaims, access, refresh, refreshErr := s.refreshTokenDirect(input.RefreshToken, expiredClaims)
			if refreshErr != nil {
				if errors.Is(refreshErr, errRefreshInProgress) || errors.Is(refreshErr, errRefreshAlreadyDone) {
					claims = expiredClaims
				} else {
					log.Error("[OAuth] Token refresh failed: %v", refreshErr)
					return nil, fmt.Errorf("%s", types.ErrInvalidRefreshToken.Error())
				}
			} else {
				claims = newClaims
				newAccessToken = access
				newRefreshToken = refresh
			}
		} else {
			return nil, fmt.Errorf("%s", types.ErrInvalidToken.Error())
		}
	}

	info := s.buildAuthInfo(claims, input.SessionID)

	return &AuthResult{
		Claims:          claims,
		Info:            info,
		NewAccessToken:  newAccessToken,
		NewRefreshToken: newRefreshToken,
	}, nil
}

// refreshTokenDirect performs token rotation without any gin/HTTP dependency.
// It shares the same refreshGates concurrency control as TryRefreshToken.
// Returns (newClaims, newAccessToken, newRefreshToken, error).
func (s *Service) refreshTokenDirect(refreshToken string, expiredClaims *types.TokenClaims) (*types.TokenClaims, string, string, error) {
	if refreshToken == "" {
		return nil, "", "", fmt.Errorf("refresh token missing")
	}

	gate := &refreshGate{done: make(chan struct{})}
	if actual, loaded := refreshGates.LoadOrStore(refreshToken, gate); loaded {
		existing := actual.(*refreshGate)
		select {
		case <-existing.done:
			return nil, "", "", errRefreshAlreadyDone
		default:
			return nil, "", "", errRefreshInProgress
		}
	}

	defer func() {
		close(gate.done)
		time.AfterFunc(30*time.Second, func() {
			refreshGates.CompareAndDelete(refreshToken, gate)
		})
	}()

	refreshClaims, err := s.VerifyRefreshToken(refreshToken)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid or expired refresh token: %w", err)
	}

	var accessTTL time.Duration
	if expiredClaims != nil && !expiredClaims.IssuedAt.IsZero() && !expiredClaims.ExpiresAt.IsZero() {
		accessTTL = expiredClaims.ExpiresAt.Sub(expiredClaims.IssuedAt)
	}
	if accessTTL <= 0 {
		accessTTL = s.config.Token.AccessTokenLifetime
	}
	if accessTTL <= 0 {
		accessTTL = time.Hour
	}

	sourceClaims := expiredClaims
	if sourceClaims == nil {
		sourceClaims = refreshClaims
	}

	extraClaims := sourceClaims.Extra
	if extraClaims == nil {
		extraClaims = make(map[string]interface{})
	}
	if sourceClaims.TeamID != "" {
		extraClaims["team_id"] = sourceClaims.TeamID
	}
	if sourceClaims.TenantID != "" {
		extraClaims["tenant_id"] = sourceClaims.TenantID
	}

	s.revokeRefreshToken(refreshToken)

	var refreshRemainingSeconds int
	if !refreshClaims.ExpiresAt.IsZero() {
		refreshRemainingSeconds = int(time.Until(refreshClaims.ExpiresAt).Seconds())
		if refreshRemainingSeconds <= 0 {
			return nil, "", "", fmt.Errorf("refresh token already expired after revocation")
		}
	} else {
		refreshTTL := s.config.Token.RefreshTokenLifetime
		if refreshTTL == 0 {
			refreshTTL = 24 * time.Hour
		}
		refreshRemainingSeconds = int(refreshTTL.Seconds())
	}

	newRefreshToken, err := s.MakeRefreshToken(
		sourceClaims.ClientID,
		sourceClaims.Scope,
		sourceClaims.Subject,
		refreshRemainingSeconds,
		extraClaims,
	)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to issue new refresh token: %w", err)
	}

	newTokenStr, err := s.MakeAccessToken(
		sourceClaims.ClientID,
		sourceClaims.Scope,
		sourceClaims.Subject,
		int(accessTTL.Seconds()),
		extraClaims,
	)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to issue access token: %w", err)
	}

	newClaims, err := s.VerifyToken(newTokenStr)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to verify refreshed token: %w", err)
	}

	log.Info("[OAuth] Token rotated for subject %s (access + refresh)", sourceClaims.Subject)
	return newClaims, newTokenStr, newRefreshToken, nil
}

// buildAuthInfo constructs AuthorizedInfo directly from token claims,
// equivalent to the SetInfo+GetInfo round-trip through gin.Context.
func (s *Service) buildAuthInfo(claims *types.TokenClaims, sessionID string) *types.AuthorizedInfo {
	teamID := claims.TeamID
	tenantID := claims.TenantID

	if claims.Extra != nil {
		if teamID == "" {
			if v, ok := claims.Extra["team_id"].(string); ok && v != "" {
				teamID = v
			}
		}
		if tenantID == "" {
			if v, ok := claims.Extra["tenant_id"].(string); ok && v != "" {
				tenantID = v
			}
		}
	}

	info := &types.AuthorizedInfo{
		Subject:   claims.Subject,
		ClientID:  claims.ClientID,
		Scope:     claims.Scope,
		SessionID: sessionID,
		TeamID:    teamID,
		TenantID:  tenantID,
	}

	userID, err := s.UserID(claims.ClientID, claims.Subject)
	if err == nil && userID != "" {
		info.UserID = userID
	}

	if info.UserID == "" && claims.Extra != nil {
		if authorizerClientID, ok := claims.Extra["authorizer_client_id"].(string); ok && authorizerClientID != "" {
			if uid, err := s.UserID(authorizerClientID, claims.Subject); err == nil && uid != "" {
				info.UserID = uid
				s.copyFingerprint(authorizerClientID, claims.ClientID, claims.Subject)
			}
		}
	}

	return info
}
