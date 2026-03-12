package sandbox_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	tairuntime "github.com/yaoapp/yao/tai/runtime"
	"github.com/yaoapp/yao/workspace"
)

// k8sSem limits concurrent K8s pod creation to avoid overwhelming the cluster.
var k8sSem = make(chan struct{}, 2)

// k8sCleanupMu serialises K8s pod cleanup to prevent overlapping API calls
// when many tests finish at once.
var k8sCleanupMu sync.Mutex

// ---------------------------------------------------------------------------
// Build-tag extension points.
// Each tag file (testutils_remote_test.go, testutils_k8s_test.go, …) appends
// provider functions in its init(). This lets tags compose freely:
//
//	go test ./sandbox/v2/...                           → local only
//	go test -tags remote ./sandbox/v2/...              → local + remote
//	go test -tags "remote,k8s" ./sandbox/v2/...        → local + remote + k8s
//	go test -tags "remote,containerized,k8s,wintest"   → all
//
// ---------------------------------------------------------------------------
var (
	extraNodeProviders     []func() []nodeConfig
	extraHostExecProviders []func() []hostExecTarget
	extraPurgeProviders    []func() []purgeTarget
)

func TestMain(m *testing.M) {
	purgeStaleContainers()
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Purge stale containers from previous runs
// ---------------------------------------------------------------------------

type purgeTarget struct {
	name    string
	addr    string
	dialOps []tai.DialOption
}

func purgeStaleContainers() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type target struct {
		name    string
		addr    string
		dialOps []tai.DialOption
	}

	var targets []target
	targets = append(targets, target{name: "local", addr: testLocalAddr()})

	for _, fn := range extraPurgeProviders {
		for _, extra := range fn() {
			targets = append(targets, target{name: extra.name, addr: extra.addr, dialOps: extra.dialOps})
		}
	}

	for _, tgt := range targets {
		res, err := dialForTest(tgt.addr, tgt.dialOps...)
		if err != nil {
			continue
		}
		sb := res.Runtime
		if sb == nil {
			res.Close()
			continue
		}
		containers, err := sb.List(ctx, tairuntime.ListOptions{All: true})
		if err != nil {
			res.Close()
			continue
		}
		for _, c := range containers {
			id := c.Name
			if id == "" {
				id = c.ID
			}
			if !strings.HasPrefix(id, "sb-") && !strings.HasPrefix(c.Labels["sandbox-id"], "sb-") {
				continue
			}
			sb.Remove(ctx, id, true)
			log.Printf("[purge] %s: removed stale container %s", tgt.name, id)
		}
		res.Close()
	}
}

// ---------------------------------------------------------------------------
// Node / HostExec configuration
// ---------------------------------------------------------------------------

type nodeConfig struct {
	Name    string
	Addr    string
	TaiID   string
	DialOps []tai.DialOption
}

type hostExecTarget struct {
	Name        string
	Addr        string
	TaiID       string
	IsWinNative bool
}

// testNodes returns node configs. "local" is always present; other
// environments are injected by build-tag files via extraNodeProviders.
func testNodes() []nodeConfig {
	nodes := []nodeConfig{
		{Name: "local", Addr: testLocalAddr()},
	}
	for _, fn := range extraNodeProviders {
		nodes = append(nodes, fn()...)
	}
	return nodes
}

// hostExecTargets returns HostExec targets. Populated entirely by
// build-tag files via extraHostExecProviders.
func hostExecTargets() []hostExecTarget {
	var targets []hostExecTarget
	for _, fn := range extraHostExecProviders {
		targets = append(targets, fn()...)
	}
	return targets
}

// ---------------------------------------------------------------------------
// Skip helpers
// ---------------------------------------------------------------------------

func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if testLocalAddr() == "" {
		t.Skip("SANDBOX_TEST_LOCAL_ADDR not set, skipping Docker tests")
	}
}

func skipIfNoTai(t *testing.T) {
	t.Helper()
	if os.Getenv("SANDBOX_TEST_REMOTE_ADDR") == "" {
		t.Skip("SANDBOX_TEST_REMOTE_ADDR not set, skipping Tai proxy tests")
	}
}

func skipIfNoHostExec(t *testing.T) {
	t.Helper()
	if len(hostExecTargets()) == 0 {
		t.Skip("no HostExec targets configured")
	}
}

// ---------------------------------------------------------------------------
// Command helpers (Windows HostExec command translation)
// ---------------------------------------------------------------------------

func linuxCmd(tgt hostExecTarget, cmd string, args ...string) (string, []string) {
	if tgt.IsWinNative {
		switch cmd {
		case "echo":
			return "cmd.exe", append([]string{"/c", "echo"}, args...)
		case "pwd":
			return "cmd.exe", []string{"/c", "cd"}
		case "env":
			return "cmd.exe", []string{"/c", "set"}
		case "sleep":
			return "cmd.exe", []string{"/c", "ping", "-n", "10", "127.0.0.1"}
		case "cat":
			return "cmd.exe", []string{"/c", "more"}
		case "sh":
			if len(args) >= 2 && args[0] == "-c" {
				return "cmd.exe", []string{"/c", args[1]}
			}
			return "cmd.exe", append([]string{"/c"}, args...)
		default:
			return cmd, args
		}
	}
	return cmd, args
}

