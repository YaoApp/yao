//go:build e2e

package tai_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/yao/e2e/env"
	modulecomputer "github.com/yaoapp/yao/modules/computer"
	computertypes "github.com/yaoapp/yao/modules/computer/types"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	taipkg "github.com/yaoapp/yao/tai"
)

var sharedEnv *env.Environment
var sharedErr error

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		sharedEnv.Close()
	}
	os.Exit(code)
}

func testEnv(t *testing.T) *env.Environment {
	t.Helper()
	if sharedEnv == nil {
		sharedEnv, sharedErr = env.NewShared(t)
	}
	if sharedErr != nil {
		t.Fatalf("e2e env: %v", sharedErr)
	}
	return sharedEnv
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestE2ETaiNodeRegistrationLifecycle(t *testing.T) {
	environment := testEnv(t)
	oauthSvc, ok := openapi.Server.OAuth.(*oauth.Service)
	if !ok {
		t.Fatalf("oauth type %T", openapi.Server.OAuth)
	}
	ctx := context.Background()
	computers := modulecomputer.New().WithTokens(oauthSvc)
	provider := taipkg.ComputerProvider{}
	computers.WithTaiNodeProvider(provider)

	connected, err := computers.Connect(ctx, computertypes.ConnectRequest{
		Name: "Tai E2E node", Visibility: computertypes.VisibilityPrivate,
		Scope: computertypes.ScopeUser, UserID: "user_tai_e2e",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = computers.Delete(context.Background(), connected.ComputerID) })

	transportPort := freePort(t)
	server, err := taipkg.NewServer(taipkg.Config{
		TransportPort: transportPort, OAuth: oauthSvc,
		Observer: &taipkg.ComputerObserver{Computers: computers},
	})
	if err != nil {
		t.Fatal(err)
	}
	startCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := server.Start(startCtx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })

	dataDir := t.TempDir()
	t.Logf("tai data dir=%s", dataDir)
	cmd := exec.Command(environment.TaiBin, "start",
		"--node-id", connected.NodeID,
		"--node-token", connected.NodeToken,
		"--server", environment.BaseURL,
		"--data", dataDir,
		"--log-level", "debug", "--log-file", filepath.Join(dataDir, "tai.log"),
	)
	cmd.Env = append(os.Environ(), "TAI_HOME="+filepath.Join(dataDir, "home"))
	cmd.Dir = dataDir
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })

	deadline := time.Now().Add(15 * time.Second)
	var got *computertypes.Computer
	for time.Now().Before(deadline) {
		got, _ = computers.Get(ctx, connected.ComputerID)
		if got != nil && got.Status == computertypes.StatusOnline {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if got == nil || got.Status != computertypes.StatusOnline {
		if raw, err := os.ReadFile(filepath.Join(dataDir, "tai.log")); err == nil {
			lines := strings.Split(string(raw), "\n")
			if len(lines) > 60 {
				lines = lines[len(lines)-60:]
			}
			for _, line := range lines {
				if strings.Contains(line, "workspace recovered") {
					continue
				}
				t.Logf("tai: %s", line)
			}
		}
		t.Fatalf("node did not register: %+v", got)
	}
	runtimes, err := computers.ListRuntimes(ctx, connected.ComputerID)
	if err != nil {
		t.Fatalf("runtimes: %v", err)
	}
	t.Logf("registered node runtimes=%d", len(runtimes))
	if _, err := computers.ListWorkspaces(ctx, connected.ComputerID); err != nil {
		t.Fatalf("workspaces: %v", err)
	}
	if _, err := computers.ListContainers(ctx, connected.ComputerID); err == nil {
		t.Fatal("containers should be unsupported")
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
	deadline = time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		got, _ = computers.Get(ctx, connected.ComputerID)
		if got != nil && got.Status == computertypes.StatusOffline {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if got == nil || got.Status != computertypes.StatusOffline {
		t.Fatalf("node did not disconnect: %+v", got)
	}
}

func TestE2EYaoTransportDiscovery(t *testing.T) {
	environment := testEnv(t)
	oauthSvc, ok := openapi.Server.OAuth.(*oauth.Service)
	if !ok {
		t.Fatalf("oauth type %T", openapi.Server.OAuth)
	}
	transportPort := freePort(t)
	server, err := taipkg.NewServer(taipkg.Config{TransportPort: transportPort, OAuth: oauthSvc})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })

	resp, body, err := environment.Get("/.well-known/yao-transport")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), fmt.Sprintf("\"transport_port\":%d", transportPort)) {
		t.Fatalf("body=%s", body)
	}
}
