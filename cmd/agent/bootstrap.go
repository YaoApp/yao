package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/yaoapp/yao/agent/eval"
	"github.com/yaoapp/yao/config"
	yaogrpc "github.com/yaoapp/yao/grpc"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/tai/registry"
)

// EvalEnv holds resources created during bootstrap that must be cleaned up.
// Always non-nil after Bootstrap returns (even on partial failure) so
// callers can safely defer env.Shutdown().
type EvalEnv struct {
	GRPCAddr  string
	TaiBin    string
	TaiStatus eval.TaiStatus
	procs     []*os.Process
	cleanup   []func()
}

// BootstrapOpts configures the bootstrap sequence.
type BootstrapOpts struct {
	TaiBin  string
	Verbose bool
}

// Bootstrap initialises the eval runtime: gRPC server, Tai detection,
// credential generation, Tai sub-process startup, and node registration wait.
//
// On partial failure the returned EvalEnv is non-nil so already-started
// resources can still be cleaned up via Shutdown (GAP-14).
func Bootstrap(ctx context.Context, opts BootstrapOpts) (*EvalEnv, error) {
	env := &EvalEnv{}

	if err := startGRPC(env); err != nil {
		return env, fmt.Errorf("gRPC: %w", err)
	}

	detectTai(env, opts.TaiBin, opts.Verbose)

	if env.TaiBin == "" {
		if opts.Verbose {
			color.Yellow("  Tai not found — running with local node only\n")
		}
		return env, nil
	}

	grpcAddr := env.GRPCAddr
	if grpcAddr == "" {
		return env, fmt.Errorf("gRPC address not available for Tai credential")
	}

	credPath, err := generateCredential("eval-tai", grpcAddr)
	if err != nil {
		return env, fmt.Errorf("credential: %w", err)
	}
	env.cleanup = append(env.cleanup, func() { os.RemoveAll(filepath.Dir(credPath)) })

	startTaiSubprocess(ctx, env, "tai-hostexec", credPath, true, opts.Verbose)

	dockerOK := checkDocker(ctx)
	env.TaiStatus.Docker = dockerOK
	if dockerOK {
		startTaiSubprocess(ctx, env, "tai-docker", credPath, false, opts.Verbose)
	}

	if err := waitForNodes(ctx, 60*time.Second, opts.Verbose); err != nil {
		return env, fmt.Errorf("tai nodes: %w", err)
	}

	return env, nil
}

// Shutdown gracefully terminates Tai sub-processes and stops the gRPC server.
// SIGTERM → 3s grace → SIGKILL (GAP-11).
func (e *EvalEnv) Shutdown() {
	if e == nil {
		return
	}

	allSignaled := true
	for _, p := range e.procs {
		if err := p.Signal(syscall.SIGTERM); err != nil {
			_ = p.Kill()
			allSignaled = false
		}
	}

	if allSignaled && len(e.procs) > 0 {
		done := make(chan struct{})
		go func() {
			for _, p := range e.procs {
				p.Wait()
			}
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			for _, p := range e.procs {
				_ = p.Kill()
			}
		}
	}

	yaogrpc.Stop()

	for _, fn := range e.cleanup {
		fn()
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func startGRPC(env *EvalEnv) error {
	if registry.Global() == nil {
		registry.Init(nil)
	}

	cfg := config.Conf
	cfg.GRPC.Enabled = "on"
	cfg.GRPC.Host = "127.0.0.1"
	cfg.GRPC.Port = 0

	if err := yaogrpc.StartServer(cfg); err != nil {
		return err
	}

	addrs := yaogrpc.Addr()
	if len(addrs) == 0 {
		return fmt.Errorf("gRPC started but no listen address")
	}
	env.GRPCAddr = addrs[0]
	return nil
}

func detectTai(env *EvalEnv, explicit string, verbose bool) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			env.TaiBin = explicit
			env.TaiStatus.Bin = explicit
			return
		}
		if verbose {
			color.Yellow("  --tai %s not found, trying $PATH\n", explicit)
		}
	}

	p, err := exec.LookPath("tai")
	if err == nil {
		env.TaiBin = p
		env.TaiStatus.Bin = p
		return
	}
}

