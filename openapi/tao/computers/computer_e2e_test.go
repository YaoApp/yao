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

	"github.com/yaoapp/yao/e2e/client"
	"github.com/yaoapp/yao/e2e/env"
	"github.com/yaoapp/yao/e2e/taocloud"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	userpkg "github.com/yaoapp/yao/openapi/user"
)

func TestE2ETaoPurchaseHTTP(t *testing.T) {
	taoServer := taocloud.New()
	t.Cleanup(taoServer.Close)
	t.Setenv("TAO_ENDPOINT", taoServer.URL())
	t.Setenv("TAO_TIMEOUT", "5s")
	environment, err := env.New(t)
	if err != nil {
		t.Fatalf("environment: %v", err)
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
	login, err := userpkg.LoginByUserID(fmt.Sprint(admin["user_id"]), nil)
	if err != nil {
		t.Fatal(err)
	}
	httpClient := client.New(environment.BaseURL)
	httpClient.Token = login.AccessToken

	resp, body, err := doJSON(httpClient, http.MethodPut, "/v1/tao/account", map[string]any{"api_key": taoServer.Token})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("bind key status=%v err=%v body=%s", resp, err, body)
	}
	resp, body, err = httpClient.PostJSON("/v1/tao/computers", map[string]any{
		"resource": "flex-2c2g", "region": "bj-cn", "name": "http-host", "billing_unit": "HOUR",
		"request_params": map[string]any{"image": "tao-base-1.0"},
		"computer":       map[string]any{"name": "http-cloud", "visibility": "private"},
	})
	if err != nil || resp.StatusCode != http.StatusAccepted {
		t.Fatalf("purchase status=%v err=%v body=%s", resp, err, body)
	}
	var purchased map[string]any
	if err := json.Unmarshal(body, &purchased); err != nil {
		t.Fatal(err)
	}
	instanceID, _ := purchased["instance_id"].(string)
	if instanceID == "" || purchased["computer_status"] != "provisioning" || purchased["computer_id"] == nil {
		t.Fatalf("purchase=%v", purchased)
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
