package sandbox

import (
	"context"
	"fmt"

	hepb "github.com/yaoapp/yao/tai/hostexec/pb"
	"github.com/yaoapp/yao/tai/workspace"
)

// Host represents a Tai host machine execution environment.
// Unlike Box (which wraps a container), Host executes commands directly on
// the Tai server's OS via HostExec gRPC and accesses files via Volume gRPC.
//
// A Host is bound to a pool and does not require Create — it is available as
// long as the pool's Tai server reports host_exec capability.
type Host struct {
	pool    string
	manager *Manager
}

// Pool returns the pool name this Host belongs to.
func (h *Host) Pool() string { return h.pool }

// Exec runs a command on the Tai host machine via HostExec gRPC.
func (h *Host) Exec(ctx context.Context, cmd string, args []string, opts ...HostExecOption) (*HostExecResult, error) {
	client, err := h.manager.getPool(h.pool)
	if err != nil {
		return nil, err
	}

	he := client.HostExec()
	if he == nil {
		return nil, fmt.Errorf("sandbox: host_exec not available on pool %q", h.pool)
	}

	cfg := &hostExecConfig{}
	for _, o := range opts {
		o(cfg)
	}

	req := &hepb.ExecRequest{
		Command:        cmd,
		Args:           args,
		WorkingDir:     cfg.WorkDir,
		Stdin:          cfg.Stdin,
		TimeoutMs:      cfg.TimeoutMs,
		MaxOutputBytes: cfg.MaxOutputBytes,
	}
	if cfg.Env != nil {
		req.Env = cfg.Env
	}

	resp, err := he.Exec(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("hostexec rpc: %w", err)
	}

	return &HostExecResult{
		ExitCode:   int(resp.ExitCode),
		Stdout:     resp.Stdout,
		Stderr:     resp.Stderr,
		DurationMs: resp.DurationMs,
		Error:      resp.Error,
		Truncated:  resp.Truncated,
	}, nil
}

// Stream runs a command on the Tai host and streams stdout/stderr in real time
// via HostExec gRPC ExecStream. Returns a HostExecStream with separate channels
// for stdout and stderr, plus Wait (blocks until exit) and Cancel.
func (h *Host) Stream(ctx context.Context, cmd string, args []string, opts ...HostExecOption) (*HostExecStream, error) {
	client, err := h.manager.getPool(h.pool)
	if err != nil {
		return nil, err
	}

	he := client.HostExec()
	if he == nil {
		return nil, fmt.Errorf("sandbox: host_exec not available on pool %q", h.pool)
	}

	cfg := &hostExecConfig{}
	for _, o := range opts {
		o(cfg)
	}

	req := &hepb.ExecRequest{
		Command:        cmd,
		Args:           args,
		WorkingDir:     cfg.WorkDir,
		Stdin:          cfg.Stdin,
		TimeoutMs:      cfg.TimeoutMs,
		MaxOutputBytes: cfg.MaxOutputBytes,
	}
	if cfg.Env != nil {
		req.Env = cfg.Env
	}

	streamCtx, cancel := context.WithCancel(ctx)
	rpcStream, err := he.ExecStream(streamCtx, req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("hostexec stream rpc: %w", err)
	}

	stdoutCh := make(chan []byte, 64)
	stderrCh := make(chan []byte, 64)
	doneCh := make(chan struct{})
	var exitCode int
	var exitErr error

	go func() {
		defer close(stdoutCh)
		defer close(stderrCh)
		defer close(doneCh)
		for {
			msg, err := rpcStream.Recv()
			if err != nil {
				exitErr = fmt.Errorf("hostexec stream recv: %w", err)
				return
			}
			if len(msg.Data) > 0 {
				switch msg.Stream {
				case hepb.ExecOutput_STDOUT:
					stdoutCh <- msg.Data
				case hepb.ExecOutput_STDERR:
					stderrCh <- msg.Data
				}
			}
			if msg.Done {
				exitCode = int(msg.ExitCode)
				if msg.Error != "" {
					exitErr = fmt.Errorf("hostexec: %s", msg.Error)
				}
				return
			}
		}
	}()

	return &HostExecStream{
		Stdout: stdoutCh,
		Stderr: stderrCh,
		Wait: func() (int, error) {
			<-doneCh
			return exitCode, exitErr
		},
		Cancel: cancel,
	}, nil
}

// Workspace returns a filesystem interface for the given session on the host.
// The sessionID typically corresponds to a workspace ID; files are stored
// under dataDir/{sessionID}/ on the Tai host, accessed via Volume gRPC.
func (h *Host) Workspace(sessionID string) workspace.FS {
	client, err := h.manager.getPool(h.pool)
	if err != nil {
		return nil
	}
	return client.Workspace(sessionID)
}