// ---------------------------------------------------------------------------
// Environment helpers
// ---------------------------------------------------------------------------

func testLocalAddr() string {
	if addr := os.Getenv("SANDBOX_TEST_LOCAL_ADDR"); addr != "" {
		return addr
	}
	return "local"
}

func testImage() string {
	if img := os.Getenv("SANDBOX_TEST_IMAGE"); img != "" {
		return img
	}
	return "alpine:latest"
}

func envPort(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			return p
		}
	}
	return fallback
}

// ---------------------------------------------------------------------------
// Dial + Register helper (replaces old tai.New)
// ---------------------------------------------------------------------------

// dialForTest calls DialLocal or DialRemote based on the address.
func dialForTest(addr string, dialOps ...tai.DialOption) (*tai.ConnResources, error) {
	if addr == "local" || addr == "" {
		return tai.DialLocal("", "", nil)
	}
	host, grpcPort := parseHostPort(addr)
	ports := tai.Ports{GRPC: grpcPort}
	return tai.DialRemote(host, ports, dialOps...)
}

// registerForTest dials and registers a node in the registry. Returns the
// taiID. On failure it calls t.Fatalf.
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
	return taiID, res
}

func taiIDFromAddr(addr string) string {
	if addr == "local" || addr == "" {
		return "local"
	}
	addr = strings.TrimPrefix(addr, "tai://")
	host, _ := parseHostPort(addr)
	return host
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

// ---------------------------------------------------------------------------
// Manager / Box setup helpers
// ---------------------------------------------------------------------------

func registerNode(t *testing.T, pc *nodeConfig) {
	t.Helper()
	taiID, res := registerForTest(t, pc.Addr, pc.DialOps...)
	pc.TaiID = taiID
	t.Cleanup(func() { res.Close() })
}

func setupManager(t *testing.T, nodes ...nodeConfig) (*sandbox.Manager, []nodeConfig) {
	t.Helper()
	if registry.Global() == nil {
		registry.Init(nil)
	}

	out := make([]nodeConfig, len(nodes))
	copy(out, nodes)
	for i := range out {
		taiID, _ := registerForTest(t, out[i].Addr, out[i].DialOps...)
		out[i].TaiID = taiID
	}

	sandbox.Init()
	m := sandbox.M()
	t.Cleanup(func() { m.Close() })
	return m, out
}

func setupManagerForNode(t *testing.T, pc *nodeConfig) *sandbox.Manager {
	t.Helper()
	m, registered := setupManager(t, *pc)
	*pc = registered[0]
	return m
}

func setupManagerWithWorkspace(t *testing.T, pc *nodeConfig) (*sandbox.Manager, *workspace.Manager) {
	t.Helper()
	sbm := setupManagerForNode(t, pc)
	return sbm, workspace.M()
}

func ensureTestImage(t *testing.T, m *sandbox.Manager, nodeID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := m.EnsureImage(ctx, nodeID, testImage(), sandbox.ImagePullOptions{}); err != nil {
		t.Fatalf("EnsureImage(%s, %s): %v", nodeID, testImage(), err)
	}
}

func createTestBox(t *testing.T, m *sandbox.Manager, pc nodeConfig, opts ...func(*sandbox.CreateOptions)) *sandbox.Box {
	t.Helper()
	co := sandbox.CreateOptions{
		Image:  testImage(),
		Owner:  "test-user",
		NodeID: pc.TaiID,
	}
	for _, fn := range opts {
		fn(&co)
	}

	nodeID := co.NodeID
	if nodeID == "" {
		nodes := m.Nodes()
		if len(nodes) > 0 {
			nodeID = nodes[0].TaiID
			co.NodeID = nodeID
		}
	}

	isK8s := pc.Name == "k8s"
	if isK8s {
		k8sSem <- struct{}{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if nodeID != "" {
		if err := m.EnsureImage(ctx, nodeID, co.Image, sandbox.ImagePullOptions{}); err != nil {
			if isK8s {
				<-k8sSem
			}
			t.Fatalf("EnsureImage(%s, %s): %v", nodeID, co.Image, err)
		}
	}

	box, err := m.Create(ctx, co)
	if err != nil {
		if isK8s {
			<-k8sSem
		}
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanCancel()
		if isK8s {
			k8sCleanupMu.Lock()
			defer k8sCleanupMu.Unlock()
		}
		if err := m.Remove(cleanCtx, box.ID()); err != nil {
			t.Logf("cleanup Remove(%s): %v", box.ID(), err)
		}
		if isK8s {
			<-k8sSem
		}
	})
	return box
}

// Ensure imports are used.
var _ = fmt.Sprintf
