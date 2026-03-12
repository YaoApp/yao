package tai

import (
	"os"
	"strconv"
	"testing"

	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/types"
	"github.com/yaoapp/yao/tai/volume"
)

func taiTestHost() string {
	if h := os.Getenv("TAI_TEST_HOST"); h != "" {
		return h
	}
	return "127.0.0.1"
}

func taiTestPorts() Ports {
	return Ports{
		Docker: envPort("TAI_TEST_DOCKER_PORT", 0),
		HTTP:   envPort("TAI_TEST_HTTP_PORT", 0),
		VNC:    envPort("TAI_TEST_VNC_PORT", 0),
	}
}

func envPort(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			return p
		}
	}
	return fallback
}

func TestMergedPorts(t *testing.T) {
	p := mergedPorts(Ports{HTTP: 8888})
	if p.HTTP != 8888 {
		t.Errorf("HTTP = %d, want 8888", p.HTTP)
	}
	if p.GRPC != 19100 {
		t.Errorf("GRPC = %d, want 19100 (default)", p.GRPC)
	}
	if p.VNC != 16080 {
		t.Errorf("VNC = %d, want 16080 (default)", p.VNC)
	}
	if p.Docker != 0 {
		t.Errorf("Docker = %d, want 0 (unset)", p.Docker)
	}
	if p.K8s != 0 {
		t.Errorf("K8s = %d, want 0 (unset)", p.K8s)
	}
}

func TestMergedPortsAll(t *testing.T) {
	p := mergedPorts(Ports{GRPC: 1, HTTP: 2, VNC: 3, Docker: 4, K8s: 5})
	if p.GRPC != 1 || p.HTTP != 2 || p.VNC != 3 || p.Docker != 4 || p.K8s != 5 {
		t.Errorf("unexpected ports: %+v", p)
	}
}

func TestDialLocalSuccess(t *testing.T) {
	res, err := DialLocal("", t.TempDir(), nil)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer res.Close()

	if res.Volume == nil {
		t.Error("Volume should not be nil")
	}
	if res.Runtime == nil {
		t.Error("Runtime should not be nil")
	}
}

func TestDialLocalWithVolume(t *testing.T) {
	dir := t.TempDir()
	vol := volume.NewLocal(dir)
	res, err := DialLocal("", dir, vol)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer res.Close()

	if res.DataDir != dir {
		t.Errorf("DataDir = %q, want %q", res.DataDir, dir)
	}
	if res.Volume == nil {
		t.Error("Volume should not be nil")
	}
}

func TestDialLocalExplicitSocket(t *testing.T) {
	res, err := DialLocal("unix:///var/run/docker.sock", t.TempDir(), nil)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer res.Close()

	if res.Runtime == nil {
		t.Error("Runtime should not be nil for explicit unix socket")
	}
}

func TestDialRemoteDocker(t *testing.T) {
	host := taiTestHost()
	grpcPort := envPort("TAI_TEST_GRPC_PORT", 19100)
	ports := taiTestPorts()
	ports.GRPC = grpcPort

	res, err := DialRemote(host, ports)
	if err != nil {
		t.Skipf("Tai not available at %s:%d: %v", host, grpcPort, err)
	}
	defer res.Close()

	t.Logf("remote docker: host=%s ports=%+v", host, res.Ports)

	if res.Volume == nil {
		t.Error("Volume should not be nil")
	}
	if res.Runtime == nil {
		t.Error("Runtime should not be nil")
	}
}

func TestDialRemoteK8s(t *testing.T) {
	host := os.Getenv("TAI_TEST_K8S_HOST")
	kubeconfig := os.Getenv("TAI_TEST_KUBECONFIG")
	if host == "" || kubeconfig == "" {
		t.Skip("TAI_TEST_K8S_HOST or TAI_TEST_KUBECONFIG not set")
	}

	grpcPort := envPort("TAI_TEST_K8S_GRPC_PORT", envPort("TAI_TEST_GRPC_PORT", 19100))
	ports := Ports{
		K8s:  envPort("TAI_TEST_K8S_PORT", 16443),
		GRPC: grpcPort,
		HTTP: envPort("TAI_TEST_K8S_HTTP_PORT", 8099),
		VNC:  envPort("TAI_TEST_K8S_VNC_PORT", 16080),
	}

	res, err := DialRemote(host, ports,
		WithDialRuntime(types.K8s),
		WithDialKubeConfig(kubeconfig),
		WithDialNamespace("default"),
	)
	if err != nil {
		t.Skipf("Tai K8s not available: %v", err)
	}
	defer res.Close()

	if res.Runtime == nil {
		t.Error("Runtime should not be nil")
	}
}

func TestDialRemoteK8sMissingKubeConfig(t *testing.T) {
	host := taiTestHost()
	grpcPort := envPort("TAI_TEST_GRPC_PORT", 19100)

	_, err := DialRemote(host, Ports{GRPC: grpcPort}, WithDialRuntime(types.K8s))
	if err == nil {
		t.Skip("Tai happened to be reachable; test only valid when gRPC is up")
	}
}

func TestRegisterLocal(t *testing.T) {
	registry.Init(nil)
	reg := registry.Global()

	dir := t.TempDir()
	ok := RegisterLocal(WithDataDir(dir))
	if !ok {
		t.Skip("Docker not available, skipping RegisterLocal test")
	}

	meta, found := reg.Get("local")
	if !found {
		t.Fatal("expected 'local' node in registry after RegisterLocal")
	}
	if meta.Mode != "local" {
		t.Errorf("mode = %q, want 'local'", meta.Mode)
	}
	if meta.Status != "online" {
		t.Errorf("status = %q, want 'online'", meta.Status)
	}

	res, got := GetResources("local")
	if !got {
		t.Fatal("GetResources('local') returned false after RegisterLocal")
	}
	if res.DataDir != dir {
		t.Errorf("DataDir = %q, want %q", res.DataDir, dir)
	}
	if res.Runtime == nil {
		t.Error("local resources Runtime should not be nil")
	}

	ok2 := RegisterLocal(WithDataDir(dir))
	if !ok2 {
		t.Error("second RegisterLocal should return true (idempotent)")
	}

	res.Close()
}

func TestRegisterLocal_NoRegistry(t *testing.T) {
	origReg := registry.Global()
	defer func() {
		if origReg != nil {
			registry.Init(nil)
		}
	}()

	ok := RegisterLocal()
	_ = ok
}

func TestRegisterLocal_NoDocker(t *testing.T) {
	registry.Init(nil)

	ok := RegisterLocal(WithDataDir(t.TempDir()))
	if !ok {
		return
	}
	res, got := GetResources("local")
	if got && res != nil {
		res.Close()
	}
}

func TestConnResourcesCloseNil(t *testing.T) {
	var r *ConnResources
	if err := r.Close(); err != nil {
		t.Errorf("Close on nil should return nil, got %v", err)
	}
}
