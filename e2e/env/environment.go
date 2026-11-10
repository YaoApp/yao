package env

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/e2e"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/seed"
	"github.com/yaoapp/yao/test"
)

// Environment is one isolated E2E application environment.
type Environment struct {
	Options    Options
	Root       string
	AppRoot    string
	RunDir     string
	BaseURL    string
	DB         Database
	TaiBin     string
	TaiVersion string
	dbDriver   string
	dbName     string
	prepared   bool

	httpServer *http.Server
	listener   net.Listener
	httpLog    *os.File
}

// New materializes the test app, creates an isolated DB, loads Yao and starts HTTP.
func New(t *testing.T) (*Environment, error) {
	t.Helper()
	opts, err := LoadOptions()
	if err != nil {
		return nil, err
	}
	env, err := NewWithOptions(t, opts)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		env.Close()
		if t.Failed() {
			t.Logf("E2E artifacts: %s", env.RunDir)
		}
	})
	return env, nil
}

// NewShared creates an environment without registering a t.Cleanup callback.
// The caller (typically TestMain) owns Close.
func NewShared(t *testing.T) (*Environment, error) {
	t.Helper()
	opts, err := LoadOptions()
	if err != nil {
		return nil, err
	}
	return NewWithOptions(t, opts)
}

// NewWithOptions is New with explicit options.
func NewWithOptions(t *testing.T, opts Options) (*Environment, error) {
	t.Helper()
	root, err := os.MkdirTemp("", "yao-e2e-*")
	if err != nil {
		return nil, err
	}
	appRoot := filepath.Join(root, "app")
	if err := e2e.Materialize(appRoot); err != nil {
		os.RemoveAll(root)
		return nil, err
	}
	if err := ensureSigningCerts(appRoot); err != nil {
		os.RemoveAll(root)
		return nil, err
	}

	runDir := filepath.Join(opts.ArtifactsDir, opts.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		os.RemoveAll(root)
		return nil, err
	}
	if err := writeLatest(opts.ArtifactsDir, opts.RunID); err != nil {
		os.RemoveAll(root)
		return nil, err
	}

	db, err := newDatabase(opts, root)
	if err != nil {
		os.RemoveAll(root)
		return nil, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		db.Close()
		os.RemoveAll(root)
		return nil, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	env := &Environment{
		Options:  opts,
		Root:     root,
		AppRoot:  appRoot,
		RunDir:   runDir,
		DB:       db,
		dbDriver: db.Driver(),
		dbName:   db.Name(),
		listener: listener,
	}
	if err := env.writeAppEnv(port); err != nil {
		env.Close()
		return nil, err
	}
	if err := env.loadApp(t); err != nil {
		env.Close()
		return nil, err
	}
	if err := env.startHTTP(); err != nil {
		env.Close()
		return nil, err
	}
	if err := env.seed(); err != nil {
		env.Close()
		return nil, err
	}
	if err := env.resolveTai(); err != nil {
		env.Close()
		return nil, err
	}
	if err := env.writeRunJSON("running"); err != nil {
		env.Close()
		return nil, err
	}
	return env, nil
}

func (e *Environment) writeAppEnv(port int) error {
	dataDir := filepath.Join(e.AppRoot, "data")
	logDir := filepath.Join(e.AppRoot, "logs")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"yao.log", "tai.log", "http.log"} {
		if err := os.WriteFile(filepath.Join(e.RunDir, name), nil, 0o644); err != nil {
			return err
		}
	}
	values := map[string]string{
		"YAO_ENV":           "test",
		"YAO_ROOT":          e.AppRoot,
		"YAO_PORT":          fmt.Sprintf("%d", port),
		"YAO_HOST":          "127.0.0.1",
		"YAO_DB_DRIVER":     e.DB.Driver(),
		"YAO_DB_PRIMARY":    e.DB.DSN(),
		"YAO_SESSION_STORE": "file",
		"YAO_LOG":           filepath.Join(e.RunDir, "yao.log"),
		"YAO_LOG_MODE":      "TEXT",
		"YAO_LANG":          "en",
		"YAO_CLIENT_ID":     "yao_e2e_client_0123456789abcdefg",
		"YAO_CLIENT_SECRET": "",
		"YAO_TAI_BIN":       "",
	}
	var b strings.Builder
	for key, value := range values {
		fmt.Fprintf(&b, "%s=%s\n", key, value)
	}
	return os.WriteFile(filepath.Join(e.AppRoot, ".env"), []byte(b.String()), 0o600)
}

func (e *Environment) loadApp(t *testing.T) error {
	t.Helper()
	for key, value := range map[string]string{
		"YAO_ROOT":             e.AppRoot,
		"YAO_TEST_APPLICATION": e.AppRoot,
	} {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	config.Init()
	test.Prepare(t, config.Conf)
	e.prepared = true
	if _, err := openapi.Load(config.Conf); err != nil {
		return fmt.Errorf("load openapi: %w", err)
	}
	if openapi.Server == nil {
		return fmt.Errorf("openapi server not initialized")
	}
	return nil
}

func (e *Environment) startHTTP() error {
	listener := e.listener
	if listener == nil {
		var err error
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
		e.listener = listener
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		started := time.Now()
		c.Next()
		if e.httpLog != nil {
			fmt.Fprintf(e.httpLog, "%s %s %d %s\n", c.Request.Method, c.Request.URL.RequestURI(), c.Writer.Status(), time.Since(started))
		}
	})
	openapi.Server.Attach(router)
	server := &http.Server{Handler: router}
	go func() {
		_ = server.Serve(listener)
	}()
	e.listener = listener
	e.httpServer = server
	e.httpLog, _ = os.OpenFile(filepath.Join(e.RunDir, "http.log"), os.O_APPEND|os.O_WRONLY, 0o644)
	e.BaseURL = "http://" + listener.Addr().String()
	return nil
}

