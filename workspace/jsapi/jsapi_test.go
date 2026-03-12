package jsapi_test

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	v8runtime "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/volume"
	"github.com/yaoapp/yao/test"
	_ "github.com/yaoapp/yao/workspace/jsapi"
)

type testMode struct {
	Name string
	Addr string // "local" or gRPC address
}

func testModes() []testMode {
	modes := []testMode{{Name: "local", Addr: "local"}}
	if addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR"); addr != "" {
		grpc := strings.TrimPrefix(addr, "tai://")
		modes = append(modes, testMode{Name: "remote", Addr: grpc})
	}
	return modes
}

func setupForMode(t *testing.T, m testMode) {
	t.Helper()
	test.Prepare(t, config.Conf)
	registry.Init(nil)

	if m.Addr == "local" {
		dataDir := t.TempDir()
		vol := volume.NewLocal(dataDir)
		res, err := tai.DialLocal("", dataDir, vol)
		if err != nil {
			t.Fatalf("DialLocal: %v", err)
		}
		reg := registry.Global()
		reg.Register(&registry.TaiNode{TaiID: "local", Mode: "local"})
		reg.SetResources("local", res)
		t.Cleanup(func() { res.Close() })
	} else {
		host, grpcPort := parseHostPort(m.Addr)
		ports := tai.Ports{GRPC: grpcPort}
		res, err := tai.DialRemote(host, ports)
		if err != nil {
			t.Fatalf("DialRemote(%s): %v", m.Addr, err)
		}
		taiID := taiIDFromAddr(m.Addr)
		reg := registry.Global()
		reg.Register(&registry.TaiNode{TaiID: taiID, Mode: "direct"})
		reg.SetResources(taiID, res)
		t.Cleanup(func() { res.Close() })
	}
}

func taiIDFromAddr(addr string) string {
	addr = strings.TrimPrefix(addr, "tai://")
	parts := strings.SplitN(addr, ":", 2)
	return parts[0]
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

func setupGlobal(t *testing.T) {
	t.Helper()
	setupForMode(t, testMode{Name: "local", Addr: "local"})
}

func runJS(t *testing.T, source string) interface{} {
	t.Helper()
	opts := v8runtime.CallOptions{
		Sid:     "test",
		Timeout: 30 * time.Second,
	}
	res, err := v8runtime.Call(opts, source)
	if err != nil {
		t.Fatalf("JS error: %v", err)
	}
	return res
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

func TestWSCreateAndDelete(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSCreateAndDelete() {
				var ws = workspace.Create({ name: "test-proj", owner: "u1", node: "local" });
				var id = ws.id;
				workspace.Delete(id);
				return id;
			}`)
			if res == nil || res == "" {
				t.Error("expected workspace ID")
			}
		})
	}
}

func TestWSGet(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSGet() {
				var ws = workspace.Create({ name: "get-test", owner: "u1", node: "local" });
				var got = workspace.Get(ws.id);
				var result = got ? got.id : "null";
				workspace.Delete(ws.id);
				return result;
			}`)
			if res == "null" {
				t.Error("Get returned null")
			}
		})
	}
}

func TestWSGetNotFound(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSGetNotFound() {
				var got = workspace.Get("ws-nonexistent");
				return got === null ? "null" : "found";
			}`)
			if res != "null" {
				t.Errorf("expected null, got %v", res)
			}
		})
	}
}

func TestWSList(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSList() {
				var ws1 = workspace.Create({ name: "list-a", owner: "u1", node: "local" });
				var ws2 = workspace.Create({ name: "list-b", owner: "u1", node: "local" });
				var list = workspace.List({ owner: "u1" });
				var count = list.length;
				workspace.Delete(ws1.id);
				workspace.Delete(ws2.id);
				return count;
			}`)
			if toInt(res) < 2 {
				t.Errorf("expected >= 2, got %v", res)
			}
		})
	}
}

