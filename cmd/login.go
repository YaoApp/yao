package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/engine"
)

var loginServer string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: L("Login to remote Yao server"),
	Long:  L("Login to remote Yao server using device authorization flow"),
	Run: func(cmd *cobra.Command, args []string) {
		if loginServer == "" {
			color.Red(L("Missing --server flag\n"))
			fmt.Println("  yao login --server https://yaoagents.com")
			os.Exit(1)
		}

		serverURL := strings.TrimRight(loginServer, "/")

		// 1. Discover OAuth endpoints via well-known metadata
		endpoints, err := discoverEndpoints(serverURL)
		if err != nil {
			color.Red("  %s %s\n", L("Server discovery failed:"), err)
			os.Exit(1)
		}

		// 2. Compute deterministic client_id from machine fingerprint
		machine, err := engine.GetMachineID()
		if err != nil {
			color.Red("Failed to compute machine ID: %s\n", err)
			os.Exit(1)
		}
		clientID := machine.ID

		// 3. Register the client (idempotent for same client_id)
		if endpoints.RegistrationEndpoint != "" {
			if err := registerClient(endpoints.RegistrationEndpoint, clientID); err != nil {
				color.Red("Client registration failed: %s\n", err)
				os.Exit(1)
			}
		}

		// 4. Start device authorization
		deviceResp, err := requestDeviceAuthorization(endpoints.DeviceAuthorizationEndpoint, clientID)
		if err != nil {
			color.Red("Device authorization failed: %s\n", err)
			os.Exit(1)
		}

		// 5. Display the code to the user
		dashboard := endpoints.Dashboard
		if dashboard == "" {
			dashboard = "/admin"
		}
		verifyURI := strings.TrimRight(serverURL, "/") + dashboard + "/auth/device"
		verifyURIComplete := verifyURI + "?user_code=" + deviceResp.UserCode

		fmt.Println()
		color.White("  %s %s\n",
			L("Open:"),
			color.CyanString(verifyURIComplete))
		fmt.Println()
		color.White("  %s %s\n",
			L("Or visit:"),
			color.CyanString(verifyURI))
		color.White("  %s %s\n",
			L("Enter code:"),
			color.YellowString(deviceResp.UserCode))
		fmt.Println()

		// 6. Poll for token
		interval := deviceResp.Interval
		if interval < 5 {
			interval = 5
		}

		color.White("  %s", L("Waiting for authorization..."))
		tokenResp, err := pollForToken(endpoints.TokenEndpoint, clientID, deviceResp.DeviceCode, interval, deviceResp.ExpiresIn)
		if err != nil {
			fmt.Println()
			color.Red("\n  %s %s\n", L("Login failed:"), err)
			os.Exit(1)
		}

		// 6. Save credential
		expiresAt := ""
		if tokenResp.ExpiresIn > 0 {
			expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).UTC().Format(time.RFC3339)
		}

		cred := &Credential{
			Server:       serverURL,
			GRPCAddr:     endpoints.GRPCAddr,
			AccessToken:  tokenResp.AccessToken,
			RefreshToken: tokenResp.RefreshToken,
			Scope:        tokenResp.Scope,
			User:         parseJWTSubject(tokenResp.AccessToken),
			ExpiresAt:    expiresAt,
		}

		if err := SaveCredential(cred); err != nil {
			color.Red("\n  Failed to save credentials: %s\n", err)
			os.Exit(1)
		}

		fmt.Print("\033[2J\033[H")
		color.Green("  ✓ %s\n", L("Login successful"))
		color.White("    %s %s\n", L("Server:"), serverURL)
		if cred.GRPCAddr != "" {
			color.White("    %s %s\n", L("gRPC:"), cred.GRPCAddr)
		}
		if cred.User != "" {
			color.White("    %s %s\n", L("User:"), cred.User)
		}
		if cred.ExpiresAt != "" {
			color.White("    %s %s\n", L("Expires:"), cred.ExpiresAt)
		}
		fmt.Println()
	},
}

func init() {
	loginCmd.PersistentFlags().StringVar(&loginServer, "server", "", L("Remote Yao server URL"))
}

// --- types ---

