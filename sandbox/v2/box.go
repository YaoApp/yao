package sandbox

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	"github.com/yaoapp/yao/tai/proxy"
	tairuntime "github.com/yaoapp/yao/tai/runtime"
	taiworkspace "github.com/yaoapp/yao/tai/workspace"
)

// Box represents a single sandbox instance.
type Box struct {
	id            string
	containerID   string
	nodeID        string
	owner         string
	policy        LifecyclePolicy
	labels        map[string]string
	lastCall      atomic.Int64
	lastHeartbeat atomic.Int64
	processCount  atomic.Int32
	status        atomic.Value // string: "running", "exited", "stopped", "created", "unknown"
	idleTimeoutD  time.Duration
	maxLifetimeD  time.Duration
	stopTimeoutD  time.Duration
	createdAt     time.Time
	vnc           bool
	image         string
	workspaceID   string
	system        SystemInfo
	displayName   string
	workDir       string
	ws            taiworkspace.FS
	manager       *Manager
}

// Compile-time check: *Box implements Computer.
var _ Computer = (*Box)(nil)

func (b *Box) ID() string          { return b.id }
func (b *Box) Owner() string       { return b.owner }
func (b *Box) ContainerID() string { return b.containerID }
func (b *Box) NodeID() string      { return b.nodeID }

// ComputerInfo returns identity and registry information for this Box.
func (b *Box) ComputerInfo() ComputerInfo {
	return ComputerInfo{
		Kind:        "box",
		NodeID:      b.nodeID,
		System:      b.system,
		Status:      "online",
		BoxID:       b.id,
		ContainerID: b.containerID,
		Owner:       b.owner,
		Image:       b.image,
		Policy:      b.policy,
		Labels:      b.labels,
		DisplayName: b.displayName,
	}
}

// BindWorkplace binds (or rebinds) a workspace to this Box. Subsequent calls
// to Workplace() return the FS for this workspace. Overrides the workspace
// set during Create.
func (b *Box) BindWorkplace(workspaceID string) {
	b.workspaceID = workspaceID
	b.ws = nil // clear cache so Workplace() re-resolves
}

// Workplace returns the workspace FS bound to this Box.
// If a workspace was bound via CreateOptions.WorkspaceID or BindWorkplace(),
// returns that workspace's FS. Otherwise returns nil.
func (b *Box) Workplace() taiworkspace.FS {
	return b.Workspace()
}

// Exec runs a command and waits for it to finish.
func (b *Box) Exec(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecResult, error) {
	b.touch()
	cfg := &execConfig{}
	for _, o := range opts {
		o(cfg)
	}

	res, err := b.manager.getNode(b.nodeID)
	if err != nil {
		return nil, err
	}

	result, err := res.Runtime.Exec(ctx, b.containerID, cmd, tairuntime.ExecOptions{
		WorkDir: cfg.WorkDir,
		Env:     cfg.Env,
	})
	if err != nil {
		return nil, err
	}

	r := &ExecResult{
		ExitCode: result.ExitCode,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}

	return r, nil
}

// Stream runs a command with real-time streaming I/O.
func (b *Box) Stream(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecStream, error) {
	b.touch()
	cfg := &execConfig{}
	for _, o := range opts {
		o(cfg)
	}

	res, err := b.manager.getNode(b.nodeID)
	if err != nil {
		return nil, err
	}

	handle, err := res.Runtime.ExecStream(ctx, b.containerID, cmd, tairuntime.ExecOptions{
		WorkDir: cfg.WorkDir,
		Env:     cfg.Env,
	})
	if err != nil {
		return nil, err
	}

	return &ExecStream{
		Stdout: io.NopCloser(handle.Stdout),
		Stderr: io.NopCloser(handle.Stderr),
		Stdin:  handle.Stdin,
		Wait:   handle.Wait,
		Cancel: handle.Cancel,
	}, nil
}

// Attach connects to a service running inside the sandbox on the given container port.
func (b *Box) Attach(ctx context.Context, port int, opts ...AttachOption) (*ServiceConn, error) {
	b.touch()
	cfg := &attachConfig{Protocol: "ws"}
	for _, o := range opts {
		o(cfg)
	}

	res, err := b.manager.getNode(b.nodeID)
	if err != nil {
		return nil, err
	}

	conn, err := res.Proxy.Connect(ctx, b.containerID, proxy.ConnectOptions{
		Port:     port,
		Path:     cfg.Path,
		Protocol: cfg.Protocol,
	})
	if err != nil {
		return nil, err
	}

	sc := &ServiceConn{
		Write:  conn.Send,
		Events: conn.Messages,
		Close:  conn.Close,
	}

	if cfg.Protocol == "ws" {
		ch := conn.Messages
		sc.Read = func() ([]byte, error) {
			msg, ok := <-ch
			if !ok {
				return nil, io.EOF
			}
			return msg, nil
		}
	}

	return sc, nil
}

