//go:build e2e

package computers_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/yaoapp/yao/e2e/client"
	"github.com/yaoapp/yao/e2e/taocloud"
	"github.com/yaoapp/yao/modules/computer"
	computertypes "github.com/yaoapp/yao/modules/computer/types"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	userpkg "github.com/yaoapp/yao/openapi/user"
)

func TestE2ECloudComputerDeleteReleasesTao(t *testing.T) {
	taoServer := deleteTaoServer
	environment := testEnvironment(t)
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
	if err := taocomputer.BindKey(context.Background(), userID, taoServer.Token); err != nil {
		t.Fatalf("bind Tao key: %v", err)
	}
	login, err := userpkg.LoginByUserID(userID, nil)
	if err != nil {
		t.Fatal(err)
	}
	httpClient := client.New(environment.BaseURL)
	httpClient.Token = login.AccessToken

	computers := computer.New()
	create := func(instanceID, name string) *computertypes.Computer {
		item, err := computers.Create(context.Background(), computertypes.CreateComputerRequest{
			InstanceID: instanceID, NodeID: "node-" + name, Name: name,
			ProviderType: "cloud", Scope: computertypes.ScopeUser,
			Visibility: computertypes.VisibilityPrivate, Status: computertypes.StatusProvisioning,
			UserID: userID,
		})
		if err != nil {
			t.Fatalf("create cloud computer: %v", err)
		}
		t.Cleanup(func() {
			if _, err := computers.Get(context.Background(), item.ID); err == nil {
				_ = computers.Delete(context.Background(), item.ID)
			}
		})
		return item
	}

	first := create("inst-delete-ok", "cloud-delete-ok")
	resp, body, err := httpClient.Do(mustRequest(http.MethodDelete, environment.BaseURL+"/v1/computers/"+first.ID, ""))
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status=%v err=%v body=%s", resp, err, body)
	}
	if _, err := computers.Get(context.Background(), first.ID); !errors.Is(err, computer.ErrNotFound) {
		t.Fatalf("computer should be gone, err=%v", err)
	}
	if !hasTaoDeleteRequest(taoServer.Requests(), "/v1/computers/inst-delete-ok") {
		t.Fatal("Tao DELETE was not called")
	}
	resp, body, err = httpClient.Do(mustRequest(http.MethodDelete, environment.BaseURL+"/v1/computers/"+first.ID, ""))
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("repeat delete status=%v err=%v body=%s", resp, err, body)
	}

	second := create("inst-delete-fail", "cloud-delete-fail")
	taoServer.FailNext(http.MethodDelete, "/v1/computers/", http.StatusServiceUnavailable, "cloud_down")
	resp, body, err = httpClient.Do(mustRequest(http.MethodDelete, environment.BaseURL+"/v1/computers/"+second.ID, ""))
	if err != nil || resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("failed release status=%v err=%v body=%s", resp, err, body)
	}
	if _, err := computers.Get(context.Background(), second.ID); err != nil {
		t.Fatalf("computer must remain after release failure: %v", err)
	}
	resp, body, err = httpClient.Do(mustRequest(http.MethodDelete, environment.BaseURL+"/v1/computers/"+second.ID, ""))
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("retry delete status=%v err=%v body=%s", resp, err, body)
	}
	if _, err := computers.Get(context.Background(), second.ID); !errors.Is(err, computer.ErrNotFound) {
		t.Fatalf("computer should be gone after retry, err=%v", err)
	}
}

func hasTaoDeleteRequest(requests []taocloud.RecordedRequest, path string) bool {
	for _, request := range requests {
		if request.Method == http.MethodDelete && request.Path == path {
			return true
		}
	}
	return false
}
