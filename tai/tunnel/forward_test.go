package tunnel

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/types"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestResolveRoute_Proxy(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantType      string
		wantContainer string
		wantPort      int
		wantSubpath   string
	}{
		{
			"basic_proxy",
			"/tai/abc/proxy/cid123:8080/foo/bar",
			"proxy", "cid123", 8080, "/foo/bar",
		},
		{
			"proxy_root",
			"/tai/abc/proxy/cid:3000",
			"proxy", "cid", 3000, "/",
		},
		{
			"proxy_host",
			"/v1/tai/abc/proxy/__host__:9090/api",
			"proxy", "__host__", 9090, "/api",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = &http.Request{URL: &url.URL{Path: tt.path}}
			c.Params = gin.Params{{Key: "taiID", Value: "abc"}}
			node := &types.NodeMeta{}

			r, err := resolveRoute(c, node)
			if err != nil {
				t.Fatalf("resolveRoute error: %v", err)
			}
			if r.channelType != tt.wantType {
				t.Errorf("channelType = %q, want %q", r.channelType, tt.wantType)
			}
			if r.containerID != tt.wantContainer {
				t.Errorf("containerID = %q, want %q", r.containerID, tt.wantContainer)
			}
			if r.containerPort != tt.wantPort {
				t.Errorf("containerPort = %d, want %d", r.containerPort, tt.wantPort)
			}
			if r.subpath != tt.wantSubpath {
				t.Errorf("subpath = %q, want %q", r.subpath, tt.wantSubpath)
			}
		})
	}
}

func TestResolveRoute_VNC(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantContainer string
		wantPort      int
	}{
		{"vnc_basic", "/tai/abc/vnc/container1/ws", "container1", defaultVNCPort},
		{"vnc_host", "/v1/tai/abc/vnc/__host__/ws", "__host__", defaultVNCPort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = &http.Request{URL: &url.URL{Path: tt.path}}
			c.Params = gin.Params{{Key: "taiID", Value: "abc"}}
			node := &types.NodeMeta{}

			r, err := resolveRoute(c, node)
			if err != nil {
				t.Fatalf("resolveRoute error: %v", err)
			}
			if r.channelType != "vnc" {
				t.Errorf("channelType = %q, want vnc", r.channelType)
			}
			if r.containerID != tt.wantContainer {
				t.Errorf("containerID = %q, want %q", r.containerID, tt.wantContainer)
			}
			if r.containerPort != tt.wantPort {
				t.Errorf("containerPort = %d, want %d", r.containerPort, tt.wantPort)
			}
		})
	}
}

func TestResolveRoute_Unknown(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = &http.Request{URL: &url.URL{Path: "/tai/abc/unknown/something"}}
	c.Params = gin.Params{{Key: "taiID", Value: "abc"}}
	node := &types.NodeMeta{}

	_, err := resolveRoute(c, node)
	if err == nil {
		t.Error("expected error for unknown route")
	}
}

func TestRewriteRequest_Proxy(t *testing.T) {
	u, _ := url.Parse("http://localhost/v1/tai/abc/proxy/cid:8080/foo")
	orig := &http.Request{
		Method:     "GET",
		URL:        u,
		RequestURI: u.RequestURI(),
		Host:       "localhost",
		Header:     http.Header{},
	}
	route := &forwardRoute{
		channelType:   "proxy",
		containerID:   "cid",
		containerPort: 8080,
		subpath:       "/foo",
	}

	got := rewriteRequest(orig, "abc", route)
	if got.URL.Path != "/foo" {
		t.Errorf("path = %q, want /foo", got.URL.Path)
	}
	if got == orig {
		t.Error("rewriteRequest should return a clone")
	}
}

func TestRewriteRequest_VNC(t *testing.T) {
	u, _ := url.Parse("http://localhost/tai/node-1/vnc/cid/ws")
	orig := &http.Request{
		Method:     "GET",
		URL:        u,
		RequestURI: u.RequestURI(),
		Host:       "localhost",
		Header: http.Header{
			"Connection": {"Upgrade"},
			"Upgrade":    {"websocket"},
		},
	}
	route := &forwardRoute{
		channelType:   "vnc",
		containerID:   "cid",
		containerPort: 5900,
		subpath:       "/vnc/cid/ws",
	}

	got := rewriteRequest(orig, "node-1", route)
	if got.URL.Path != "/vnc/cid/ws" {
		t.Errorf("path = %q, want /vnc/cid/ws", got.URL.Path)
	}
	if got.Header.Get("Connection") != "Upgrade" {
		t.Error("expected Connection header preserved")
	}
}

func TestHandleForwardLazy_NilHandler(t *testing.T) {
	old := globalHandler
	globalHandler = nil
	defer func() { globalHandler = old }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/tai/abc/proxy/test", nil)

	HandleForwardLazy(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleForward_NodeNotFound(t *testing.T) {
	reg := registry.NewForTest()
	h := NewTunnelHandler(reg)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/tai/nonexistent/proxy/api", nil)
	c.Params = gin.Params{{Key: "taiID", Value: "nonexistent"}}

	h.HandleForward(c)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestHandleForward_UnknownRoute(t *testing.T) {
	reg := registry.NewForTest()
	h := NewTunnelHandler(reg)

	reg.Register(&registry.TaiNode{
		TaiID: "online-node",
		Mode:  "tunnel",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/tai/online-node/unknown/foo", nil)
	c.Params = gin.Params{{Key: "taiID", Value: "online-node"}}

	h.HandleForward(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unresolvable route, got %d", w.Code)
	}
}

func TestHandleForwardLazy_WithHandler(t *testing.T) {
	reg := registry.NewForTest()
	old := globalHandler
	globalHandler = NewTunnelHandler(reg)
	defer func() { globalHandler = old }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/tai/missing/proxy/api", nil)
	c.Params = gin.Params{{Key: "taiID", Value: "missing"}}

	HandleForwardLazy(c)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestHandleForward_ViaRealHTTP(t *testing.T) {
	reg := registry.NewForTest()
	h := NewTunnelHandler(reg)

	reg.Register(&registry.TaiNode{
		TaiID: "http-node",
		Mode:  "tunnel",
	})

	router := gin.New()
	router.Any("/tai/:taiID/proxy/*path", func(c *gin.Context) { h.HandleForward(c) })

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/tai/http-node/proxy/cid:8080/api")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		t.Error("expected non-200 response for failed forward")
	}
}
