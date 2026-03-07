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