func TestWSReadWriteFile(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSReadWriteFile() {
				var ws = workspace.Create({ name: "rw-test", owner: "u1", node: "local" });
				ws.WriteFile("hello.txt", "Hello, World!");
				var content = ws.ReadFile("hello.txt");
				workspace.Delete(ws.id);
				return content;
			}`)
			if res != "Hello, World!" {
				t.Errorf("content = %v", res)
			}
		})
	}
}

func TestWSReadDir(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSReadDir() {
				var ws = workspace.Create({ name: "readdir", owner: "u1", node: "local" });
				ws.WriteFile("a.txt", "aaa");
				ws.MkdirAll("sub");
				ws.WriteFile("sub/b.txt", "bbb");
				var entries = ws.ReadDir(".");
				workspace.Delete(ws.id);
				return entries.length;
			}`)
			if toInt(res) < 2 {
				t.Errorf("expected >= 2 entries, got %v", res)
			}
		})
	}
}

func TestWSReadDirRecursive(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSReadDirRecursive() {
				var ws = workspace.Create({ name: "readdir-r", owner: "u1", node: "local" });
				ws.WriteFile("a.txt", "aaa");
				ws.MkdirAll("sub/deep");
				ws.WriteFile("sub/b.txt", "bbb");
				ws.WriteFile("sub/deep/c.txt", "ccc");
				var entries = ws.ReadDir(".", true);
				workspace.Delete(ws.id);
				return entries.length;
			}`)
			if toInt(res) < 4 {
				t.Errorf("expected >= 4 recursive, got %v", res)
			}
		})
	}
}

func TestWSStat(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSStat() {
				var ws = workspace.Create({ name: "stat-test", owner: "u1", node: "local" });
				ws.WriteFile("file.txt", "12345");
				var info = ws.Stat("file.txt");
				workspace.Delete(ws.id);
				return info.size;
			}`)
			if toInt(res) != 5 {
				t.Errorf("expected size 5, got %v", res)
			}
		})
	}
}

func TestWSExistsIsDirIsFile(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSExistsIsDirIsFile() {
				var ws = workspace.Create({ name: "checks", owner: "u1", node: "local" });
				ws.WriteFile("f.txt", "data");
				ws.MkdirAll("d");
				var r = [
					ws.Exists("f.txt"),
					ws.Exists("nope"),
					ws.IsFile("f.txt"),
					ws.IsFile("d"),
					ws.IsDir("d"),
					ws.IsDir("f.txt")
				];
				workspace.Delete(ws.id);
				return JSON.stringify(r);
			}`)
			if res != "[true,false,true,false,true,false]" {
				t.Errorf("checks = %v", res)
			}
		})
	}
}

func TestWSRemoveAndRename(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSRemoveAndRename() {
				var ws = workspace.Create({ name: "ops", owner: "u1", node: "local" });
				ws.WriteFile("del.txt", "x");
				ws.Remove("del.txt");
				var a = ws.Exists("del.txt");

				ws.MkdirAll("rmdir/sub");
				ws.WriteFile("rmdir/sub/f.txt", "x");
				ws.RemoveAll("rmdir");
				var b = ws.Exists("rmdir");

				ws.WriteFile("old.txt", "x");
				ws.Rename("old.txt", "new.txt");
				var c = ws.Exists("old.txt");
				var d = ws.Exists("new.txt");

				workspace.Delete(ws.id);
				return JSON.stringify([a, b, c, d]);
			}`)
			if res != "[false,false,false,true]" {
				t.Errorf("ops = %v", res)
			}
		})
	}
}

func TestWSBase64(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSBase64() {
				var ws = workspace.Create({ name: "b64", owner: "u1", node: "local" });
				ws.WriteFile("src.txt", "base64 test");
				var b64 = ws.ReadFileBase64("src.txt");
				ws.WriteFileBase64("dst.txt", b64);
				var content = ws.ReadFile("dst.txt");
				workspace.Delete(ws.id);
				return content;
			}`)
			if res != "base64 test" {
				t.Errorf("base64 roundtrip = %v", res)
			}
		})
	}
}

func TestWSCopyInternal(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSCopyInternal() {
				var ws = workspace.Create({ name: "copy", owner: "u1", node: "local" });
				ws.WriteFile("src.txt", "copy me");
				ws.Copy("src.txt", "dst.txt");
				var content = ws.ReadFile("dst.txt");
				workspace.Delete(ws.id);
				return content;
			}`)
			if res != "copy me" {
				t.Errorf("copy = %v", res)
			}
		})
	}
}

