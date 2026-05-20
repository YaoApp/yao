package sandboxtest

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	v8 "github.com/yaoapp/gou/runtime/v8"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	tairuntime "github.com/yaoapp/yao/tai/runtime"
	"github.com/yaoapp/yao/tai/types"
)

// CleanupRegistrar is set by testprepare to route cleanup functions to the
// global cleanup list (avoiding circular import).
var CleanupRegistrar func(func())

func registerGlobalCleanup(fn func()) {
	if CleanupRegistrar != nil {
		CleanupRegistrar(fn)
	}
}

// ---------------------------------------------------------------------------
// Environment readers
// ---------------------------------------------------------------------------

func TestImage() string {
	if img := os.Getenv("SANDBOX_TEST_IMAGE"); img != "" {
		return img
	}
	return "alpine:latest"
}

func EnvPort(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			return p
		}
	}
	return fallback
}

// ---------------------------------------------------------------------------
// Environment checks (t.Fatal, never t.Skip)
// ---------------------------------------------------------------------------

// RequireDocker verifies that a Docker-capable Tai node is registered in the
// global registry. Must be called after InitStack.
func RequireDocker(t *testing.T) {
	t.Helper()
	reg := registry.Global()
	if reg == nil {
		t.Fatal("sandboxtest.RequireDocker: registry not initialized")
	}
	for _, n := range reg.List() {
		if n.Capabilities.Docker {
			return
		}
	}
	t.Fatal("sandboxtest.RequireDocker: no Docker-capable node registered")
}

// RequireHostExec verifies that a HostExec-capable Tai node is registered.
func RequireHostExec(t *testing.T) {
	t.Helper()
	reg := registry.Global()
	if reg == nil {
		t.Fatal("sandboxtest.RequireHostExec: registry not initialized")
	}
	for _, n := range reg.List() {
		if n.Capabilities.HostExec {
			return
		}
	}
	t.Fatal("sandboxtest.RequireHostExec: no HostExec-capable node registered")
}

// ---------------------------------------------------------------------------
// Node lookup helpers
// ---------------------------------------------------------------------------

// DockerNodeID returns the TaiID of the first Docker-capable node.
func DockerNodeID(t *testing.T) string {
	t.Helper()
	reg := registry.Global()
	if reg == nil {
		t.Fatal("sandboxtest.DockerNodeID: registry not initialized")
	}
	for _, n := range reg.List() {
		if n.Capabilities.Docker {
			return n.TaiID
		}
	}
	t.Fatal("sandboxtest.DockerNodeID: no Docker-capable node")
	return ""
}

// HostExecNodeID returns the TaiID of the first HostExec-capable node.
func HostExecNodeID(t *testing.T) string {
	t.Helper()
	reg := registry.Global()
	if reg == nil {
		t.Fatal("sandboxtest.HostExecNodeID: registry not initialized")
	}
	for _, n := range reg.List() {
		if n.Capabilities.HostExec {
			return n.TaiID
		}
	}
	t.Fatal("sandboxtest.HostExecNodeID: no HostExec-capable node")
	return ""
}

// ---------------------------------------------------------------------------
// Sandbox stack initialization (called by testprepare.PrepareSandbox)
//
// InitStack manages the full Tai lifecycle within the test process:
//  1. Build tai binary from source (cached)
//  2. Generate credentials with proper JWT (via oauth.OAuth.MakeAccessToken)
//  3. Start two tai sub-processes via os/exec (Docker + HostExec)
//  4. Wait for tunnel registration + ConnResources ready
//  5. Start sandbox.Manager
//  6. t.Cleanup: kill Tai processes + close manager
// ---------------------------------------------------------------------------

func InitStack(t *testing.T, yaoSrcRoot string, grpcPort int) {
	t.Helper()

	reg := registry.Global()
	if reg == nil {
		t.Fatal("sandboxtest.InitStack: registry not initialized")
	}
	if grpcPort == 0 {
		t.Fatal("sandboxtest.InitStack: gRPC port is 0 (server not started?)")
	}

	grpcAddr := fmt.Sprintf("127.0.0.1:%d", grpcPort)

	taiExe := TaiBinaryPath(t, yaoSrcRoot)

	startTaiProcess(t, taiExe, "tai-docker", grpcAddr, false)
	startTaiProcess(t, taiExe, "tai-hostexec", grpcAddr, true)

	waitForTunnelNodes(t, reg, 60*time.Second)

	sandbox.Init()
	m := sandbox.M()
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("sandboxtest.InitStack: sandbox manager start: %v", err)
	}
	registerGlobalCleanup(func() { m.Close() })
}

