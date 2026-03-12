package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/types"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTest() func() {
	r := registry.NewForTest()
	registry.SetGlobalForTest(r)

	origAuth := authenticateBearer
	authenticateBearer = func(token string) (types.AuthInfo, error) {
		return types.AuthInfo{
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
		NodeID:       "9100",
		MachineID:    "m-001",
		Version:      "0.2.0",
		DisplayName:  "My Dev Machine",
		Addr:         "192.168.1.100",
		Ports:        map[string]int{"grpc": 19100, "http": 8099},
		Capabilities: map[string]bool{"docker": true, "host_exec": false},
		System: types.SystemInfo{
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
	taiID, _ := resp["tai_id"].(string)
	if taiID == "" || len(taiID) < 5 || taiID[:4] != "tai-" {
		t.Errorf("tai_id = %v, want server-generated tai-xxx", resp["tai_id"])
	}
	if _, ok := resp["remote_ip"]; !ok {
		t.Error("response missing remote_ip")
	}

	snap, ok := registry.Global().Get(taiID)
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
	if snap.DisplayName != "My Dev Machine" {
		t.Errorf("DisplayName = %q, want %q", snap.DisplayName, "My Dev Machine")
	}
}

func TestHandleRegister_ServerGeneratedTaiID(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	body := registerRequest{
		NodeID:       "19100",
		ClientID:     "local-uuid-001",
		MachineID:    "m-001",
		Version:      "0.2.0",
		DisplayName:  "Generated ID Node",
		Addr:         "192.168.1.200",
		Ports:        map[string]int{"grpc": 19100},
		Capabilities: map[string]bool{"docker": true},
		System:       types.SystemInfo{OS: "darwin", Arch: "arm64", Hostname: "mac-01", NumCPU: 12},
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

	generatedID, ok := resp["tai_id"].(string)
	if !ok || generatedID == "" {
		t.Fatal("response missing tai_id")
	}
	if generatedID == "19100" {
		t.Error("tai_id should be server-generated, not the raw node_id")
	}
	if len(generatedID) != 26 {
		t.Errorf("tai_id length = %d, want 26 (tai- + 22 base62); got %q", len(generatedID), generatedID)
	}

	snap, ok2 := registry.Global().Get(generatedID)
	if !ok2 {
		t.Fatalf("node %q not found in registry", generatedID)
	}
	if snap.DisplayName != "Generated ID Node" {
		t.Errorf("DisplayName = %q, want %q", snap.DisplayName, "Generated ID Node")
	}

	// Deterministic: same inputs produce same ID
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("POST", "/tai-nodes/register", jsonBody(body))
	c2.Request.Header.Set("Authorization", "Bearer test-token")
	c2.Request.Header.Set("Content-Type", "application/json")
	HandleRegister(c2)

	var resp2 map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2["tai_id"] != generatedID {
		t.Errorf("not deterministic: %v != %v", resp2["tai_id"], generatedID)
	}
}

func TestHandleRegister_MissingAuth(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/tai-nodes/register", jsonBody(registerRequest{NodeID: "x", MachineID: "m1"}))
	c.Request.Header.Set("Content-Type", "application/json")

	HandleRegister(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleRegister_MissingTaiIDAndClientID(t *testing.T) {
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
		Auth:  types.AuthInfo{ClientID: "tai-abc123"},
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
		Auth:  types.AuthInfo{ClientID: "different-client"},
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
		Auth:  types.AuthInfo{ClientID: "tai-abc123"},
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
		Auth:  types.AuthInfo{ClientID: "different-client"},
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