// Workspace returns an fs.FS-compatible filesystem for this sandbox.
// If a workspace is mounted (WorkspaceID set), uses the workspace ID as session;
// otherwise falls back to the sandbox ID (backward compatible).
func (b *Box) Workspace() taiworkspace.FS {
	b.touch()
	if b.ws != nil {
		return b.ws
	}
	sessionID := b.workspaceID
	if sessionID == "" {
		sessionID = b.id
	}
	res, err := b.manager.getNode(b.nodeID)
	if err != nil {
		return nil
	}
	b.ws = taiworkspace.New(res.Volume, sessionID)
	return b.ws
}

// GetWorkDir returns the container-internal working directory for command execution.
func (b *Box) GetWorkDir() string {
	if b.workDir != "" {
		return b.workDir
	}
	return "/workspace"
}

// WorkspaceID returns the workspace ID mounted to this sandbox, or empty string.
func (b *Box) WorkspaceID() string { return b.workspaceID }

// Snapshot returns a local-only BoxInfo snapshot without any remote calls.
// Status is maintained by the sandbox watcher (see watcher.go).
func (b *Box) Snapshot() BoxInfo {
	s, _ := b.status.Load().(string)
	if s == "" {
		s = "unknown"
	}
	return BoxInfo{
		ID:           b.id,
		ContainerID:  b.containerID,
		NodeID:       b.nodeID,
		Owner:        b.owner,
		Status:       s,
		Policy:       b.policy,
		Labels:       b.labels,
		Image:        b.image,
		CreatedAt:    b.createdAt,
		LastActive:   b.lastActiveTime(),
		ProcessCount: int(b.processCount.Load()),
		VNC:          b.vnc,
	}
}

// VNC returns the VNC WebSocket URL.
func (b *Box) VNC(ctx context.Context) (string, error) {
	b.touch()
	res, err := b.manager.getNode(b.nodeID)
	if err != nil {
		return "", err
	}
	return res.VNC.URL(ctx, b.containerID)
}

// Proxy returns the HTTP URL for a service on the given port inside the sandbox.
func (b *Box) Proxy(ctx context.Context, port int, path string) (string, error) {
	b.touch()
	res, err := b.manager.getNode(b.nodeID)
	if err != nil {
		return "", err
	}
	return res.Proxy.URL(ctx, b.containerID, port, path)
}

// Start starts a stopped sandbox.
func (b *Box) Start(ctx context.Context) error {
	res, err := b.manager.getNode(b.nodeID)
	if err != nil {
		return err
	}
	return res.Runtime.Start(ctx, b.containerID)
}

// Stop stops the sandbox without removing it.
func (b *Box) Stop(ctx context.Context) error {
	res, err := b.manager.getNode(b.nodeID)
	if err != nil {
		return err
	}
	return res.Runtime.Stop(ctx, b.containerID, b.stopTimeout())
}

// Remove stops and removes the sandbox.
func (b *Box) Remove(ctx context.Context) error {
	return b.manager.Remove(ctx, b.id)
}

// Info returns current sandbox status.
func (b *Box) Info(ctx context.Context) (*BoxInfo, error) {
	res, err := b.manager.getNode(b.nodeID)
	if err != nil {
		return nil, err
	}

	info, err := res.Runtime.Inspect(ctx, b.containerID)
	if err != nil {
		return nil, err
	}

	return &BoxInfo{
		ID:           b.id,
		ContainerID:  b.containerID,
		NodeID:       b.nodeID,
		Owner:        b.owner,
		Status:       info.Status,
		Policy:       b.policy,
		Labels:       b.labels,
		Image:        info.Image,
		CreatedAt:    b.createdAt,
		LastActive:   b.lastActiveTime(),
		ProcessCount: int(b.processCount.Load()),
		VNC:          b.vnc,
	}, nil
}

func (b *Box) touch() {
	b.lastCall.Store(time.Now().UnixMilli())
}

func (b *Box) lastActiveTime() time.Time {
	call := b.lastCall.Load()
	hb := b.lastHeartbeat.Load()
	ts := call
	if hb > ts {
		ts = hb
	}
	return time.UnixMilli(ts)
}

// idleSince returns the timestamp of the last business call (Exec/Stream/VNC/etc).
// Unlike lastActiveTime, heartbeats do NOT reset this — only real user activity does.
func (b *Box) idleSince() time.Time {
	return time.UnixMilli(b.lastCall.Load())
}

func (b *Box) idleTimeout() time.Duration {
	return b.idleTimeoutD
}

func (b *Box) maxLifetime() time.Duration {
	return b.maxLifetimeD
}

func (b *Box) stopTimeout() time.Duration {
	if b.stopTimeoutD > 0 {
		return b.stopTimeoutD
	}
	return DefaultStopTimeout
}

// IsStopped reports whether the box's last known status indicates a non-running container.
func (b *Box) IsStopped() bool {
	s, _ := b.status.Load().(string)
	return s == "exited" || s == "stopped"
}

// inspectStatus queries the container runtime for the real container state.
func (b *Box) inspectStatus(ctx context.Context) string {
	res, err := b.manager.getNode(b.nodeID)
	if err != nil || res.Runtime == nil {
		return "unknown"
	}
	info, err := res.Runtime.Inspect(ctx, b.containerID)
	if err != nil {
		return "unknown"
	}
	return info.Status
}
