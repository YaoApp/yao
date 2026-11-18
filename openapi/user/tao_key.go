package user

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/yaoapp/kun/maps"
)

// taoEnsureManagedKey creates, reuses, or rotates the Tao managed service key
// for one Yao user. Tao scopes managed keys by the caller's OAuth client_id;
// the key belongs to this deployment, while yao_user.tao_api_key stores the
// per-user plaintext (AES encrypted by the model).
func taoEnsureManagedKey(ctx context.Context, userID, accessToken string) error {
	if userID == "" || accessToken == "" {
		return fmt.Errorf("user id and access token are required")
	}
	provider, err := taoGetUserProvider()
	if err != nil || provider == nil {
		return fmt.Errorf("user provider is not available")
	}
	if u, err := provider.GetUserForAuth(ctx, userID, "user_id"); err == nil && u != nil {
		if key, _ := u["tao_api_key"].(string); key != "" {
			return nil
		}
	}

	key, err := taoCreateManagedKey(ctx, accessToken)
	if err != nil {
		return err
	}
	if key == "" {
		// Tao reports that this (user, client) namespace already has a managed
		// key but cannot replay the plaintext. Rotate it and store the new key.
		key, err = taoRotateManagedKey(ctx, accessToken)
		if err != nil {
			return err
		}
	}
	return saveTaoManagedKey(ctx, userID, key)
}

// EnsureManagedKeyForUser ensures the user's managed Tao key exists before a
// Tao-backed API call. The caller must pass the actual billing user.
func EnsureManagedKeyForUser(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user id is required")
	}
	provider, err := taoGetUserProvider()
	if err != nil || provider == nil {
		return fmt.Errorf("user provider is not available")
	}
	if u, err := provider.GetUserForAuth(ctx, userID, "user_id"); err == nil && u != nil {
		if key, _ := u["tao_api_key"].(string); key != "" {
			return nil
		}
	}
	token, err := taoRefreshUserToken(ctx, userID)
	if err != nil {
		return err
	}
	return taoEnsureManagedKey(ctx, userID, token.AccessToken)
}

func taoCreateManagedKey(ctx context.Context, accessToken string) (string, error) {
	tao, err := taoProvider()
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(map[string]interface{}{
		"name": "yao-managed", "managed": true, "origin": "server",
	})
	endpoint := strings.TrimRight(tao.Issuer, "/") + "/api/v1/keys"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := taoHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("managed key status %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		Key     string `json:"key"`
		Created bool   `json:"created"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decode managed key: %w", err)
	}
	return out.Key, nil
}

func taoRotateManagedKey(ctx context.Context, accessToken string) (string, error) {
	tao, err := taoProvider()
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(map[string]string{"name": "yao-managed"})
	endpoint := strings.TrimRight(tao.Issuer, "/") + "/api/v1/keys/managed/rotate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := taoHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("managed key rotate status %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decode managed key rotate: %w", err)
	}
	if out.Key == "" {
		return "", fmt.Errorf("managed key rotate did not return plaintext")
	}
	return out.Key, nil
}

func saveTaoManagedKey(ctx context.Context, userID, key string) error {
	provider, err := taoGetUserProvider()
	if err != nil || provider == nil {
		return fmt.Errorf("user provider is not available")
	}
	return provider.UpdateUser(ctx, userID, maps.MapStrAny{"tao_api_key": key})
}
