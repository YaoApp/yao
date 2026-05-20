package sandboxtest

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yaoapp/yao/openapi/oauth"
)

// GenerateCredential creates a Tai credential file (base64-encoded JSON)
// using the loaded OAuth service to produce a properly signed JWT.
// The returned path points to the written credentials file.
func GenerateCredential(t *testing.T, taiID string, grpcAddr string) string {
	t.Helper()
	return generateCredentialInDir(t, taiID, grpcAddr, t.TempDir())
}

func generateCredentialInDir(t *testing.T, taiID, grpcAddr, baseDir string) string {
	t.Helper()

	if oauth.OAuth == nil {
		t.Fatal("sandboxtest.GenerateCredential: oauth.OAuth not initialized (openapi.Load not called?)")
	}

	token, err := oauth.OAuth.MakeAccessToken(
		"ci-"+taiID,
		"tai:tunnel",
		"ci-"+taiID,
		86400, // 24h
		map[string]interface{}{
			"user_id": "ci-test-user",
			"team_id": "ci-test-team",
		},
	)
	if err != nil {
		t.Fatalf("sandboxtest.GenerateCredential: MakeAccessToken: %v", err)
	}

	cred := map[string]interface{}{
		"client_id":     "ci-" + taiID,
		"machine_id":    "ci-" + taiID,
		"server":        "http://127.0.0.1:0",
		"yao_grpc_addr": grpcAddr,
		"access_token":  token,
		"scope":         "tai:tunnel",
		"expires_at":    "2099-01-01T00:00:00Z",
		"registered":    false,
	}

	data, err := json.Marshal(cred)
	if err != nil {
		t.Fatalf("sandboxtest.GenerateCredential: marshal: %v", err)
	}

	credDir := filepath.Join(baseDir, "cred")
	if err := os.MkdirAll(credDir, 0o755); err != nil {
		t.Fatalf("sandboxtest.GenerateCredential: mkdir: %v", err)
	}

	credPath := filepath.Join(credDir, "credentials")
	encoded := base64.StdEncoding.EncodeToString(data)
	if err := os.WriteFile(credPath, []byte(encoded), 0o600); err != nil {
		t.Fatalf("sandboxtest.GenerateCredential: write: %v", err)
	}

	t.Logf("sandboxtest: credential for %s written to %s (grpc=%s)", taiID, credPath, grpcAddr)
	return credPath
}

// TaiBinaryPath returns the path to a pre-built tai binary.
// It checks $TAI_BINARY first, then builds from source if needed.
func TaiBinaryPath(t *testing.T, yaoSrcRoot string) string {
	t.Helper()

	if p := os.Getenv("TAI_BINARY"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return buildTaiBinary(t, yaoSrcRoot)
}

func buildTaiBinary(t *testing.T, yaoSrcRoot string) string {
	t.Helper()

	taiSrc := filepath.Join(yaoSrcRoot, "..", "tai")
	if _, err := os.Stat(filepath.Join(taiSrc, "go.mod")); err != nil {
		t.Fatalf("sandboxtest: tai source not found at %s", taiSrc)
	}

	buildDir := filepath.Join(yaoSrcRoot, ".build", "test")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatalf("sandboxtest: mkdir build dir: %v", err)
	}

	exe := filepath.Join(buildDir, "tai-sandbox-test")
	if isWindows() {
		exe += ".exe"
	}

	if info, err := os.Stat(exe); err == nil {
		srcInfo, srcErr := os.Stat(filepath.Join(taiSrc, "main.go"))
		if srcErr == nil && info.ModTime().After(srcInfo.ModTime()) {
			t.Logf("sandboxtest: reusing cached tai binary at %s", exe)
			return exe
		}
	}

	t.Logf("sandboxtest: building tai from %s ...", taiSrc)
	cmd := exec.Command("go", "build", "-o", exe, ".")
	cmd.Dir = taiSrc
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sandboxtest: build tai failed: %v\n%s", err, out)
	}
	t.Logf("sandboxtest: tai binary built at %s", exe)
	return exe
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}
