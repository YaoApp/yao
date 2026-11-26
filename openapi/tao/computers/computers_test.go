package computers_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/e2e/taocloud"
	computertypes "github.com/yaoapp/yao/modules/computer/types"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	taocomputers "github.com/yaoapp/yao/openapi/tao/computers"
	"golang.org/x/crypto/ssh"
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

func testControlKey() *taocomputer.ControlKey {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(privateKey)
	block, _ := ssh.MarshalPrivateKey(privateKey, "")
	return &taocomputer.ControlKey{
		KeyID: "test-control", PublicKey: string(ssh.MarshalAuthorizedKey(signer.PublicKey())),
		PrivateKey: string(pem.EncodeToMemory(block)),
	}
}

func newService(t *testing.T) (*taocomputer.Service, *taocloud.Server, *fakeComputers) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	store := &fakeComputers{}
	return taocomputer.NewService(taocomputer.NewClient(srv.URL()), store).WithControlKey(testControlKey()).WithServerURL("https://yao.example.com"), srv, store
}

func TestHandlerPurchaseAndValidation(t *testing.T) {
	svc, srv, _ := newService(t)
	tasks := newMemTaskStore()
	svc.WithCloudTasks(tasks).WithTaskKeyResolver(fakeKeys{key: srv.Token})
	h := &taocomputers.Handler{Service: svc, Keys: fakeKeys{key: srv.Token}}
	body, _ := json.Marshal(map[string]any{
		"resource": "flex-2c2g", "region": "bj-cn", "name": "host", "billing_unit": "HOUR",
		"request_params": map[string]any{"image": "tao-base-1.0"},
		"computer":       map[string]any{"name": "cloud", "visibility": "private", "data_dir": "/data"},
	})
	c, w := newContext(http.MethodPost, "/v1/tao/computers", body, "user-1")
	h.Purchase(c)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	taskID, _ := out["task_id"].(string)
	if taskID == "" || out["status"] != taocomputer.TaskStatusQueued {
		t.Fatalf("response=%v", out)
	}
	task, err := tasks.Get(context.Background(), taskID)
	if err != nil {
		t.Fatalf("task not stored: %v", err)
	}
	task.Status = taocomputer.TaskStatusFailed
	task.InstanceID = "inst-existing"
	if err := tasks.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	c, w = newContext(http.MethodPost, "/v1/tao/computers", body, "user-1")
	h.Purchase(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("instance conflict status=%d body=%s", w.Code, w.Body.String())
	}
	task.Status = taocomputer.TaskStatusQueued
	task.InstanceID = ""
	if err := tasks.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	// The same server-side intent returns the same task without a second purchase.
	c, w = newContext(http.MethodPost, "/v1/tao/computers", body, "user-1")
	h.Purchase(c)
	var duplicate map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &duplicate)
	if w.Code != http.StatusAccepted || duplicate["task_id"] != taskID {
		t.Fatalf("duplicate purchase status=%d body=%s", w.Code, w.Body.String())
	}
	// Task lookup is authenticated by session, not by the Tao managed key.
	c, w = newContext(http.MethodGet, "/v1/tao/computers/tasks/"+taskID, nil, "user-1")
	c.Params = gin.Params{{Key: "task_id", Value: taskID}}
	h.GetTask(c)
	if w.Code != http.StatusOK {
		t.Fatalf("task get status=%d body=%s", w.Code, w.Body.String())
	}
	c, w = newContext(http.MethodGet, "/v1/tao/computers/tasks/"+taskID, nil, "other")
	c.Params = gin.Params{{Key: "task_id", Value: taskID}}
	h.GetTask(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("foreign task status=%d body=%s", w.Code, w.Body.String())
	}
	// user SSH key is rejected: control key is managed by Yao
	userSSH, _ := json.Marshal(map[string]any{
		"resource": "flex-2c2g", "region": "bj-cn", "name": "host", "billing_unit": "HOUR",
		"request_params": map[string]any{"image": "tao-base-1.0"},
		"ssh":            map[string]any{"public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI user@host"},
		"computer":       map[string]any{"name": "cloud"},
	})
	c, w = newContext(http.MethodPost, "/v1/tao/computers", userSSH, "user-1")
	h.Purchase(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("user ssh status=%d body=%s", w.Code, w.Body.String())
	}

	// structurally valid JSON that fails server-side purchase validation
	invalidIntent, _ := json.Marshal(map[string]any{"resource": "flex-2c2g"})
	c, w = newContext(http.MethodPost, "/v1/tao/computers", invalidIntent, "user-1")
	h.Purchase(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid intent status=%d body=%s", w.Code, w.Body.String())
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

func TestHandlerActionsAndDelete(t *testing.T) {
	svc, srv, store := newService(t)
	h := &taocomputers.Handler{Service: svc, Keys: fakeKeys{key: srv.Token}}
	result, err := svc.Purchase(context.Background(), srv.Token, taocomputer.PurchaseRequest{
		Resource: "flex-2c2g", Region: "bj-cn", Name: "host", BillingUnit: "HOUR",
		RequestParams: map[string]any{"image": "tao-base-1.0"},
		Computer:      taocomputer.ComputerSpec{Name: "cloud", Visibility: "private"},
		UserID:        "user-1",
	})
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}
	instanceID := result.Instance.InstanceID
	srv.SetInstanceStatus(instanceID, "running", "")
	c, w := newContext(http.MethodGet, "/v1/tao/computers/"+instanceID, nil, "user-1")

	c, w = newContext(http.MethodPost, "/v1/tao/computers/"+instanceID+"/stop", nil, "user-1")
	c.Params = gin.Params{{Key: "instance_id", Value: instanceID}}
	h.Action(c)
	if w.Code != http.StatusAccepted {
		t.Fatalf("stop status=%d body=%s", w.Code, w.Body.String())
	}
	if store.lastReq.Status != computertypes.StatusSuspended {
		t.Fatalf("store=%+v", store.lastReq)
	}

	c, w = newContext(http.MethodPost, "/v1/tao/computers/"+instanceID+"/start", nil, "user-1")
	c.Params = gin.Params{{Key: "instance_id", Value: instanceID}}
	h.Action(c)
	if w.Code != http.StatusAccepted {
		t.Fatalf("start status=%d body=%s", w.Code, w.Body.String())
	}

	c, w = newContext(http.MethodPost, "/v1/tao/computers/"+instanceID+"/bad", nil, "user-1")
	c.Params = gin.Params{{Key: "instance_id", Value: instanceID}}
	h.Action(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad action status=%d body=%s", w.Code, w.Body.String())
	}

	c, w = newContext(http.MethodDelete, "/v1/tao/computers/"+instanceID, nil, "user-1")
	c.Params = gin.Params{{Key: "instance_id", Value: instanceID}}
	h.Delete(c)
	if w.Code != http.StatusAccepted {
		t.Fatalf("delete status=%d body=%s", w.Code, w.Body.String())
	}
	if store.lastReq.Status != computertypes.StatusReleased {
		t.Fatalf("store=%+v", store.lastReq)
	}
}
