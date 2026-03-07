package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/tai/registry"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTest() func() {
	r := registry.NewForTest()
	registry.SetGlobalForTest(r)

	origAuth := authenticateBearer
	authenticateBearer = func(token string) (registry.AuthInfo, error) {
		return registry.AuthInfo{
			Subject:  "sub-001",
			UserID:   "user-alice",
			ClientID: "tai-abc123",
			Scope:    "tai:connect",
			TeamID:   "team-dev",
		}, nil
	}

	return func() {
		authenticateBearer = origAuth
		registry.SetGlobalForTest(nil)
	}
}

func jsonBody(v interface{}) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

func TestHandleRegister_Success(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	body := registerRequest{
		TaiID:        "tai-abc123",
		MachineID:    "m-001",
		Version:      "0.2.0",
		Addr:         "192.168.1.100",
		Ports:        map[string]int{"grpc": 9100, "http": 8080},
		Capabilities: map[string]bool{"docker": true, "host_exec": false},
		System: registry.SystemInfo{
			OS: "linux", Arch: "amd64", Hostname: "docker-host-01", NumCPU: 16,
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/tai-nodes/register", jsonBody(body))
	c.Request.Header.Set("Authorization", "Bearer test-token")
	c.Request.Header.Set("Content-Type", "application/json")

	HandleRegister(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "registered" {
		t.Errorf("status = %v, want registered", resp["status"])
	}
	if resp["tai_id"] != "tai-abc123" {
		t.Errorf("tai_id = %v, want tai-abc123", resp["tai_id"])
	}
	if _, ok := resp["remote_ip"]; !ok {
		t.Error("response missing remote_ip")
	}

	snap, ok := registry.Global().Get("tai-abc123")
	if !ok {
		t.Fatal("node not found in registry after register")
	}
	if snap.Mode != "direct" {
		t.Errorf("Mode = %q, want direct", snap.Mode)
	}
	if snap.System.OS != "linux" {
		t.Errorf("System.OS = %q, want linux", snap.System.OS)
	}
	if snap.Auth.UserID != "user-alice" {
		t.Errorf("Auth.UserID = %q, want user-alice", snap.Auth.UserID)
	}
}

func TestHandleRegister_MissingAuth(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/tai-nodes/register", jsonBody(registerRequest{TaiID: "x"}))
	c.Request.Header.Set("Content-Type", "application/json")

	HandleRegister(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleRegister_MissingTaiID(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/tai-nodes/register", jsonBody(registerRequest{}))
	c.Request.Header.Set("Authorization", "Bearer test-token")
	c.Request.Header.Set("Content-Type", "application/json")

	HandleRegister(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleHeartbeat_Success(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	reg := registry.Global()
	reg.Register(&registry.TaiNode{
		TaiID: "tai-abc123",
		Mode:  "direct",
		Auth:  registry.AuthInfo{ClientID: "tai-abc123"},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/tai-nodes/heartbeat",
		jsonBody(heartbeatRequest{TaiID: "tai-abc123"}))
	c.Request.Header.Set("Authorization", "Bearer test-token")
	c.Request.Header.Set("Content-Type", "application/json")

	HandleHeartbeat(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandleHeartbeat_WrongOwner(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	reg := registry.Global()
	reg.Register(&registry.TaiNode{
		TaiID: "tai-other",
		Mode:  "direct",
		Auth:  registry.AuthInfo{ClientID: "different-client"},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/tai-nodes/heartbeat",
		jsonBody(heartbeatRequest{TaiID: "tai-other"}))
	c.Request.Header.Set("Authorization", "Bearer test-token")
	c.Request.Header.Set("Content-Type", "application/json")

	HandleHeartbeat(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleHeartbeat_NotFound(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/tai-nodes/heartbeat",
		jsonBody(heartbeatRequest{TaiID: "ghost"}))
	c.Request.Header.Set("Authorization", "Bearer test-token")
	c.Request.Header.Set("Content-Type", "application/json")

	HandleHeartbeat(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleUnregister_Success(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	reg := registry.Global()
	reg.Register(&registry.TaiNode{
		TaiID: "tai-abc123",
		Mode:  "direct",
		Auth:  registry.AuthInfo{ClientID: "tai-abc123"},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/tai-nodes/register/tai-abc123", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	c.Params = gin.Params{{Key: "tai_id", Value: "tai-abc123"}}

	HandleUnregister(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	if _, ok := reg.Get("tai-abc123"); ok {
		t.Error("node should be removed after unregister")
	}
}

func TestHandleUnregister_WrongOwner(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	reg := registry.Global()
	reg.Register(&registry.TaiNode{
		TaiID: "tai-other",
		Mode:  "direct",
		Auth:  registry.AuthInfo{ClientID: "different-client"},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/tai-nodes/register/tai-other", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	c.Params = gin.Params{{Key: "tai_id", Value: "tai-other"}}

	HandleUnregister(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleUnregister_NotFound(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/tai-nodes/register/ghost", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	c.Params = gin.Params{{Key: "tai_id", Value: "ghost"}}

	HandleUnregister(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
