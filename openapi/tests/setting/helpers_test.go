package setting_test

import (
	"testing"

	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/tests/testutils"
	"github.com/yaoapp/yao/setting"
)

func initSettingRegistry(t *testing.T) {
	t.Helper()
	if setting.Global == nil {
		if err := setting.Init(); err != nil {
			t.Fatalf("setting.Init: %v", err)
		}
	}
}

func obtainToken(t *testing.T, serverURL string) string {
	t.Helper()
	client := testutils.RegisterTestClient(t, "Setting Test", []string{"https://localhost/callback"})
	t.Cleanup(func() { testutils.CleanupTestClient(t, client.ClientID) })
	token := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")
	return token.AccessToken
}

func obtainRestrictedToken(t *testing.T, serverURL, scope string) string {
	t.Helper()
	client := testutils.RegisterTestClient(t, "Setting Restricted", []string{"https://localhost/callback"})
	t.Cleanup(func() { testutils.CleanupTestClient(t, client.ClientID) })

	oauthService := oauth.OAuth
	if oauthService == nil {
		t.Fatal("Global OAuth service not initialized")
	}

	token := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")
	subject, err := oauthService.Subject(client.ClientID, token.UserID)
	if err != nil {
		t.Fatalf("Failed to create subject: %v", err)
	}

	accessToken, err := oauthService.MakeAccessToken(client.ClientID, scope, subject, 3600)
	if err != nil {
		t.Fatalf("Failed to create access token: %v", err)
	}
	return accessToken
}
