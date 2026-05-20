package jsapi_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	v8runtime "github.com/yaoapp/gou/runtime/v8"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"

	_ "github.com/yaoapp/yao/sandbox/v2/jsapi"
)

func TestMain(m *testing.M) {
	testprepare.MustLoadEnv()
	sandboxtest.PurgeStaleContainers("sb-")
	code := m.Run()
	testprepare.Cleanup()
	os.Exit(code)
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

func nodeID() string {
	return sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
}

func TestJSAPI_Create(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	img := sandboxtest.TestImage()
	nid := nodeID()
	sandboxtest.EnsureImage(t, sandbox.M(), nid)

	res := runJS(t, fmt.Sprintf(`function TestCreate() {
		var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
		if (pc.kind !== "box") throw new Error("kind=" + pc.kind);
		if (!pc.id) throw new Error("no id");
		var id = pc.id;
		sandbox.Delete(id);
		return id;
	}`, img, nid))
	if res == nil || res == "" {
		t.Error("expected box id")
	}
}

func TestJSAPI_Get(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	img := sandboxtest.TestImage()
	nid := nodeID()
	sandboxtest.EnsureImage(t, sandbox.M(), nid)

	res := runJS(t, fmt.Sprintf(`function TestGet() {
		var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
		var id = pc.id;
		var got = sandbox.Get(id);
		if (!got) throw new Error("Get returned null");
		if (got.kind !== "box") throw new Error("kind=" + got.kind);
		sandbox.Delete(id);
		return id;
	}`, img, nid))
	if res == nil || res == "" {
		t.Error("expected box id")
	}
}

func TestJSAPI_GetNotFound(t *testing.T) {
	testprepare.PrepareSandbox(t)

	res := runJS(t, `function TestGetNotFound() {
		var got = sandbox.Get("sb-nonexistent-id");
		return got === null ? "null" : "found";
	}`)
	if res != "null" {
		t.Errorf("expected null, got %v", res)
	}
}

func TestJSAPI_Delete(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	img := sandboxtest.TestImage()
	nid := nodeID()
	sandboxtest.EnsureImage(t, sandbox.M(), nid)

	res := runJS(t, fmt.Sprintf(`function TestDelete() {
		var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
		var id = pc.id;
		sandbox.Delete(id);
		var got = sandbox.Get(id);
		return got === null ? "deleted" : "still exists";
	}`, img, nid))
	if res != "deleted" {
		t.Errorf("expected deleted, got %v", res)
	}
}

func TestJSAPI_List(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	img := sandboxtest.TestImage()
	nid := nodeID()
	sandboxtest.EnsureImage(t, sandbox.M(), nid)

	res := runJS(t, fmt.Sprintf(`function TestList() {
		var a = sandbox.Create({ image: "%s", owner: "list-user", node_id: "%s" });
		var b = sandbox.Create({ image: "%s", owner: "list-user", node_id: "%s" });
		var list = sandbox.List({ owner: "list-user" });
		var count = list.length;
		sandbox.Delete(a.id);
		sandbox.Delete(b.id);
		return count;
	}`, img, nid, img, nid))
	n := toInt(res)
	if n < 2 {
		t.Errorf("expected >= 2, got %d", n)
	}
}

func TestJSAPI_Exec(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	img := sandboxtest.TestImage()
	nid := nodeID()
	sandboxtest.EnsureImage(t, sandbox.M(), nid)

	res := runJS(t, fmt.Sprintf(`function TestExec() {
		var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
		var r = pc.Exec(["echo", "hello-jsapi"]);
		sandbox.Delete(pc.id);
		return r.stdout;
	}`, img, nid))
	s := fmt.Sprintf("%v", res)
	if !strings.Contains(s, "hello-jsapi") {
		t.Errorf("stdout = %q, want contain 'hello-jsapi'", s)
	}
}

func TestJSAPI_Stream(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	img := sandboxtest.TestImage()
	nid := nodeID()
	sandboxtest.EnsureImage(t, sandbox.M(), nid)

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
	}`, img, nid))
	s := fmt.Sprintf("%v", res)
	if !strings.Contains(s, "streaming|0") {
		t.Errorf("result = %q, want contain 'streaming|0'", s)
	}
}

func TestJSAPI_ComputerInfo(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	img := sandboxtest.TestImage()
	nid := nodeID()
	sandboxtest.EnsureImage(t, sandbox.M(), nid)

	res := runJS(t, fmt.Sprintf(`function TestComputerInfo() {
		var pc = sandbox.Create({ image: "%s", owner: "test-user", node_id: "%s" });
		var info = pc.ComputerInfo();
		sandbox.Delete(pc.id);
		return info.kind;
	}`, img, nid))
	if res != "box" {
		t.Errorf("kind = %q, want 'box'", res)
	}
}

func TestJSAPI_Nodes(t *testing.T) {
	testprepare.PrepareSandbox(t)

	res := runJS(t, `function TestNodes() {
		var nodes = sandbox.Nodes();
		return Array.isArray(nodes) ? "array" : typeof nodes;
	}`)
	if res != "array" {
		t.Errorf("expected array, got %v", res)
	}
}

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