func (e *Environment) seed() error {
	options := seed.ImportOption{ChunkSize: 100, Duplicate: seed.DuplicateIgnore, Mode: seed.ImportModeBatch}
	for _, item := range []struct {
		file  string
		model string
	}{
		{"roles.csv", "__yao.role"},
		{"types.csv", "__yao.user.type"},
	} {
		result, err := seed.Import(item.file, item.model, options)
		if err != nil {
			return fmt.Errorf("seed %s: %w", item.file, err)
		}
		if result.Failure > 0 {
			return fmt.Errorf("seed %s failed: %d", item.file, result.Failure)
		}
	}
	svc, ok := openapi.Server.OAuth.(*oauth.Service)
	if !ok {
		return fmt.Errorf("unexpected oauth service type %T", openapi.Server.OAuth)
	}
	provider, err := svc.GetUserProvider()
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, user := range []struct {
		email string
		role  string
	}{
		{"admin@e2e.local", "system:admin"},
		{"user@e2e.local", "user:free"},
	} {
		if _, err := provider.CreateUser(ctx, maps.MapStrAny{
			"email":         user.email,
			"name":          user.email,
			"password_hash": "YaoE2E2026!",
			"role_id":       user.role,
			"type_id":       "free",
			"status":        "active",
		}); err != nil {
			return fmt.Errorf("seed user %s: %w", user.email, err)
		}
	}
	return nil
}

func (e *Environment) resolveTai() error {
	path, err := taiBinary(e.Options)
	if err != nil {
		return err
	}
	cmd := exec.Command(path, "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tai version failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	e.TaiBin = path
	e.TaiVersion = strings.TrimSpace(string(out))
	_ = os.Setenv("YAO_TAI_BIN", path)
	_ = os.Setenv("TAI_BIN", path)
	return nil
}

func (e *Environment) writeRunJSON(result string) error {
	info := map[string]interface{}{
		"run_id":     e.Options.RunID,
		"goos":       platformName(),
		"go_version": runtimeVersion(),
		"db_driver":  e.dbDriver,
		"db_name":    e.dbName,
		"tai_bin":    e.TaiBin,
		"started_at": time.Now().UTC().Format(time.RFC3339),
		"result":     result,
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(e.RunDir, "run.json"), data, 0o644)
}

// Close tears down HTTP, Yao, DB and temporary app in reverse order.
func (e *Environment) Close() {
	if e.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = e.httpServer.Shutdown(ctx)
		cancel()
		e.httpServer = nil
	}
	if e.listener != nil {
		_ = e.listener.Close()
		e.listener = nil
	}
	if e.httpLog != nil {
		_ = e.httpLog.Close()
		e.httpLog = nil
	}
	if e.prepared {
		test.Clean()
		openapi.Server = nil
		e.prepared = false
	}
	if e.DB != nil {
		_ = e.DB.Close()
		e.DB = nil
	}
	if e.Root != "" && !e.Options.Keep {
		_ = os.RemoveAll(e.Root)
	}
	_ = e.writeRunJSON("finished")
}

func (e *Environment) do(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

// Get performs an HTTP GET against the test app.
func (e *Environment) Get(path string) (*http.Response, []byte, error) {
	resp, err := e.do(mustRequest(http.MethodGet, e.BaseURL+path))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp, body, err
}

func mustRequest(method, url string) *http.Request {
	req, _ := http.NewRequest(method, url, nil)
	return req
}

func taiBinary(opts Options) (string, error) {
	if opts.TaiBin != "" {
		if _, err := os.Stat(opts.TaiBin); err != nil {
			return "", fmt.Errorf("YAO_E2E_TAI_BIN not found: %s", opts.TaiBin)
		}
		return opts.TaiBin, nil
	}
	exe := "tai"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	for _, binRoot := range []string{
		filepath.Join(opts.RepoRoot, "bin", "tai"),
		filepath.Join(filepath.Dir(opts.RepoRoot), "bin", "tai"),
	} {
		latest, err := os.ReadFile(filepath.Join(binRoot, "LATEST"))
		if err != nil {
			continue
		}
		sha := strings.TrimSpace(string(latest))
		path := filepath.Join(binRoot, sha, runtime.GOOS+"-"+runtime.GOARCH, exe)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("tai artifact missing for %s/%s; run scripts/build-tai-all.sh", runtime.GOOS, runtime.GOARCH)
}

func writeLatest(dir, runID string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "LATEST"), []byte(runID+"\n"), 0o644)
}

func runtimeVersion() string {
	return strings.TrimPrefix(runtime.Version(), "go")
}
