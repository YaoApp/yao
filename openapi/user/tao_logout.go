package user

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
)

// taoLogoutByUserID revokes the Tao refresh token for a Yao user. It is the
// best-effort hook used by OAuth client logout. It never fails the logout.
func taoLogoutByUserID(ctx context.Context, userID string) error {
	provider, err := taoGetUserProvider()
	if err != nil {
		return err
	}
	accounts, err := provider.GetUserOAuthAccounts(ctx, userID)
	if err != nil {
		return err
	}
	var refreshToken string
	for _, account := range accounts {
		if fmt.Sprint(account["provider"]) != "tao" {
			continue
		}
		refreshToken = fmt.Sprint(account["refresh_token"])
		if refreshToken != "" {
			break
		}
	}
	if refreshToken == "" {
		return nil
	}
	tao, err := taoProvider()
	if err != nil {
		return err
	}
	creds, err := loadTaoClientCredentials()
	if err != nil || !creds.complete() {
		return nil
	}
	form := url.Values{}
	form.Set("token", refreshToken)
	form.Set("token_type_hint", "refresh_token")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(tao.Issuer, "/")+"/oauth/revoke", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(creds.ClientID, creds.ClientSecret)
	resp, err := taoHTTPClient().Do(req)
	if err != nil {
		log.Warn("Tao logout revoke failed: %v", err)
		return nil
	}
	defer resp.Body.Close()
	return nil
}

// clearTaoRefreshTokenForUser removes local Tao refresh-token custody after a
// user logout. It is best effort and never blocks the logout.
func clearTaoRefreshTokenForUser(ctx context.Context, userID string) {
	provider, err := taoGetUserProvider()
	if err != nil || provider == nil {
		return
	}
	accounts, err := provider.GetUserOAuthAccounts(ctx, userID)
	if err != nil {
		return
	}
	for _, account := range accounts {
		if fmt.Sprint(account["provider"]) != "tao" {
			continue
		}
		subject := fmt.Sprint(account["sub"])
		if subject == "" || subject == "<nil>" {
			continue
		}
		_ = provider.UpdateOAuthAccount(ctx, "tao", subject, maps.MapStrAny{"refresh_token": ""})
	}
}
