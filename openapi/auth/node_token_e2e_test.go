//go:build e2e

package auth_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/e2e/client"
	"github.com/yaoapp/yao/e2e/env"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	userpkg "github.com/yaoapp/yao/openapi/user"
)

// TestNodeTokenE2E drives the node-token HTTP surface against the isolated E2E app.
func TestNodeTokenE2E(t *testing.T) {
	environment, err := env.New(t)
	if err != nil {
		t.Fatalf("create E2E environment: %v", err)
	}
	service, ok := openapi.Server.OAuth.(*oauth.Service)
	if !ok || service == nil {
		t.Fatal("OAuth service is not available")
	}
	ctx := context.Background()

	// Create a computer owned by the seeded admin so rotate/revoke pass ownership.
	computerID := "cmp_http_e2e"
	computerModel := model.Select("__yao.computers")
	if computerModel == nil {
		t.Fatal("computer model not loaded")
	}
	columns := []string{"id", "node_id", "name", "provider_type", "scope", "status"}
	values := [][]interface{}{{computerID, "node-httpe2e1", "HTTP E2E", "local", "user", "offline"}}
	if err := computerModel.Insert(columns, values); err != nil {
		t.Fatalf("insert computer: %v", err)
	}
	t.Cleanup(func() { _ = computerModel.Delete(computerID) })

	provider, err := service.GetUserProvider()
	if err != nil {
		t.Fatalf("get user provider: %v", err)
	}
	admin, err := provider.GetUserByEmail(ctx, "admin@e2e.local")
	if err != nil {
		t.Fatalf("get admin: %v", err)
	}
	adminID := fmt.Sprint(admin["user_id"])
	login, err := userpkg.LoginByUserID(adminID, nil)
	if err != nil {
		t.Fatalf("login admin: %v", err)
	}
	if login.AccessToken == "" {
		t.Fatal("admin login returned empty access token")
	}

	// Rotate must reject unauthenticated requests.
	anonymous := client.New(environment.BaseURL)
	resp, body, err := anonymous.PostJSON("/v1/auth/node-tokens/rotate", map[string]string{"computer_id": computerID})
	if err != nil {
		t.Fatalf("anonymous rotate: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous rotate status=%d body=%s", resp.StatusCode, body)
	}

	// Rotate must reject a non-owner non-admin user.
	normal, err := provider.GetUserByEmail(ctx, "user@e2e.local")
	if err != nil {
		t.Fatalf("get normal user: %v", err)
	}
	normalLogin, err := userpkg.LoginByUserID(fmt.Sprint(normal["user_id"]), nil)
	if err != nil {
		t.Fatalf("login normal user: %v", err)
	}
	normalClient := client.New(environment.BaseURL)
	normalClient.Token = normalLogin.AccessToken
	resp, body, err = normalClient.PostJSON("/v1/auth/node-tokens/rotate", map[string]string{"computer_id": computerID})
	if err != nil {
		t.Fatalf("normal user rotate: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("normal user rotate status=%d body=%s", resp.StatusCode, body)
	}

	token, err := service.IssueNodeToken(ctx, oauth.IssueRequest{
		ComputerID: "cmp_http_e2e",
		NodeID:     "node-httpe2e1",
		Scope:      "user",
		Visibility: "private",
		UserID:     "user_http_e2e",
		TeamID:     "team_http_e2e",
	})
	if err != nil {
		t.Fatalf("issue node token: %v", err)
	}

	httpClient := client.New(environment.BaseURL)
	httpClient.Token = token.Token
	resp, body, err = httpClient.PostJSON("/v1/auth/node-tokens/refresh", nil)
	if err != nil {
		t.Fatalf("HTTP refresh: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HTTP refresh status=%d body=%s", resp.StatusCode, body)
	}
	var refreshed oauth.NodeToken
	if err := json.Unmarshal(body, &refreshed); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if refreshed.Token == "" {
		t.Fatal("HTTP refresh returned empty token")
	}
	row, err := computerModel.Find(computerID, model.QueryParam{})
	if err != nil || fmt.Sprint(row["status"]) != "online" {
		t.Fatalf("computer status=%v err=%v", row["status"], err)
	}
	if _, err := service.ValidateNodeToken(ctx, token.Token); err == nil {
		t.Fatal("old token remained valid after HTTP refresh")
	}
	info, err := service.ValidateNodeToken(ctx, refreshed.Token)
	if err != nil {
		t.Fatalf("refreshed token invalid: %v", err)
	}
	if err := service.RevokeNodeToken(ctx, info.JTI); err != nil {
		t.Fatalf("revoke node token: %v", err)
	}
	if _, err := service.ValidateNodeToken(ctx, refreshed.Token); err == nil {
		t.Fatal("revoked token remained valid")
	}
}