func startTaiProcess(t *testing.T, taiExe, name, grpcAddr string, hostExec bool) {
	t.Helper()

	baseDir, err := os.MkdirTemp("", "sandboxtest-"+name+"-*")
	if err != nil {
		t.Fatalf("sandboxtest: create temp dir for %s: %v", name, err)
	}

	credPath := generateCredentialInDir(t, name, grpcAddr, baseDir)

	dataDir := filepath.Join(baseDir, "data")
	os.MkdirAll(dataDir, 0o755)

	args := []string{
		"server",
		"-grpc", "127.0.0.1:0",
		"-data", dataDir,
		"-display-name", name,
	}
	if hostExec {
		args = append(args, "-host-exec", "-host-exec-full-access")
	}
	args = append(args, "http://127.0.0.1:0") // server URL (not used, credentials have yao_grpc_addr)

	cmd := execCommand(taiExe, args...)
	cmd.Env = append(os.Environ(), "TAI_CREDENTIALS="+credPath)

	logFile := filepath.Join(dataDir, name+".log")
	lf, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("sandboxtest: create log file for %s: %v", name, err)
	}
	cmd.Stdout = lf
	cmd.Stderr = lf

	if err := cmd.Start(); err != nil {
		lf.Close()
		t.Fatalf("sandboxtest: start %s: %v", name, err)
	}

	t.Logf("sandboxtest: %s started (PID %d, log=%s)", name, cmd.Process.Pid, logFile)

	registerGlobalCleanup(func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
		lf.Close()
		os.RemoveAll(baseDir)
	})
}

func waitForTunnelNodes(t *testing.T, reg *registry.Registry, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	poll := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		var readyCount int
		for _, n := range reg.List() {
			if n.Status != "online" {
				continue
			}
			raw, ok := reg.GetResources(n.TaiID)
			if !ok || raw == nil {
				continue
			}
			readyCount++
			t.Logf("sandboxtest: Tai node %q connected (mode=%s, docker=%v, hostexec=%v)",
				n.TaiID, n.Mode, n.Capabilities.Docker, n.Capabilities.HostExec)
		}
		if readyCount > 0 {
			t.Logf("sandboxtest: %d Tai node(s) ready via tunnel", readyCount)
			return
		}
		time.Sleep(poll)
	}

	nodes := reg.List()
	var status []string
	for _, n := range nodes {
		_, hasRes := reg.GetResources(n.TaiID)
		status = append(status, fmt.Sprintf("%s(status=%s,res=%v)", n.TaiID, n.Status, hasRes))
	}
	t.Fatalf("sandboxtest.InitStack: no Tai node ready within %s. Nodes: %v", timeout, status)
}

// ---------------------------------------------------------------------------
// RegisterHostExec registers a HostExec node for tests that need it
// separately from InitStack. Kept for backward compatibility.
// ---------------------------------------------------------------------------

func RegisterHostExec(t *testing.T) {
	t.Helper()
	RequireHostExec(t)
}

// ---------------------------------------------------------------------------
// Test resource creation (use after PrepareSandbox)
// ---------------------------------------------------------------------------

func EnsureImage(t *testing.T, m *sandbox.Manager, nodeID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := m.EnsureImage(ctx, nodeID, TestImage(), sandbox.ImagePullOptions{}); err != nil {
		t.Fatalf("sandboxtest.EnsureImage(%s, %s): %v", nodeID, TestImage(), err)
	}
}

