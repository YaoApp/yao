package openapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// registerDeviceClient registers a device-flow-capable public client via HTTP POST to /oauth/register.
// Returns the client ID.
func registerDeviceClient(t *testing.T, serverURL, baseURL string) string {
	t.Helper()

	endpoint := serverURL + baseURL + "/oauth/register"
	req := types.DynamicClientRegistrationRequest{
		ClientName:              "device-test-client",
		RedirectURIs:            []string{"http://localhost/device-callback"},
		GrantTypes:              []string{types.GrantTypeDeviceCode, types.GrantTypeRefreshToken},
		TokenEndpointAuthMethod: types.TokenEndpointAuthNone,
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "device client registration should succeed")

	var regResp types.DynamicClientRegistrationResponse
	err = json.NewDecoder(resp.Body).Decode(&regResp)
	assert.NoError(t, err)
	assert.NotEmpty(t, regResp.ClientID)

	return regResp.ClientID
}

// registerConfidentialClient registers a confidential client with client_credentials grant.
// Returns clientID and clientSecret.
func registerConfidentialClient(t *testing.T, serverURL, baseURL string) (string, string) {
	t.Helper()

	endpoint := serverURL + baseURL + "/oauth/register"
	req := types.DynamicClientRegistrationRequest{
		ClientName:              "confidential-token-client",
		RedirectURIs:            []string{"http://localhost/callback"},
		GrantTypes:              []string{types.GrantTypeClientCredentials},
		TokenEndpointAuthMethod: types.TokenEndpointAuthBasic,
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "confidential client registration should succeed")

	var regResp types.DynamicClientRegistrationResponse
	err = json.NewDecoder(resp.Body).Decode(&regResp)
	assert.NoError(t, err)
	assert.NotEmpty(t, regResp.ClientID)
	assert.NotEmpty(t, regResp.ClientSecret)

	return regResp.ClientID, regResp.ClientSecret
}

func TestDeviceAuthorization_Success(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := openapi.Server.Config.BaseURL
	clientID := registerDeviceClient(t, serverURL, baseURL)

	endpoint := serverURL + baseURL + "/oauth/device_authorization"
	form := url.Values{}
	form.Set("client_id", clientID)

	resp, err := http.PostForm(endpoint, form)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	var devResp types.DeviceAuthorizationResponse
	err = json.Unmarshal(bodyBytes, &devResp)
	assert.NoError(t, err)

	assert.NotEmpty(t, devResp.DeviceCode)
	assert.NotEmpty(t, devResp.UserCode)
	// user_code format XXXX-XXXX (9 chars including hyphen)
	assert.Len(t, devResp.UserCode, 9)
	assert.Regexp(t, regexp.MustCompile(`^[A-Z0-9]{4}-[A-Z0-9]{4}$`), devResp.UserCode)
	assert.NotEmpty(t, devResp.VerificationURI)
	assert.Greater(t, devResp.ExpiresIn, 0)
	assert.Greater(t, devResp.Interval, 0)
}

