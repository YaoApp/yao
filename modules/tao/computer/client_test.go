package computer_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yaoapp/yao/e2e/taocloud"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
)

const testTAOKey = "sk-tao-e2e"

func validCreate(resource string) taocomputer.CreateInstanceRequest {
	return taocomputer.CreateInstanceRequest{
		Resource: resource, Region: "bj-cn", Name: "host", BillingUnit: "HOUR",
		RequestParams: map[string]any{"image": "tao-base-1.0"},
		SSH:           &taocomputer.SSHRequest{PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI user@host"},
	}
}

func TestClientCreateGetList(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())

	instance, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if instance.InstanceID == "" || instance.Status != "pending" {
		t.Fatalf("unexpected instance: %+v", instance)
	}
	got, err := client.Get(context.Background(), testTAOKey, instance.InstanceID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != "running" {
		t.Fatalf("auto-run status=%s", got.Status)
	}
	items, err := client.List(context.Background(), testTAOKey, taocomputer.ListOptions{Status: "running"})
	if err != nil || len(items) != 1 {
		t.Fatalf("List: items=%d err=%v", len(items), err)
	}
}

func TestClientErrors(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	if _, err := client.Get(context.Background(), "bad-key", "missing"); err == nil {
		t.Fatal("expected auth error")
	}
	if _, err := client.Create(context.Background(), testTAOKey, validCreate("missing")); err == nil {
		t.Fatal("expected unsupported resource error")
	}
	var apiErr *taocomputer.APIError
	if _, err := client.Create(context.Background(), testTAOKey, taocomputer.CreateInstanceRequest{}); err == nil || !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientRequiresEndpoint(t *testing.T) {
	client := taocomputer.NewClient("")
	if _, err := client.Get(context.Background(), testTAOKey, "x"); err == nil {
		t.Fatal("expected endpoint error")
	}
}

func TestClientCatalog(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	catalog, err := taocomputer.NewClient(srv.URL()).GetCatalog(context.Background(), testTAOKey)
	if err != nil || len(catalog.Services) == 0 || catalog.Services[0].ID != "computer" {
		t.Fatalf("catalog=%+v err=%v", catalog, err)
	}
}

func TestClientListOptionsAndCatalogError(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	items, err := client.List(context.Background(), testTAOKey, taocomputer.ListOptions{Region: "bj-cn", Page: 1, Size: 10})
	if err != nil || len(items) != 0 {
		t.Fatalf("items=%d err=%v", len(items), err)
	}
	if _, err := client.GetCatalog(context.Background(), "bad"); err != nil {
		t.Fatalf("catalog is public: %v", err)
	}
	var apiErr *taocomputer.APIError
	if err := (&taocomputer.APIError{Status: 418, Body: []byte("x")}); err == nil {
		t.Fatal("unreachable")
	} else if err.Error() == "" {
		t.Fatal("empty api error")
	}
	_ = apiErr
}

func TestNewClientEnv(t *testing.T) {
	t.Setenv("TAO_ENDPOINT", "http://example.test/")
	t.Setenv("TAO_TIMEOUT", "2s")
	client := taocomputer.NewClient("")
	if client.Endpoint != "http://example.test" || client.HTTP.Timeout != 2*time.Second {
		t.Fatalf("client=%+v timeout=%v", client, client.HTTP.Timeout)
	}
	t.Setenv("TAO_TIMEOUT", "bad")
	client = taocomputer.NewClient("")
	if client.HTTP.Timeout == 0 {
		t.Fatal("default timeout missing")
	}
}

func TestClientServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	if _, err := taocomputer.NewClient(srv.URL).GetCatalog(context.Background(), testTAOKey); err == nil {
		t.Fatal("expected server error")
	}
}

func TestClientBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()
	if _, err := taocomputer.NewClient(srv.URL).Get(context.Background(), testTAOKey, "x"); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestClientActionsAndDelete(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	srv.SetInstanceStatus(created.InstanceID, "running", "")
	started, err := client.Action(context.Background(), testTAOKey, created.InstanceID, "start")
	if err != nil || started.Status != "running" {
		t.Fatalf("start=%+v err=%v", started, err)
	}
	stopped, err := client.Action(context.Background(), testTAOKey, created.InstanceID, "stop")
	if err != nil || stopped.Status != "stopped" {
		t.Fatalf("stop=%+v err=%v", stopped, err)
	}
	started, err = client.Action(context.Background(), testTAOKey, created.InstanceID, "start")
	if err != nil || started.Status != "running" {
		t.Fatalf("start2=%+v err=%v", started, err)
	}
	rebooted, err := client.Action(context.Background(), testTAOKey, created.InstanceID, "reboot")
	if err != nil || rebooted.Status != "pending" {
		t.Fatalf("reboot=%+v err=%v", rebooted, err)
	}
	if _, err := client.Action(context.Background(), testTAOKey, created.InstanceID, "bad"); err == nil {
		t.Fatal("expected invalid action")
	}
	if err := client.Delete(context.Background(), testTAOKey, created.InstanceID); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Get(context.Background(), testTAOKey, created.InstanceID); err == nil {
		t.Fatal("expected deleted 404")
	}
}