func TestWSCopyLocalToLocal(t *testing.T) {
	setupGlobal(t)

	srcDir := t.TempDir()
	dstDir := t.TempDir()
	os.WriteFile(srcDir+"/test.txt", []byte("local-to-local"), 0o644)

	srcRel := srcDir[len(os.TempDir()):]
	dstRel := dstDir[len(os.TempDir()):]

	runJS(t, `function TestWSCopyLocalToLocal() {
		var ws = workspace.Create({ name: "l2l", owner: "u1", node: "local" });
		ws.Copy("tmp://`+srcRel+`", "tmp://`+dstRel+`");
		workspace.Delete(ws.id);
		return "ok";
	}`)

	data, err := os.ReadFile(dstDir + "/test.txt")
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "local-to-local" {
		t.Errorf("content = %s", data)
	}
}

func TestWSZipUnzip(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSZipUnzip() {
				var ws = workspace.Create({ name: "zip", owner: "u1", node: "local" });
				ws.MkdirAll("src");
				ws.WriteFile("src/a.txt", "zip content");
				ws.WriteFile("src/b.txt", "more");
				var zr = ws.Zip("src", "out.zip");
				var ur = ws.Unzip("out.zip", "extracted");
				var content = ws.ReadFile("extracted/a.txt");
				workspace.Delete(ws.id);
				return content;
			}`)
			if res != "zip content" {
				t.Errorf("zip/unzip = %v", res)
			}
		})
	}
}

func TestWSGzipGunzip(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSGzipGunzip() {
				var ws = workspace.Create({ name: "gzip", owner: "u1", node: "local" });
				ws.WriteFile("data.txt", "gzip test");
				ws.Gzip("data.txt", "data.txt.gz");
				ws.Gunzip("data.txt.gz", "restored.txt");
				var content = ws.ReadFile("restored.txt");
				workspace.Delete(ws.id);
				return content;
			}`)
			if res != "gzip test" {
				t.Errorf("gzip/gunzip = %v", res)
			}
		})
	}
}

func TestWSTarUntar(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSTarUntar() {
				var ws = workspace.Create({ name: "tar", owner: "u1", node: "local" });
				ws.MkdirAll("src");
				ws.WriteFile("src/a.txt", "tar a");
				ws.Tar("src", "out.tar");
				ws.Untar("out.tar", "extracted");
				var content = ws.ReadFile("extracted/a.txt");
				workspace.Delete(ws.id);
				return content;
			}`)
			if res != "tar a" {
				t.Errorf("tar/untar = %v", res)
			}
		})
	}
}

func TestWSTgzUntgz(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSTgzUntgz() {
				var ws = workspace.Create({ name: "tgz", owner: "u1", node: "local" });
				ws.MkdirAll("src");
				ws.WriteFile("src/x.txt", "tgz x");
				ws.Tgz("src", "out.tgz");
				ws.Untgz("out.tgz", "extracted");
				var content = ws.ReadFile("extracted/x.txt");
				workspace.Delete(ws.id);
				return content;
			}`)
			if res != "tgz x" {
				t.Errorf("tgz/untgz = %v", res)
			}
		})
	}
}

func TestWSZipExcludes(t *testing.T) {
	for _, m := range testModes() {
		t.Run(m.Name, func(t *testing.T) {
			setupForMode(t, m)
			res := runJS(t, `function TestWSZipExcludes() {
				var ws = workspace.Create({ name: "zip-exc", owner: "u1", node: "local" });
				ws.MkdirAll("src");
				ws.WriteFile("src/keep.txt", "keep");
				ws.WriteFile("src/skip.log", "skip");
				ws.Zip("src", "filtered.zip", { excludes: ["*.log"] });
				ws.Unzip("filtered.zip", "out");
				var hasKeep = ws.Exists("out/keep.txt");
				var hasSkip = ws.Exists("out/skip.log");
				workspace.Delete(ws.id);
				return JSON.stringify([hasKeep, hasSkip]);
			}`)
			if res != "[true,false]" {
				t.Errorf("zip excludes = %v", res)
			}
		})
	}
}
