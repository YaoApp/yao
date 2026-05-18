package testprepare

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"
)

var (
	unitOnce    sync.Once
	unitErr     error
	appOnce     sync.Once
	sandboxOnce sync.Once
	yaoSrcRoot  string
	agentAppDir string

	globalCleanups []func()
	cleanupMu      sync.Mutex
)

func init() {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("testprepare: cannot determine source file location")
	}
	// thisFile = .../yao/unit-test/agent/testprepare/prepare.go
	yaoSrcRoot = filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	agentAppDir = filepath.Join(yaoSrcRoot, "unit-test", "agent", "app")

	sandboxtest.CleanupRegistrar = RegisterCleanup
}

// YaoSrcRoot returns the absolute path to the yao source root.
func YaoSrcRoot() string { return yaoSrcRoot }

// AgentAppDir returns the absolute path to the test application directory.
func AgentAppDir() string { return agentAppDir }

// PrepareUnit loads agent-test.env, generates app/.env, and validates
// basic paths. It does NOT start any services (DB, Tai, LLM, V8).
// Suitable for pure-function unit tests that have no external dependencies.
//
// Fails with t.Fatal if the env file is missing or the app directory
// does not exist. Never calls t.Skip.
func PrepareUnit(t *testing.T) {
	t.Helper()
	unitOnce.Do(func() {
		unitErr = loadAgentTestEnv()
	})
	if unitErr != nil {
		t.Fatalf("testprepare.PrepareUnit: %v", unitErr)
	}

	if _, err := os.Stat(filepath.Join(agentAppDir, "app.yao")); os.IsNotExist(err) {
		t.Fatalf("testprepare.PrepareUnit: app.yao not found in %s", agentAppDir)
	}
}

// PrepareSandbox performs full Yao Runtime loading (DB, models, scripts,
// V8, Agent stack) from the test application, then initializes Tai registry
// and sandbox manager.
//
// Returns *TestIdentity containing the database-generated Team/User IDs
// for Alpha (mock) and Beta (real) teams. Tests should use these IDs
// in AuthorizedInfo instead of hardcoded strings.
//
// Yao is a Runtime — the test process must load the complete application
// in-process. loadApp() calls config.Init() which reads app/.env via
// godotenv.Overload, making app/.env the authoritative configuration source.
//
// Internally uses sync.Once so that heavy initialization runs exactly once
// even when multiple test functions call PrepareSandbox.
//
// Fails with t.Fatal if any required service is unavailable.
// Never calls t.Skip.
func PrepareSandbox(t *testing.T) *TestIdentity {
	t.Helper()
	PrepareUnit(t)

	// Full application load: DB + models + scripts + V8 + Agent stack
	appOnce.Do(func() {
		loadApp(t)
	})

	// Tai registry + sandbox manager
	sandboxOnce.Do(func() {
		sandboxtest.InitStack(t, yaoSrcRoot, TestGRPCPort())
	})

	return cachedIdentity
}

// PrepareE2E extends PrepareSandbox with LLM availability validation.
// Suitable for tests that need Tai + real/mock LLM.
//
// Returns *TestIdentity — use identity.BetaTeamID / identity.BetaOwnerUserID
// for E2E tests (real LLM connectors via Team role matrix).
//
// Checks mock-llm health via 127.0.0.1 (avoids DNS dependency), then
// verifies host.tai.internal is resolvable so containers and HostExec
// sub-processes can reach the mock server through the same hostname.
//
// Fails with t.Fatal if no LLM endpoint is reachable.
// Never calls t.Skip.
func PrepareE2E(t *testing.T) *TestIdentity {
	t.Helper()
	identity := PrepareSandbox(t)

	if identity == nil {
		t.Fatal("testprepare.PrepareE2E: test identity not available — SetupTestUsers may have failed")
	}

	// Step 1: check mock-llm process via localhost (no DNS dependency).
	port := os.Getenv("MOCK_LLM_PORT")
	if port != "" {
		healthURL := "http://127.0.0.1:" + port + "/healthz"
		client := &http.Client{Timeout: 3 * time.Second}
		if resp, err := client.Get(healthURL); err == nil && resp.StatusCode == http.StatusOK {
			// Step 2: verify host.tai.internal resolves on this machine.
			checkTaiHostResolvable(t)
			return identity
		}
	}

	llmKeys := []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "DEEPSEEK_V4_API_KEY"}
	for _, k := range llmKeys {
		if os.Getenv(k) != "" {
			return identity
		}
	}

	t.Fatal("testprepare.PrepareE2E: no LLM available — " +
		"mock-llm is not running on 127.0.0.1:" + port + " and no API keys found " +
		"(checked OPENAI_API_KEY, ANTHROPIC_API_KEY, DEEPSEEK_V4_API_KEY)")
	return nil
}

const taiInternalHost = "host.tai.internal"

func checkTaiHostResolvable(t *testing.T) {
	t.Helper()
	if _, err := net.LookupHost(taiInternalHost); err != nil {
		t.Fatalf("testprepare.PrepareE2E: %s cannot be resolved.\n"+
			"Docker containers and HostExec processes need this hostname to reach mock-llm.\n"+
			"Add to /etc/hosts (Linux/macOS) or %%SystemRoot%%\\System32\\drivers\\etc\\hosts (Windows):\n"+
			"  127.0.0.1 %s", taiInternalHost, taiInternalHost)
	}
}

// RegisterCleanup adds a function to run during Cleanup().
// This is for shared resources that outlive individual tests.
func RegisterCleanup(fn func()) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	globalCleanups = append(globalCleanups, fn)
}

