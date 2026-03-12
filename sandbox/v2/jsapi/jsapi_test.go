package jsapi_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	v8runtime "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/test"

	_ "github.com/yaoapp/yao/sandbox/v2/jsapi"
)

type testMode struct {
	Name  string
	Addr  string
	TaiID string
}

func testModes() []testMode {
	modes := []testMode{{Name: "local", Addr: "local"}}
	if addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR"); addr != "" {
		modes = append(modes, testMode{Name: "remote", Addr: addr})
	}
	return modes
}

func testImage() string {
	if img := os.Getenv("SANDBOX_TEST_IMAGE"); img != "" {
		return img
	}
	return "alpine:latest"
}

func setupSandbox(t *testing.T, m *testMode) {
	t.Helper()
	test.Prepare(t, config.Conf)

	reg := registry.Global()
	if reg == nil {
		registry.Init(nil)
		reg = registry.Global()
	}

	taiID, _ := registerForTest(t, m.Addr)
	m.TaiID = taiID

	sandbox.Init()
	mgr := sandbox.M()
	t.Cleanup(func() { mgr.Close() })
}

func registerForTest(t testing.TB, addr string, dialOps ...tai.DialOption) (string, *tai.ConnResources) {
	t.Helper()
	if registry.Global() == nil {
		registry.Init(nil)
	}
	res, err := dialForTest(addr, dialOps...)
	if err != nil {
		t.Fatalf("dialForTest(%s): %v", addr, err)
	}
	taiID := taiIDFromAddr(addr)
	reg := registry.Global()
	reg.Register(&registry.TaiNode{TaiID: taiID, Mode: modeForAddr(addr)})
	reg.SetResources(taiID, res)
	t.Cleanup(func() { res.Close() })
	return taiID, res
}

func dialForTest(addr string, dialOps ...tai.DialOption) (*tai.ConnResources, error) {
	if addr == "local" || addr == "" {
		return tai.DialLocal("", "", nil)
	}
	host, grpcPort := parseHostPort(addr)
	ports := tai.Ports{GRPC: grpcPort}
	return tai.DialRemote(host, ports, dialOps...)
}

func taiIDFromAddr(addr string) string {
	if addr == "local" || addr == "" {
		return "local"
	}
	addr = strings.TrimPrefix(addr, "tai://")
	parts := strings.SplitN(addr, ":", 2)
	return parts[0]
}

func modeForAddr(addr string) string {
	if addr == "local" || addr == "" {
		return "local"
	}
	return "direct"
}

func parseHostPort(addr string) (string, int) {
	addr = strings.TrimPrefix(addr, "tai://")
	parts := strings.SplitN(addr, ":", 2)
	h := parts[0]
	if len(parts) == 2 {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			return h, p
		}
	}
	return h, 19100
}

func runJS(t *testing.T, source string) interface{} {
	t.Helper()
	res, err := v8runtime.Call(v8runtime.CallOptions{
		Sid:     "test",
		Timeout: 60 * time.Second,
	}, source)
	if err != nil {
		t.Fatalf("JS error: %v", err)
	}
	return res
}

func runJSExpectError(t *testing.T, source string) string {
	t.Helper()
	_, err := v8runtime.Call(v8runtime.CallOptions{
		Sid:     "test",
		Timeout: 30 * time.Second,
	}, source)
	if err == nil {
		t.Fatal("expected JS error, got nil")
	}
	return err.Error()
}

func skipIfNoDocker(t *testing.T) {
	t.Helper()
	addr := os.Getenv("SANDBOX_TEST_LOCAL_ADDR")
	if addr == "" {
		addr = "local"
	}
	_ = addr
}

// ---------------------------------------------------------------------------
// sandbox.Create / sandbox.Get / sandbox.Delete
// ---------------------------------------------------------------------------

func TestCreate(t *testing.T) {
	skipIfNoDocker(t)
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupSandbox(t, &m)
			img := testImage()
			res := runJS(t, fmt.Sprintf(`function TestCreate() {
				var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
				if (pc.kind !== "box") throw new Error("kind=" + pc.kind);
				if (!pc.id) throw new Error("no id");
				var id = pc.id;
				sandbox.Delete(id);
				return id;
			}`, img, m.TaiID))
			if res == nil || res == "" {
				t.Error("expected box id")
			}
		})
	}
}

func TestGet(t *testing.T) {
	skipIfNoDocker(t)
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupSandbox(t, &m)
			img := testImage()
			res := runJS(t, fmt.Sprintf(`function TestGet() {
				var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
				var id = pc.id;
				var got = sandbox.Get(id);
				if (!got) throw new Error("Get returned null");
				if (got.kind !== "box") throw new Error("kind=" + got.kind);
				sandbox.Delete(id);
				return id;
			}`, img, m.TaiID))
			if res == nil || res == "" {
				t.Error("expected box id")
			}
		})
	}
}