func TestDeviceAuthorization_MissingClientID(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := openapi.Server.Config.BaseURL
	endpoint := serverURL + baseURL + "/oauth/device_authorization"

	form := url.Values{}
	// no client_id

	resp, err := http.PostForm(endpoint, form)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeviceAuthorization_InvalidClient(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := openapi.Server.Config.BaseURL
	endpoint := serverURL + baseURL + "/oauth/device_authorization"

	form := url.Values{}
	form.Set("client_id", "nonexistent")

	resp, err := http.PostForm(endpoint, form)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeviceToken_AuthorizationPending(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := openapi.Server.Config.BaseURL
	clientID := registerDeviceClient(t, serverURL, baseURL)

	// Get device code
	devAuthEndpoint := serverURL + baseURL + "/oauth/device_authorization"
	form := url.Values{}
	form.Set("client_id", clientID)

	resp, err := http.PostForm(devAuthEndpoint, form)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var devResp types.DeviceAuthorizationResponse
	err = json.NewDecoder(resp.Body).Decode(&devResp)
	assert.NoError(t, err)
	assert.NotEmpty(t, devResp.DeviceCode)

	// Poll token endpoint before user authorizes - should get authorization_pending
	tokenEndpoint := serverURL + baseURL + "/oauth/token"
	tokenForm := url.Values{}
	tokenForm.Set("grant_type", types.GrantTypeDeviceCode)
	tokenForm.Set("device_code", devResp.DeviceCode)
	tokenForm.Set("client_id", clientID)

	tokenResp, err := http.PostForm(tokenEndpoint, tokenForm)
	assert.NoError(t, err)
	defer tokenResp.Body.Close()

	// RFC 8628: authorization_pending returns 400 with error
	assert.Equal(t, http.StatusBadRequest, tokenResp.StatusCode)

	bodyBytes, err := io.ReadAll(tokenResp.Body)
	assert.NoError(t, err)

	var errResp types.ErrorResponse
	err = json.Unmarshal(bodyBytes, &errResp)
	assert.NoError(t, err)
	assert.Equal(t, types.ErrorAuthorizationPending, errResp.Code)
}

func TestDeviceToken_InvalidDeviceCode(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := openapi.Server.Config.BaseURL
	clientID := registerDeviceClient(t, serverURL, baseURL)

	tokenEndpoint := serverURL + baseURL + "/oauth/token"
	tokenForm := url.Values{}
	tokenForm.Set("grant_type", types.GrantTypeDeviceCode)
	tokenForm.Set("device_code", "bogus-invalid-device-code")
	tokenForm.Set("client_id", clientID)

	resp, err := http.PostForm(tokenEndpoint, tokenForm)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	var errResp types.ErrorResponse
	err = json.Unmarshal(bodyBytes, &errResp)
	assert.NoError(t, err)
	assert.NotEmpty(t, errResp.Code)
}

func TestDeviceFlow_EndToEnd(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := openapi.Server.Config.BaseURL

	// a. Register device client
	deviceClientID := registerDeviceClient(t, serverURL, baseURL)

	// b. POST /oauth/device_authorization -> get device_code + user_code
	devAuthEndpoint := serverURL + baseURL + "/oauth/device_authorization"
	form := url.Values{}
	form.Set("client_id", deviceClientID)

	resp, err := http.PostForm(devAuthEndpoint, form)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var devResp types.DeviceAuthorizationResponse
	err = json.NewDecoder(resp.Body).Decode(&devResp)
	assert.NoError(t, err)
	assert.NotEmpty(t, devResp.DeviceCode)
	assert.NotEmpty(t, devResp.UserCode)

	// c. Get bearer token: register confidential client, get token via client_credentials.
	// Device authorize requires a token with subject; client_credentials tokens have no subject.
	// Use ObtainAccessTokenWithRootPermission to get a token with subject for device authorize.
	confClientID, confClientSecret := registerConfidentialClient(t, serverURL, baseURL)
	tokenInfo := testutils.ObtainAccessTokenWithRootPermission(t, serverURL, confClientID, confClientSecret, "http://localhost/callback", "openid profile")
	bearerToken := tokenInfo.AccessToken

	tokenEndpoint := serverURL + baseURL + "/oauth/token"

	// d. POST /oauth/device/authorize with bearer + user_code -> assert 200
	deviceAuthorizeEndpoint := serverURL + baseURL + "/oauth/device/authorize"
	authForm := url.Values{}
	authForm.Set("user_code", devResp.UserCode)

	authReq, err := http.NewRequest("POST", deviceAuthorizeEndpoint, bytes.NewBufferString(authForm.Encode()))
	assert.NoError(t, err)
	authReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	authReq.Header.Set("Authorization", "Bearer "+bearerToken)

	authResp, err := http.DefaultClient.Do(authReq)
	assert.NoError(t, err)
	defer authResp.Body.Close()

	assert.Equal(t, http.StatusOK, authResp.StatusCode, "device authorize should succeed")

	// e. POST /oauth/token with device_code -> assert access_token returned
	dcForm := url.Values{}
	dcForm.Set("grant_type", types.GrantTypeDeviceCode)
	dcForm.Set("device_code", devResp.DeviceCode)
	dcForm.Set("client_id", deviceClientID)

	dcResp, err := http.PostForm(tokenEndpoint, dcForm)
	assert.NoError(t, err)
	defer dcResp.Body.Close()

	assert.Equal(t, http.StatusOK, dcResp.StatusCode, "device token exchange should succeed")

	var finalToken struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	err = json.NewDecoder(dcResp.Body).Decode(&finalToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, finalToken.AccessToken)
	assert.Equal(t, "Bearer", finalToken.TokenType)
}