// Cleanup runs all registered cleanup functions in reverse order.
// Call from TestMain after m.Run() returns.
func Cleanup() {
	cleanupMu.Lock()
	fns := make([]func(), len(globalCleanups))
	copy(fns, globalCleanups)
	globalCleanups = nil
	cleanupMu.Unlock()

	for i := len(fns) - 1; i >= 0; i-- {
		fns[i]()
	}
}

// MustLoadEnv loads agent-test.env without a *testing.T.
// Use in TestMain or init() where no T is available.
// Panics on failure.
func MustLoadEnv() {
	unitOnce.Do(func() {
		unitErr = loadAgentTestEnv()
	})
	if unitErr != nil {
		panic(fmt.Sprintf("testprepare.MustLoadEnv: %v", unitErr))
	}
}

// loadAgentTestEnv reads agent-test.env, generates app/.env, and sets
// process environment variables unconditionally (env file always wins).
//
// Steps:
//  1. Parse agent-test.env — set ALL variables via os.Setenv (unconditional)
//  2. Generate app/.env — write all key=val lines except TEST_*/SANDBOX_TEST_*
//     so that Yao's config.Init() and connector $ENV.* resolution work correctly
//  3. Set path variables: YAO_ROOT, YAO_AGENT_TEST_APPLICATION, YAO_TEST_APPLICATION
func loadAgentTestEnv() error {
	agentDir := filepath.Join(yaoSrcRoot, "unit-test", "agent")

	envFile := filepath.Join(agentDir, "env", "agent-test.env")
	if testDB := os.Getenv("YAO_TEST_DB"); testDB == "postgres" {
		pgFile := filepath.Join(agentDir, "env", "agent-test-pg.env")
		if _, err := os.Stat(pgFile); err == nil {
			envFile = pgFile
			fmt.Printf("[testprepare] YAO_TEST_DB=postgres → using %s\n", pgFile)
		} else {
			fmt.Printf("[testprepare] YAO_TEST_DB=postgres but %s not found, falling back to agent-test.env\n", pgFile)
		}
	}

	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf("env file not found at %s", envFile)
	}

	// Parse all key=val pairs from agent-test.env
	pairs, err := parseEnvFile(envFile)
	if err != nil {
		return err
	}

	// Load .env.local overlay (API keys, local DB overrides)
	localFile := filepath.Join(agentDir, "env", ".env.local")
	if _, err := os.Stat(localFile); err == nil {
		localPairs, err := parseEnvFile(localFile)
		if err != nil {
			return fmt.Errorf("read .env.local: %w", err)
		}
		for k, v := range localPairs {
			pairs[k] = v
		}
	}

	// Isolate SQLite DB per process to avoid cross-package corruption when
	// go test runs multiple packages in parallel (each as a separate OS process).
	// Only applies to sqlite3 — Postgres DSN (postgres://...) must not be modified.
	if driver, _ := pairs["YAO_DB_DRIVER"]; driver == "sqlite3" || driver == "" {
		if v, ok := pairs["YAO_DB_PRIMARY"]; ok && !filepath.IsAbs(v) {
			ext := filepath.Ext(v)
			base := strings.TrimSuffix(v, ext)
			pairs["YAO_DB_PRIMARY"] = fmt.Sprintf("%s-%d%s", base, os.Getpid(), ext)
		}
	}

	// Set all variables unconditionally (env file always wins)
	for k, v := range pairs {
		os.Setenv(k, v)
	}

	// Generate app/.env for Yao runtime
	if err := generateAppEnv(pairs); err != nil {
		return fmt.Errorf("generate app/.env: %w", err)
	}

	// Set path variables
	os.Setenv("YAO_ROOT", agentAppDir)
	os.Setenv("YAO_AGENT_TEST_APPLICATION", agentAppDir)
	os.Setenv("YAO_TEST_APPLICATION", agentAppDir)

	return nil
}

// generateAppEnv writes app/.env from the merged key-val pairs.
// Skips TEST_* and SANDBOX_TEST_* keys which are test-orchestration only.
func generateAppEnv(pairs map[string]string) error {
	target := filepath.Join(agentAppDir, ".env")
	os.MkdirAll(filepath.Dir(target), 0o755)

	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	// Read original file to preserve key order
	envFile := filepath.Join(agentAppDir, "..", "env", "agent-test.env")
	ordered, err := orderedKeys(envFile)
	if err != nil {
		// Fallback: write in arbitrary order
		for k, v := range pairs {
			if strings.HasPrefix(k, "TEST_") || strings.HasPrefix(k, "SANDBOX_TEST_") {
				continue
			}
			fmt.Fprintf(f, "%s=%s\n", k, v)
		}
		return nil
	}

	written := make(map[string]bool)
	for _, k := range ordered {
		if strings.HasPrefix(k, "TEST_") || strings.HasPrefix(k, "SANDBOX_TEST_") {
			continue
		}
		v, ok := pairs[k]
		if !ok {
			continue
		}
		fmt.Fprintf(f, "%s=%s\n", k, v)
		written[k] = true
	}
	// Write any keys from .env.local that aren't in agent-test.env
	for k, v := range pairs {
		if written[k] {
			continue
		}
		if strings.HasPrefix(k, "TEST_") || strings.HasPrefix(k, "SANDBOX_TEST_") {
			continue
		}
		fmt.Fprintf(f, "%s=%s\n", k, v)
	}
	return nil
}

// parseEnvFile reads a dotenv-style file into a map.
// Lines starting with # are comments; inline comments after # are stripped.
func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Strip inline comments
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if key == "" {
			continue
		}
		result[key] = val
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return result, nil
}

// orderedKeys returns keys in file order from a dotenv file.
func orderedKeys(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var keys []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys, scanner.Err()
}