func generateCredential(taiID, grpcAddr string) (string, error) {
	if oauth.OAuth == nil {
		return "", fmt.Errorf("oauth not initialized (openapi.Load not called?)")
	}

	token, err := oauth.OAuth.MakeAccessToken(
		"eval-"+taiID,
		"tai:tunnel",
		"eval-"+taiID,
		86400,
		map[string]interface{}{
			"user_id": "eval-user",
			"team_id": "eval-team",
		},
	)
	if err != nil {
		return "", fmt.Errorf("MakeAccessToken: %w", err)
	}

	cred := map[string]interface{}{
		"client_id":     "eval-" + taiID,
		"machine_id":    "eval-" + taiID,
		"server":        "http://127.0.0.1:0",
		"yao_grpc_addr": grpcAddr,
		"access_token":  token,
		"scope":         "tai:tunnel",
		"expires_at":    "2099-01-01T00:00:00Z",
		"registered":    false,
	}

	data, err := json.Marshal(cred)
	if err != nil {
		return "", err
	}

	tmpDir, err := os.MkdirTemp("", "yao-eval-cred-*")
	if err != nil {
		return "", err
	}

	credPath := filepath.Join(tmpDir, "credentials")
	encoded := base64.StdEncoding.EncodeToString(data)
	if err := os.WriteFile(credPath, []byte(encoded), 0o600); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	return credPath, nil
}

func startTaiSubprocess(ctx context.Context, env *EvalEnv, name, credPath string, hostExec, verbose bool) {
	dataDir, err := os.MkdirTemp("", "yao-eval-"+name+"-*")
	if err != nil {
		if verbose {
			color.Yellow("  Failed to create temp dir for %s: %v\n", name, err)
		}
		return
	}
	env.cleanup = append(env.cleanup, func() { os.RemoveAll(dataDir) })

	args := []string{
		"server",
		"-grpc", "127.0.0.1:0",
		"-data", dataDir,
		"-display-name", name,
	}
	if hostExec {
		args = append(args, "-host-exec", "-host-exec-full-access")
	}
	args = append(args, "http://127.0.0.1:0")

	cmd := exec.CommandContext(ctx, env.TaiBin, args...)
	cmd.Env = append(os.Environ(), "TAI_CREDENTIALS="+credPath)

	logFile := filepath.Join(dataDir, name+".log")
	lf, _ := os.Create(logFile)
	if lf != nil {
		cmd.Stdout = lf
		cmd.Stderr = lf
	}

	setProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		if lf != nil {
			lf.Close()
		}
		if verbose {
			color.Yellow("  Failed to start %s: %v\n", name, err)
		}
		return
	}

	if verbose {
		color.Green("  %s started (PID %d)\n", name, cmd.Process.Pid)
	}

	if hostExec {
		env.TaiStatus.HostExec = true
	}
	env.procs = append(env.procs, cmd.Process)
	env.cleanup = append(env.cleanup, func() {
		if lf != nil {
			lf.Close()
		}
	})
}

func checkDocker(ctx context.Context) bool {
	ctx5, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return exec.CommandContext(ctx5, "docker", "info").Run() == nil
}

func waitForNodes(ctx context.Context, timeout time.Duration, verbose bool) error {
	reg := registry.Global()
	if reg == nil {
		return fmt.Errorf("registry not initialized")
	}

	deadline := time.Now().Add(timeout)
	poll := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

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
			if verbose {
				color.Green("  Tai node %q connected (mode=%s)\n", n.TaiID, n.Mode)
			}
		}
		if readyCount > 0 {
			return nil
		}

		time.Sleep(poll)
	}

	return fmt.Errorf("timed out waiting for Tai nodes (%v)", timeout)
}
