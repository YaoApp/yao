package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"

	hepb "github.com/yaoapp/yao/tai/hostexec/pb"
	taiworkspace "github.com/yaoapp/yao/tai/workspace"
)

// Host represents a Tai host machine execution environment.
// Unlike Box (which wraps a container), Host executes commands directly on
// the Tai server's OS via HostExec gRPC and accesses files via Volume gRPC.
//
// Host implements the Computer interface.
type Host struct {
	nodeID      string
	workplaceID string
	system      SystemInfo
	manager     *Manager
}

// Compile-time check: *Host implements Computer.
var _ Computer = (*Host)(nil)

// ComputerInfo returns identity and registry information for the host.
// Registry-level details (TaiID, System, etc.) are populated when the node
// is backed by a registered Tai node; otherwise only Kind and NodeID are set.
func (h *Host) ComputerInfo() ComputerInfo {
	return ComputerInfo{
		Kind:   "host",
		NodeID: h.nodeID,
		System: h.system,
		Status: "online",
	}
}

// Exec runs a command on the Tai host machine via HostExec gRPC.
// cmd[0] is the program, cmd[1:] are arguments.
func (h *Host) Exec(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecResult, error) {
	if len(cmd) == 0 {
		return nil, fmt.Errorf("sandbox: empty command")
	}

	res, err := h.manager.getNode(h.nodeID)
	if err != nil {
		return nil, err
	}

	he := res.HostExec
	if he == nil {
		return nil, fmt.Errorf("sandbox: host_exec not available on node %q", h.nodeID)
	}

	cfg := &execConfig{}
	for _, o := range opts {
		o(cfg)
	}

	req := &hepb.ExecRequest{
		Command: cmd[0],
		Args:    cmd[1:],
		Stdin:   cfg.Stdin,
	}
	if cfg.WorkDir != "" {
		req.WorkingDir = cfg.WorkDir
	}
	if cfg.Env != nil {
		req.Env = cfg.Env
	}
	if cfg.Timeout > 0 {
		req.TimeoutMs = cfg.Timeout.Milliseconds()
	}
	if cfg.MaxOutputBytes > 0 {
		req.MaxOutputBytes = cfg.MaxOutputBytes
	}

	resp, err := he.Exec(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("hostexec rpc: %w", err)
	}

	return &ExecResult{
		ExitCode:   int(resp.ExitCode),
		Stdout:     string(resp.Stdout),
		Stderr:     string(resp.Stderr),
		DurationMs: resp.DurationMs,
		Error:      resp.Error,
		Truncated:  resp.Truncated,
	}, nil
}

// Stream runs a command on the Tai host and streams stdout/stderr in real time
// via HostExec gRPC ExecStream. Returns a unified ExecStream with io.ReadCloser
// for stdout/stderr.
func (h *Host) Stream(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecStream, error) {
	if len(cmd) == 0 {
		return nil, fmt.Errorf("sandbox: empty command")
	}

	res, err := h.manager.getNode(h.nodeID)
	if err != nil {
		return nil, err
	}

	he := res.HostExec
	if he == nil {
		return nil, fmt.Errorf("sandbox: host_exec not available on node %q", h.nodeID)
	}

	cfg := &execConfig{}
	for _, o := range opts {
		o(cfg)
	}

	req := &hepb.ExecRequest{
		Command: cmd[0],
		Args:    cmd[1:],
		Stdin:   cfg.Stdin,
	}
	if cfg.WorkDir != "" {
		req.WorkingDir = cfg.WorkDir
	}
	if cfg.Env != nil {
		req.Env = cfg.Env
	}
	if cfg.Timeout > 0 {
		req.TimeoutMs = cfg.Timeout.Milliseconds()
	}
	if cfg.MaxOutputBytes > 0 {
		req.MaxOutputBytes = cfg.MaxOutputBytes
	}

	streamCtx, cancel := context.WithCancel(ctx)
	rpcStream, err := he.ExecStream(streamCtx, req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("hostexec stream rpc: %w", err)
	}

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	doneCh := make(chan struct{})
	var exitCode int
	var exitErr error

	go func() {
		defer stdoutW.Close()
		defer stderrW.Close()
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
					stdoutW.Write(msg.Data)
				case hepb.ExecOutput_STDERR:
					stderrW.Write(msg.Data)
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

	return &ExecStream{
		Stdout: stdoutR,
		Stderr: stderrR,
		Stdin:  nopWriteCloser{&bytes.Buffer{}},
		Wait: func() (int, error) {
			<-doneCh
			return exitCode, exitErr
		},
		Cancel: cancel,
	}, nil
}

// VNC returns the VNC WebSocket URL for the Tai host machine.
// Uses the special __host__ identifier to route to localhost:5900 on the Tai server.
func (h *Host) VNC(ctx context.Context) (string, error) {
	res, err := h.manager.getNode(h.nodeID)
	if err != nil {
		return "", err
	}
	if res.VNC == nil {
		return "", fmt.Errorf("sandbox: vnc not available on node %q", h.nodeID)
	}
	return res.VNC.URL(ctx, "__host__")
}

// Proxy returns the HTTP URL for a service running on the Tai host machine.
// Uses the special __host__ identifier to route to localhost:{port} on the Tai server.
func (h *Host) Proxy(ctx context.Context, port int, path string) (string, error) {
	res, err := h.manager.getNode(h.nodeID)
	if err != nil {
		return "", err
	}
	if res.Proxy == nil {
		return "", fmt.Errorf("sandbox: proxy not available on node %q", h.nodeID)
	}
	return res.Proxy.URL(ctx, "__host__", port, path)
}

// BindWorkplace binds a workspace to this host by ID. Subsequent calls to
// Workplace() will return the FS for this workspace. Call again to rebind.
func (h *Host) BindWorkplace(workspaceID string) {
	h.workplaceID = workspaceID
}

// Workplace returns the workspace FS bound to this host, or nil if unbound.
func (h *Host) Workplace() taiworkspace.FS {
	if h.workplaceID == "" {
		return nil
	}
	res, err := h.manager.getNode(h.nodeID)
	if err != nil {
		return nil
	}
	if res.Volume == nil {
		return nil
	}
	return taiworkspace.New(res.Volume, h.workplaceID)
}

// GetWorkDir returns the host working directory for command execution.
// Resolves from the bound workspace's root path on disk, falling back to
// the system temp directory if no workspace is bound or root resolution fails.
func (h *Host) GetWorkDir() string {
	if ws := h.Workplace(); ws != nil {
		if root, err := ws.GetRoot(); err == nil && root != "" {
			return root
		}
	}
	if h.system.TempDir != "" {
		return h.system.TempDir
	}
	return "/tmp"
}

// NodeID returns the node ID this Host belongs to.
func (h *Host) NodeID() string { return h.nodeID }

// nopWriteCloser wraps an io.Writer with a no-op Close.
type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
