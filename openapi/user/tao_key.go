package user

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
)

// taoEnsureManagedKey creates or reuses the Tao managed sk-tao key for a user.
// The plaintext key is only returned on first creation, so it must be persisted
// in user.tao_api_key immediately.
func taoEnsureManagedKey(ctx context.Context, userID, accessToken string) error {
	if userID == "" || accessToken == "" {
		return fmt.Errorf("user id and access token are required")
	}
	provider, err := taoGetUserProvider()
	if err != nil {
		return err
	}
	u, err := provider.GetUserForAuth(ctx, userID, "user_id")
	if err == nil && u != nil {
		if key, _ := u["tao_api_key"].(string); key != "" {
			return nil
		}
	}
	tao, err := taoProvider()
	if err != nil {
		return err
	}
	payload := map[string]interface{}{"name": "yao-managed", "managed": true, "origin": "server"}
	body, _ := json.Marshal(payload)
	endpoint := strings.TrimRight(tao.Issuer, "/") + "/api/v1/keys"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := taoHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Key  string `json:"key"`
		Item struct {
			ID string `json:"id"`
		} `json:"item"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("decode managed key: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("managed key status %d", resp.StatusCode)
	}
	if out.Key == "" {
		return fmt.Errorf("managed key plaintext was not returned; local key is missing")
	}
	return taocomputer.BindKey(ctx, userID, out.Key)
}

// EnsureManagedKeyForUser ensures the user's managed Tao key exists before a
// Tao-backed API call. If the key is missing, it refreshes the stored Tao token
// and creates the key; it never returns the plaintext key.
func EnsureManagedKeyForUser(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user id is required")
	}
	provider, err := taoGetUserProvider()
	if err != nil || provider == nil {
		return fmt.Errorf("user provider is not available")
	}
	if user, err := provider.GetUserForAuth(ctx, userID, "user_id"); err == nil && user != nil {
		if key, _ := user["tao_api_key"].(string); key != "" {
			return nil
		}
	}
	token, err := taoRefreshUserToken(ctx, userID)
	if err != nil {
		return err
	}
	return taoEnsureManagedKey(ctx, userID, token.AccessToken)
}
