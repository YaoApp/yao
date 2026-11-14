//go:build e2e

package computers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/yaoapp/yao/e2e/client"
	"github.com/yaoapp/yao/e2e/env"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	userpkg "github.com/yaoapp/yao/openapi/user"
)

type connectResponse struct {
	ComputerID string `json:"computer_id"`
	NodeID     string `json:"node_id"`
	NodeToken  string `json:"node_token"`
	Computer   struct {
		ID     string `json:"id"`
		NodeID string `json:"node_id"`
		Status string `json:"status"`
	} `json:"computer"`
}

type computerResponse struct {
	ID     string `json:"id"`
	NodeID string `json:"node_id"`
	Status string `json:"status"`
}

func TestComputerHTTPE2E(t *testing.T) {
	environment, err := env.New(t)
	if err != nil {
		t.Fatalf("create E2E environment: %v", err)
	}
	service, ok := openapi.Server.OAuth.(*oauth.Service)
	if !ok || service == nil {
		t.Fatal("OAuth service is not available")
	}
	ctx := context.Background()
	provider, err := service.GetUserProvider()
	if err != nil {
		t.Fatalf("get user provider: %v", err)
	}
	admin, err := provider.GetUserByEmail(ctx, "admin@e2e.local")
	if err != nil {
		t.Fatalf("get admin: %v", err)
	}
	login, err := userpkg.LoginByUserID(fmt.Sprint(admin["user_id"]), nil)
	if err != nil {
		t.Fatalf("login admin: %v", err)
	}

	httpClient := client.New(environment.BaseURL)
	httpClient.Token = login.AccessToken
	anon := client.New(environment.BaseURL)
	resp, body, err := anon.PostJSON("/v1/computers/connect", map[string]interface{}{"name": "anon"})
	if err != nil {
		t.Fatalf("anonymous connect: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous connect status=%d body=%s", resp.StatusCode, body)
	}

	resp, body, err = httpClient.PostJSON("/v1/computers/connect", map[string]interface{}{
		"name":       "HTTP E2E Computer",
		"data_dir":   "/tmp/e2e-computer",
		"visibility": "private",
	})
	if err != nil {
		t.Fatalf("connect computer: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("connect status=%d body=%s", resp.StatusCode, body)
	}
	var created connectResponse
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode connect response: %v", err)
	}
	if created.ComputerID == "" || created.NodeToken == "" || created.Computer.Status != "pending" {
		t.Fatalf("unexpected connect response: %+v", created)
	}
	t.Cleanup(func() { _ = service.RevokeNodeTokenByComputer(context.Background(), created.ComputerID) })

	resp, body, err = httpClient.Do(mustRequest(http.MethodPost, environment.BaseURL+"/v1/computers/"+created.ComputerID+"/renew", `{"duration":"1h"}`))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("renew status=%v err=%v body=%s", resp, err, body)
	}

	resp, body, err = httpClient.Get("/v1/computers")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("list computers status=%v err=%v body=%s", resp, err, body)
	}

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
	resp, body, err = normalClient.Get("/v1/computers/" + created.ComputerID)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("normal user get status=%v err=%v body=%s", resp, err, body)
	}

	resp, body, err = httpClient.Get("/v1/computers/" + created.ComputerID)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("get computer status=%v err=%v body=%s", resp, err, body)
	}
	var got computerResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode computer: %v", err)
	}
	if got.NodeID != created.NodeID || got.Status != "pending" {
		t.Fatalf("unexpected computer: %+v", got)
	}

	resp, body, err = httpClient.Do(mustRequest(http.MethodPatch, environment.BaseURL+"/v1/computers/"+created.ComputerID, `{"name":"Renamed"}`))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("patch computer status=%v err=%v body=%s", resp, err, body)
	}
	resp, body, err = httpClient.Do(mustRequest(http.MethodPost, environment.BaseURL+"/v1/computers/"+created.ComputerID+"/disconnect", ""))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("disconnect status=%v err=%v body=%s", resp, err, body)
	}
	if err := json.Unmarshal(body, &got); err != nil || got.Status != "disconnected" {
		t.Fatalf("disconnect response=%s err=%v", body, err)
	}
	resp, body, err = httpClient.Do(mustRequest(http.MethodPost, environment.BaseURL+"/v1/computers/"+created.ComputerID+"/reconnect", ""))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("reconnect status=%v err=%v body=%s", resp, err, body)
	}
	if err := json.Unmarshal(body, &got); err != nil || got.Status != "pending" {
		t.Fatalf("reconnect response=%s err=%v", body, err)
	}
	for _, path := range []string{"workspaces", "runtimes", "containers"} {
		resp, body, err = httpClient.Get("/v1/computers/" + created.ComputerID + "/" + path)
		if err != nil || resp.StatusCode != http.StatusNotImplemented {
			t.Fatalf("%s status=%v err=%v body=%s", path, resp, err, body)
		}
	}
	resp, body, err = httpClient.Do(mustRequest(http.MethodDelete, environment.BaseURL+"/v1/computers/"+created.ComputerID, ""))
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status=%v err=%v body=%s", resp, err, body)
	}
	if _, err := service.ValidateNodeToken(ctx, created.NodeToken); err == nil {
		t.Fatal("node token remained valid after computer delete")
	}
}

func mustRequest(method, url, body string) *http.Request {
	req, _ := http.NewRequest(method, url, stringsReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func stringsReader(value string) *strings.Reader {
	return strings.NewReader(value)
}
