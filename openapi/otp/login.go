package otp

import (
	"context"
	"fmt"
	"strings"

	"github.com/yaoapp/yao/openapi/oauth"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/user"
	"github.com/yaoapp/yao/openapi/utils"
)

// Login verifies an OTP code, resolves the user identity, issues tokens,
// and returns the result. It does NOT set HTTP cookies.
func (s *Service) Login(code string, locale string) (*LoginResult, error) {
	payload, err := s.Verify(code)
	if err != nil {
		return nil, err
	}

	userID, teamID, err := s.resolveIdentity(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve OTP identity: %w", err)
	}

	loginCtx := &oauthtypes.LoginContext{
		Locale:     locale,
		AuthSource: "otp",
	}

	opts := &user.LoginOptions{
		SkipRefreshToken: true,
		TokenExpiresIn:   payload.TokenExpiresIn,
	}
	if payload.Scope != "" {
		opts.Scopes = strings.Fields(payload.Scope)
	}

	loginResp, err := user.LoginWithOptions(userID, teamID, loginCtx, opts)
	if err != nil {
		return nil, fmt.Errorf("OTP login failed: %w", err)
	}

	return &LoginResult{
		UserID:                loginResp.UserID,
		Subject:               loginResp.Subject,
		AccessToken:           loginResp.AccessToken,
		IDToken:               loginResp.IDToken,
		RefreshToken:          loginResp.RefreshToken,
		ExpiresIn:             loginResp.ExpiresIn,
		RefreshTokenExpiresIn: loginResp.RefreshTokenExpiresIn,
		TokenType:             loginResp.TokenType,
		Scope:                 loginResp.Scope,
		Redirect:              payload.Redirect,
	}, nil
}

// resolveIdentity determines the user_id and team_id from the OTP payload.
// When member_id is present, it resolves the user_id from the member record.
func (s *Service) resolveIdentity(payload *Payload) (userID string, teamID string, err error) {
	teamID = payload.TeamID

	if payload.UserID != "" {
		return payload.UserID, teamID, nil
	}

	if payload.MemberID == "" {
		return "", "", fmt.Errorf("payload has neither user_id nor member_id")
	}

	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		return "", "", fmt.Errorf("failed to get user provider: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	member, err := userProvider.GetMemberByMemberID(ctx, payload.MemberID)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve member %s: %w", payload.MemberID, err)
	}

	userID = utils.ToString(member["user_id"])
	if userID == "" {
		return "", "", fmt.Errorf("member %s has no associated user_id", payload.MemberID)
	}

	if teamID == "" {
		teamID = utils.ToString(member["team_id"])
	}

	return userID, teamID, nil
}
