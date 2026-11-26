//go:build e2e

package computers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/yaoapp/yao/e2e/client"
	"github.com/yaoapp/yao/e2e/env"
	"github.com/yaoapp/yao/e2e/taocloud"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	userpkg "github.com/yaoapp/yao/openapi/user"
)

func TestE2ETaoPurchaseHTTP(t *testing.T) {
	taoServer := taocloud.New()
	t.Cleanup(taoServer.Close)
	t.Setenv("TAO_ENDPOINT", taoServer.URL())
	t.Setenv("TAO_TIMEOUT", "5s")
	t.Setenv("YAO_SERVER_URL", "http://e2e.example:5099")
	environment, err := env.New(t)
	if err != nil {
		t.Fatalf("environment: %v", err)
	}
	enforcer, ok := acl.Global.(*acl.ACL)
	if !ok || enforcer == nil || enforcer.Scope == nil {
		t.Fatal("ACL scope manager is not loaded")
	}
	for _, tc := range []struct {
		method string
		path   string
		scope  string
	}{
		{http.MethodGet, "/products", "team:owner"},
		{http.MethodGet, "/products", "team:member"},
		{http.MethodGet, "/tao/computers", "team:owner"},
		{http.MethodPost, "/tao/computers", "team:owner"},
		{http.MethodGet, "/tao/computers/tasks/task_1", "team:owner"},
	} {
		decision := enforcer.Scope.Check(&acl.AccessRequest{Method: tc.method, Path: tc.path, Scopes: []string{tc.scope}})
		if !decision.Allowed {
			t.Fatalf("scope config %s %s via %s: %s missing=%v", tc.method, tc.path, tc.scope, decision.Reason, decision.MissingScopes)
		}
	}
	service, ok := openapi.Server.OAuth.(*oauth.Service)
	if !ok || service == nil {
		t.Fatal("OAuth service is not available")
	}
	provider, err := service.GetUserProvider()
	if err != nil {
		t.Fatal(err)
	}
	admin, err := provider.GetUserByEmail(context.Background(), "admin@e2e.local")
	if err != nil {
		t.Fatal(err)
	}
	userID := fmt.Sprint(admin["user_id"])
	originalIdentitySource := fmt.Sprint(admin["identity_source"])
	if err := provider.UpdateUser(context.Background(), userID, map[string]interface{}{"identity_source": "tao"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = provider.UpdateUser(context.Background(), userID, map[string]interface{}{"identity_source": originalIdentitySource})
	})
	login, err := userpkg.LoginByUserID(userID, nil)
	if err != nil {
		t.Fatal(err)
	}
	httpClient := client.New(environment.BaseURL)
	httpClient.Token = login.AccessToken

	resp, body, err := doJSON(httpClient, http.MethodPut, "/v1/tao/account", map[string]any{"api_key": taoServer.Token})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("bind key status=%v err=%v body=%s", resp, err, body)
	}
	// Live product list: no local products table involved.
	resp, body, err = httpClient.Get("/v1/products?source=tao&policy=paid&status=enabled")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("products status=%v err=%v body=%s", resp, err, body)
	}
	if !bytes.Contains(body, []byte(`"resource":"flex-2c2g"`)) || !bytes.Contains(body, []byte(`"billing_units"`)) ||
		!bytes.Contains(body, []byte(`"duration_tiers"`)) || !bytes.Contains(body, []byte(`"images"`)) {
		t.Fatalf("products body missing live fields: %s", body)
	}

	purchaseBody := map[string]any{
		"resource": "flex-2c2g", "region": "bj-cn", "name": "http-host", "billing_unit": "HOUR",
		"request_params": map[string]any{"image": "tao-base-1.0"},
		"computer":       map[string]any{"name": "http-cloud", "visibility": "private"},
	}
	resp, body, err = httpClient.PostJSON("/v1/tao/computers", purchaseBody)
	if err != nil || resp.StatusCode != http.StatusAccepted {
		t.Fatalf("purchase status=%v err=%v body=%s", resp, err, body)
	}
	var purchased map[string]any
	if err := json.Unmarshal(body, &purchased); err != nil {
		t.Fatal(err)
	}
	taskID, _ := purchased["task_id"].(string)
	if taskID == "" || purchased["status"] != "queued" {
		t.Fatalf("purchase=%v", purchased)
	}

	// The task is processed by the server-side worker, never by CUI.
	var task map[string]any
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		resp, body, err = httpClient.Get("/v1/tao/computers/tasks/" + taskID)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("task status=%v err=%v body=%s", resp, err, body)
		}
		_ = json.Unmarshal(body, &task)
		if task["status"] == "succeeded" || task["status"] == "failed" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if task["status"] != "succeeded" && task["status"] != "failed" {
		t.Fatalf("task did not reach terminal state: %v", task)
	}
	if task["status"] == "failed" {
		if task["error"] == nil || task["error"] == "" {
			t.Fatalf("failed task must carry the real error: %v", task)
		}
	}
	instanceID, _ := task["instance_id"].(string)
	if instanceID == "" {
		t.Fatalf("task should persist the Tao instance before Tai: %v", task)
	}
	if computerID, _ := task["computer_id"].(string); computerID != "" {
		resp, body, err = httpClient.Get("/v1/computers/" + computerID)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("computer detail status=%v err=%v body=%s", resp, err, body)
		}
		if !bytes.Contains(body, []byte(`"resource":"flex-2c2g"`)) || !bytes.Contains(body, []byte(`"image":"tao-base-1.0"`)) {
			t.Fatalf("computer purchase snapshot missing: %s", body)
		}
	}
	taoServer.SetInstanceStatus(instanceID, "running", "")
	for _, action := range []string{"stop", "start"} {
		resp, body, err = doJSON(httpClient, http.MethodPost, "/v1/tao/computers/"+instanceID+"/"+action, nil)
		if err != nil || resp.StatusCode != http.StatusAccepted {
			t.Fatalf("%s status=%v err=%v body=%s", action, resp, err, body)
		}
	}
	resp, body, err = httpClient.Get("/v1/tao/computers")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("list status=%v err=%v body=%s", resp, err, body)
	}
	resp, body, err = doJSON(httpClient, http.MethodDelete, "/v1/tao/computers/"+instanceID, nil)
	if err != nil || resp.StatusCode != http.StatusAccepted {
		t.Fatalf("delete status=%v err=%v body=%s", resp, err, body)
	}
	resp, body, err = doJSON(httpClient, http.MethodDelete, "/v1/tao/account", nil)
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("unbind status=%v err=%v body=%s", resp, err, body)
	}
}

func doJSON(c *client.Client, method, path string, payload any) (*http.Response, []byte, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return c.Do(req)
}
