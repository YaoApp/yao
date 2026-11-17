//go:build uatmanual

package user_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/e2e/env"
)

func TestTaoRealUATLogin(t *testing.T) {
	issuer := os.Getenv("TAO_OAUTH_ISSUER")
	clientID := os.Getenv("TAO_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("TAO_OAUTH_CLIENT_SECRET")
	account := os.Getenv("TAO_UAT_ACCOUNT")
	password := os.Getenv("TAO_UAT_PASSWORD")
	require.NotEmpty(t, issuer, "TAO_OAUTH_ISSUER is required")
	require.NotEmpty(t, clientID, "TAO_OAUTH_CLIENT_ID is required")
	require.NotEmpty(t, clientSecret, "TAO_OAUTH_CLIENT_SECRET is required")
	require.NotEmpty(t, account, "TAO_UAT_ACCOUNT is required")
	require.NotEmpty(t, password, "TAO_UAT_PASSWORD is required")
	t.Setenv("TAO_OAUTH_ISSUER", issuer)
	t.Setenv("TAO_OAUTH_CLIENT_ID", clientID)
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", clientSecret)

	environment, err := env.New(t)
	require.NoError(t, err)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	httpClient := &http.Client{Jar: jar, Timeout: 30 * time.Second}
	uat := issuer

	// Browser-style Tao login with the documented UAT smoke account.
	form := url.Values{"account": {account}, "password": {password}, "next": {uat + "/oauth/authorize"}}
	req, _ := http.NewRequest(http.MethodPost, uat+"/oauth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", uat)
	req.Header.Set("Referer", uat+"/oauth/login")
	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	// Ask Yao to start a real Tao login; it returns Tao authorize URL + state.
	resp, err = httpClient.Get(environment.BaseURL + "/v1/user/oauth/tao/authorize?locale=en")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	var authResp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &authResp))
	authURL, _ := authResp["authorization_url"].(string)
	require.NotEmpty(t, authURL, string(body))
	authParsed, err := url.Parse(authURL)
	require.NoError(t, err)
	require.Equal(t, uat, authParsed.Scheme+"://"+authParsed.Host)

	// Open consent page and submit the consent form.
	resp, err = httpClient.Get(authURL)
	require.NoError(t, err)
	page, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusOK && strings.Contains(string(page), `name="consent"`) {
		consentForm := url.Values{}
		inputRE := regexp.MustCompile(`name="([^"]+)" value="([^"]*)"`)
		for _, m := range inputRE.FindAllStringSubmatch(string(page), -1) {
			consentForm.Set(m[1], m[2])
		}
		require.NotEmpty(t, consentForm.Get("client_id"))
		consentForm.Set("consent", "allow")
		req, _ = http.NewRequest(http.MethodPost, uat+"/oauth/authorize", strings.NewReader(consentForm.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", uat)
		req.Header.Set("Referer", authURL)
		resp, err = httpClient.Do(req)
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	// Yao callback establishes the session and the user must be Tao-backed.
	resp, err = httpClient.Get(environment.BaseURL + "/v1/user/profile")
	require.NoError(t, err)
	profileBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, string(profileBody))
	var profile map[string]interface{}
	require.NoError(t, json.Unmarshal(profileBody, &profile))
	require.Equal(t, "tao", fmt.Sprint(profile["identity_source"]), string(profileBody))
	require.NotEmpty(t, fmt.Sprint(profile["user_id"]), string(profileBody))
}