func TestGetNotFound(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupSandbox(t, &m)
			res := runJS(t, `function TestGetNotFound() {
				var got = sandbox.Get("sb-nonexistent-id");
				return got === null ? "null" : "found";
			}`)
			if res != "null" {
				t.Errorf("expected null, got %v", res)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	skipIfNoDocker(t)
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupSandbox(t, &m)
			img := testImage()
			res := runJS(t, fmt.Sprintf(`function TestDelete() {
				var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
				var id = pc.id;
				sandbox.Delete(id);
				var got = sandbox.Get(id);
				return got === null ? "deleted" : "still exists";
			}`, img, m.TaiID))
			if res != "deleted" {
				t.Errorf("expected deleted, got %v", res)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// sandbox.List
// ---------------------------------------------------------------------------

func TestList(t *testing.T) {
	skipIfNoDocker(t)
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupSandbox(t, &m)
			img := testImage()
			res := runJS(t, fmt.Sprintf(`function TestList() {
				var a = sandbox.Create({ image: "%s", owner: "list-user", node_id: "%s" });
				var b = sandbox.Create({ image: "%s", owner: "list-user", node_id: "%s" });
				var list = sandbox.List({ owner: "list-user" });
				var count = list.length;
				sandbox.Delete(a.id);
				sandbox.Delete(b.id);
				return count;
			}`, img, m.TaiID, img, m.TaiID))
			n := toInt(res)
			if n < 2 {
				t.Errorf("expected >= 2, got %d", n)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Computer.Exec
// ---------------------------------------------------------------------------

func TestExec(t *testing.T) {
	skipIfNoDocker(t)
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupSandbox(t, &m)
			img := testImage()
			res := runJS(t, fmt.Sprintf(`function TestExec() {
				var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
				var r = pc.Exec(["echo", "hello-jsapi"]);
				sandbox.Delete(pc.id);
				return r.stdout;
			}`, img, m.TaiID))
			s := fmt.Sprintf("%v", res)
			if !strings.Contains(s, "hello-jsapi") {
				t.Errorf("stdout = %q, want contain 'hello-jsapi'", s)
			}
		})
	}
}

func TestExecWithOptions(t *testing.T) {
	skipIfNoDocker(t)
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupSandbox(t, &m)
			img := testImage()
			res := runJS(t, fmt.Sprintf(`function TestExecWithOptions() {
				var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
				var r = pc.Exec(["pwd"], { workdir: "/tmp" });
				sandbox.Delete(pc.id);
				return r.stdout;
			}`, img, m.TaiID))
			s := fmt.Sprintf("%v", res)
			if !strings.Contains(s, "/tmp") {
				t.Errorf("stdout = %q, want contain '/tmp'", s)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Computer.Stream
// ---------------------------------------------------------------------------

func TestStream(t *testing.T) {
	skipIfNoDocker(t)
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupSandbox(t, &m)
			img := testImage()
			res := runJS(t, fmt.Sprintf(`function TestStream() {
				var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
				var chunks = [];
				var exitCode = -1;
				pc.Stream(["echo", "streaming"], function(type, data) {
					if (type === "stdout") chunks.push(data);
					if (type === "exit") exitCode = data;
				});
				sandbox.Delete(pc.id);
				return chunks.join("").trim() + "|" + exitCode;
			}`, img, m.TaiID))
			s := fmt.Sprintf("%v", res)
			if !strings.Contains(s, "streaming|0") {
				t.Errorf("result = %q, want contain 'streaming|0'", s)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Computer.ComputerInfo
// ---------------------------------------------------------------------------

func TestComputerInfo(t *testing.T) {
	skipIfNoDocker(t)
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupSandbox(t, &m)
			img := testImage()
			res := runJS(t, fmt.Sprintf(`function TestComputerInfo() {
				var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
				var info = pc.ComputerInfo();
				sandbox.Delete(pc.id);
				return info.kind;
			}`, img, m.TaiID))
			if res != "box" {
				t.Errorf("kind = %q, want 'box'", res)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Computer.Info (box-only)
// ---------------------------------------------------------------------------

func TestBoxInfo(t *testing.T) {
	skipIfNoDocker(t)
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupSandbox(t, &m)
			img := testImage()
			res := runJS(t, fmt.Sprintf(`function TestBoxInfo() {
				var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
				var info = pc.Info();
				sandbox.Delete(pc.id);
				return info.id ? "ok" : "no-id";
			}`, img, m.TaiID))
			if res != "ok" {
				t.Errorf("expected ok, got %v", res)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Box-only method on host → error
// ---------------------------------------------------------------------------

func TestHostBoxMethodsThrow(t *testing.T) {
	if os.Getenv("SANDBOX_TEST_REMOTE_ADDR") == "" {
		t.Skip("no remote host configured")
	}
	for _, m := range testModes() {
		if m.Name == "local" {
			continue
		}
		t.Run(m.Name, func(t *testing.T) {
			setupSandbox(t, &m)
			errMsg := runJSExpectError(t, fmt.Sprintf(`function TestHostBoxMethodsThrow() {
				var host = sandbox.Host("%s");
				host.Info();
			}`, m.TaiID))
			if !strings.Contains(errMsg, "not supported") {
				t.Errorf("expected 'not supported' error, got: %s", errMsg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Computer.kind property
// ---------------------------------------------------------------------------

func TestComputerKind(t *testing.T) {
	skipIfNoDocker(t)
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupSandbox(t, &m)
			img := testImage()
			res := runJS(t, fmt.Sprintf(`function TestComputerKind() {
				var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
				var k = pc.kind;
				sandbox.Delete(pc.id);
				return k;
			}`, img, m.TaiID))
			if res != "box" {
				t.Errorf("kind = %q, want 'box'", res)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// sandbox.Nodes (requires registry)
// ---------------------------------------------------------------------------

func TestNodes(t *testing.T) {
	test.Prepare(t, config.Conf)
	registry.Init(nil)
	res := runJS(t, `function TestNodes() {
		var nodes = sandbox.Nodes();
		return Array.isArray(nodes) ? "array" : typeof nodes;
	}`)
	if res != "array" {
		t.Errorf("expected array, got %v", res)
	}
}

func TestGetNodeNotFound(t *testing.T) {
	test.Prepare(t, config.Conf)
	registry.Init(nil)
	res := runJS(t, `function TestGetNodeNotFound() {
		var node = sandbox.GetNode("tai-nonexistent");
		return node === null ? "null" : "found";
	}`)
	if res != "null" {
		t.Errorf("expected null, got %v", res)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}