func CreateBox(t *testing.T, m *sandbox.Manager, nodeID string, opts ...func(*sandbox.CreateOptions)) *sandbox.Box {
	t.Helper()

	co := sandbox.CreateOptions{
		ID:     fmt.Sprintf("sb-test-%d", time.Now().UnixNano()),
		Image:  TestImage(),
		Owner:  "test-user",
		NodeID: nodeID,
	}
	for _, fn := range opts {
		fn(&co)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if co.NodeID != "" {
		if err := m.EnsureImage(ctx, co.NodeID, co.Image, sandbox.ImagePullOptions{}); err != nil {
			t.Fatalf("sandboxtest.CreateBox EnsureImage(%s, %s): %v", co.NodeID, co.Image, err)
		}
	}

	box, err := m.Create(ctx, co)
	if err != nil {
		t.Fatalf("sandboxtest.CreateBox Create: %v", err)
	}
	t.Cleanup(func() {
		cCtx, cCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cCancel()
		if err := m.Remove(cCtx, box.ID()); err != nil {
			t.Logf("sandboxtest.CreateBox cleanup Remove(%s): %v", box.ID(), err)
		}
	})
	return box
}

func CreateHost(t *testing.T, m *sandbox.Manager, taiID string) *sandbox.Host {
	t.Helper()
	host, err := m.Host(context.Background(), taiID)
	if err != nil {
		t.Fatalf("sandboxtest.CreateHost(%s): %v", taiID, err)
	}
	return host
}

// ---------------------------------------------------------------------------
// Purge stale containers (safe: only runs if local Docker is reachable)
// ---------------------------------------------------------------------------

func PurgeStaleContainers(prefixes ...string) {
	if len(prefixes) == 0 {
		prefixes = []string{"sb-"}
	}

	res, err := tai.DialLocal("", "", nil)
	if err != nil || res == nil || res.Runtime == nil {
		return
	}
	defer res.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	containers, err := res.Runtime.List(ctx, tairuntime.ListOptions{All: true})
	if err != nil {
		return
	}
	for _, c := range containers {
		id := c.Name
		if id == "" {
			id = c.ID
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(id, prefix) || strings.HasPrefix(c.Labels["sandbox-id"], prefix) {
				res.Runtime.Remove(ctx, id, true)
				log.Printf("[purge] removed stale container %s", id)
				break
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Command translation (Windows HostExec compatibility)
// ---------------------------------------------------------------------------

func StartV8ForTest(t *testing.T) {
	t.Helper()
	opt := &v8.Option{
		MinSize:        1,
		MaxSize:        5,
		HeapSizeLimit:  1500,
		DefaultTimeout: 200,
		ContextTimeout: 200,
	}
	if err := v8.Start(opt); err != nil {
		t.Fatalf("sandboxtest.StartV8ForTest: %v", err)
	}
	t.Cleanup(func() { v8.Stop() })
}

func TranslateCmd(isWinNative bool, cmd string, args ...string) (string, []string) {
	if !isWinNative {
		return cmd, args
	}
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

// ---------------------------------------------------------------------------
// Backward-compatible helpers (node ID lookup by capability)
// ---------------------------------------------------------------------------

// FindNodeWithCaps finds the first node matching the given capabilities.
func FindNodeWithCaps(docker, hostExec bool) (string, bool) {
	reg := registry.Global()
	if reg == nil {
		return "", false
	}
	for _, n := range reg.List() {
		if docker && !n.Capabilities.Docker {
			continue
		}
		if hostExec && !n.Capabilities.HostExec {
			continue
		}
		if _, ok := reg.GetResources(n.TaiID); !ok {
			continue
		}
		return n.TaiID, true
	}
	return "", false
}

// NodeCapabilities returns capabilities for a registered node.
func NodeCapabilities(taiID string) types.Capabilities {
	reg := registry.Global()
	if reg == nil {
		return types.Capabilities{}
	}
	meta, ok := reg.Get(taiID)
	if !ok {
		return types.Capabilities{}
	}
	return meta.Capabilities
}

// ---------------------------------------------------------------------------
// Backward-compatible address helpers
//
// Old tests used TaiIDFromAddr(TestLocalAddr()) to get the Docker Tai node ID,
// and TaiIDFromAddr(HostExecAddr()) for the HostExec node.
// With tunnel mode, nodes register dynamically and don't have fixed addresses.
// These functions now look up nodes from the registry by capability.
// ---------------------------------------------------------------------------

// TestLocalAddr returns the Tai ID of the Docker-capable node.
// In the new architecture, this is resolved from the registry, not from env.
func TestLocalAddr() string {
	id, ok := FindNodeWithCaps(true, false)
	if ok {
		return id
	}
	return os.Getenv("TAI_DOCKER_ADDR")
}

// HostExecAddr returns the Tai ID of the HostExec-capable node.
func HostExecAddr() string {
	id, ok := FindNodeWithCaps(false, true)
	if ok {
		return id
	}
	return os.Getenv("TAI_HOSTEXEC_ADDR")
}

// TaiIDFromAddr returns the tai ID for a given addr/identifier.
// When called with the result of TestLocalAddr() or HostExecAddr() (which
// now return the actual TaiID), this simply returns the argument unchanged.
func TaiIDFromAddr(addr string) string {
	return addr
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// ParseHostPort splits "host:port" into host and port number.
// Falls back to ("127.0.0.1", 0) on errors.
func ParseHostPort(addr string) (string, int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "127.0.0.1", 0
	}
	port, _ := strconv.Atoi(portStr)
	if host == "" {
		host = "127.0.0.1"
	}
	return host, port
}
