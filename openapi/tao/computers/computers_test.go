package computers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/e2e/taocloud"
	computertypes "github.com/yaoapp/yao/modules/computer/types"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	taocomputers "github.com/yaoapp/yao/openapi/tao/computers"
)

type fakeComputers struct {
	computer *computertypes.Computer
	lastReq  computertypes.CreateOrUpdateComputerRequest
}

func (f *fakeComputers) CreateOrUpdate(_ context.Context, req computertypes.CreateOrUpdateComputerRequest) (*computertypes.UpsertResult, error) {
	f.lastReq = req
	f.computer = &computertypes.Computer{ID: "cmp-test", NodeID: "node-test", InstanceID: req.InstanceID, Status: req.Status, DataDir: req.DataDir}
	return &computertypes.UpsertResult{Computer: f.computer}, nil
}
func (f *fakeComputers) Get(context.Context, string) (*computertypes.Computer, error) {
	return f.computer, nil
}

type fakeKeys struct {
	key string
	err error
}

func (f fakeKeys) ResolveTaoKey(context.Context, string) (string, error) { return f.key, f.err }

func newContext(method, path string, body []byte, userID string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if userID != "" {
		c.Set("__user_id", userID)
	}
	return c, w
}

func newService(t *testing.T) (*taocomputer.Service, *taocloud.Server, *fakeComputers) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	store := &fakeComputers{}
	return taocomputer.NewService(taocomputer.NewClient(srv.URL()), store), srv, store
}

func TestHandlerPurchaseAndValidation(t *testing.T) {
	svc, srv, store := newService(t)
	h := &taocomputers.Handler{Service: svc, Keys: fakeKeys{key: srv.Token}}
	body, _ := json.Marshal(map[string]any{
		"resource": "flex-2c2g", "region": "bj-cn", "name": "host", "billing_unit": "HOUR",
		"request_params": map[string]any{"image": "tao-base-1.0"},
		"ssh":            map[string]any{"public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI user@host"},
		"computer":       map[string]any{"name": "cloud", "visibility": "private", "data_dir": "/data", "ssh_key_ref": "vault://key"},
	})
	c, w := newContext(http.MethodPost, "/v1/tao/computers", body, "user-1")
	h.Purchase(c)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["instance_id"] == nil || out["computer_id"] != "cmp-test" {
		t.Fatalf("response=%v", out)
	}
	if store.lastReq.DataDir != "/data" || store.lastReq.Metadata["ssh_key_ref"] != "vault://key" {
		t.Fatalf("request not bound: data_dir=%q metadata=%v", store.lastReq.DataDir, store.lastReq.Metadata)
	}
	// invalid JSON
	c, w = newContext(http.MethodPost, "/v1/tao/computers", []byte("{"), "user-1")
	h.Purchase(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid status=%d", w.Code)
	}
	// no key resolver
	c, w = newContext(http.MethodPost, "/v1/tao/computers", body, "user-1")
	h.Keys = nil
	h.Purchase(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("key status=%d", w.Code)
	}
}

func TestHandlerGetAndList(t *testing.T) {
	svc, srv, _ := newService(t)
	h := &taocomputers.Handler{Service: svc, Keys: fakeKeys{key: srv.Token}}
	c, w := newContext(http.MethodGet, "/v1/tao/computers", nil, "user-1")
	h.List(c)
	if w.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", w.Code, w.Body.String())
	}
	// unknown instance maps to Tao 404
	c, w = newContext(http.MethodGet, "/v1/tao/computers/missing", nil, "user-1")
	c.Params = gin.Params{{Key: "instance_id", Value: "missing"}}
	h.Get(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get status=%d body=%s", w.Code, w.Body.String())
	}
	_ = srv
	_ = time.Second
}