type oauthEndpoints struct {
	RegistrationEndpoint        string `json:"registration_endpoint"`
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	TokenEndpoint               string `json:"token_endpoint"`
	RevocationEndpoint          string `json:"revocation_endpoint"`
	Dashboard                   string `json:"-"`
	GRPCAddr                    string `json:"-"`
}

type deviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type oauthError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// --- HTTP helpers ---

// discoverEndpoints fetches OAuth endpoint URLs from /.well-known/yao,
// using the openapi base prefix to construct correct API paths.
func discoverEndpoints(serverURL string) (*oauthEndpoints, error) {
	return discoverFromYaoMetadata(serverURL)
}

type yaoMetadataResponse struct {
	OpenAPI   string `json:"openapi"`
	Dashboard string `json:"dashboard"`
	GRPC      string `json:"grpc"`
}

func discoverFromYaoMetadata(serverURL string) (*oauthEndpoints, error) {
	resp, err := http.Get(serverURL + "/.well-known/yao")
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("/.well-known/yao returned %d", resp.StatusCode)
	}

	var meta yaoMetadataResponse
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("invalid /.well-known/yao response: %w", err)
	}

	base := strings.TrimRight(serverURL, "/") + meta.OpenAPI

	return &oauthEndpoints{
		RegistrationEndpoint:        base + "/oauth/register",
		DeviceAuthorizationEndpoint: base + "/oauth/device_authorization",
		TokenEndpoint:               base + "/oauth/token",
		RevocationEndpoint:          base + "/oauth/revoke",
		Dashboard:                   meta.Dashboard,
		GRPCAddr:                    meta.GRPC,
	}, nil
}

func registerClient(endpoint, clientID string) error {
	body := fmt.Sprintf(
		`{"client_id":"%s","client_name":"yao-cli","grant_types":["urn:ietf:params:oauth:grant-type:device_code"],"token_endpoint_auth_method":"none","redirect_uris":["http://localhost"]}`,
		clientID,
	)
	resp, err := http.Post(endpoint, "application/json", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	var oerr oauthError
	if json.Unmarshal(respBody, &oerr) == nil && oerr.Error == "invalid_client_metadata" {
		return nil // client already registered, idempotent
	}
	return fmt.Errorf("registration returned %d: %s", resp.StatusCode, string(respBody))
}

func requestDeviceAuthorization(endpoint, clientID string) (*deviceAuthResponse, error) {
	data := url.Values{
		"client_id": {clientID},
		"scope":     {"grpc:run grpc:stream grpc:shell grpc:mcp grpc:llm grpc:agent"},
	}
	resp, err := http.PostForm(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var oerr oauthError
		json.Unmarshal(respBody, &oerr)
		if oerr.ErrorDescription != "" {
			return nil, fmt.Errorf("%s", oerr.ErrorDescription)
		}
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result deviceAuthResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return &result, nil
}

func pollForToken(endpoint, clientID, deviceCode string, interval, expiresIn int) (*tokenResponse, error) {
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("device code expired")
		}

		data := url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"client_id":   {clientID},
			"device_code": {deviceCode},
		}

		resp, err := http.PostForm(endpoint, data)
		if err != nil {
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var tok tokenResponse
			if err := json.Unmarshal(respBody, &tok); err != nil {
				return nil, fmt.Errorf("invalid token response: %w", err)
			}
			return &tok, nil
		}

		var oerr oauthError
		json.Unmarshal(respBody, &oerr)
		switch oerr.Error {
		case "authorization_pending":
			fmt.Print(".")
			continue
		case "slow_down":
			interval += 5
			ticker.Reset(time.Duration(interval) * time.Second)
			continue
		case "expired_token":
			return nil, fmt.Errorf("device code expired")
		case "access_denied":
			return nil, fmt.Errorf("authorization denied by user")
		default:
			desc := oerr.ErrorDescription
			if desc == "" {
				desc = oerr.Error
			}
			return nil, fmt.Errorf("%s", desc)
		}
	}
	return nil, fmt.Errorf("device code expired")
}

// parseJWTSubject extracts the "sub" claim from a JWT access token
// without verifying the signature (display-only).
func parseJWTSubject(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Sub string `json:"sub"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return ""
	}
	return claims.Sub
}
