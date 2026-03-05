package sandbox

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	"github.com/yaoapp/yao/tai/proxy"
	taisandbox "github.com/yaoapp/yao/tai/sandbox"
	"github.com/yaoapp/yao/tai/workspace"
)

// Box represents a single sandbox instance.
type Box struct {
	id            string
	containerID   string
	pool          string
	owner         string
	policy        LifecyclePolicy
	labels        map[string]string
	lastCall      atomic.Int64
	lastHeartbeat atomic.Int64
	processCount  atomic.Int32
	idleTimeoutD  time.Duration
	createdAt     time.Time
	refreshToken  string
	vnc           bool
	image         string
	ws            workspace.FS
	manager       *Manager
}

func (b *Box) ID() string          { return b.id }
func (b *Box) Owner() string       { return b.owner }
func (b *Box) ContainerID() string { return b.containerID }
func (b *Box) Pool() string        { return b.pool }

// Exec runs a command and waits for it to finish.
func (b *Box) Exec(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecResult, error) {
	b.touch()
	cfg := &execConfig{}
	for _, o := range opts {
		o(cfg)
	}

	client, err := b.manager.getPool(b.pool)
	if err != nil {
		return nil, err
	}

	result, err := client.Sandbox().Exec(ctx, b.containerID, cmd, taisandbox.ExecOptions{
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

	if b.policy == OneShot {
		b.manager.Remove(ctx, b.id)
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

	client, err := b.manager.getPool(b.pool)
	if err != nil {
		return nil, err
	}

	handle, err := client.Sandbox().ExecStream(ctx, b.containerID, cmd, taisandbox.ExecOptions{
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

	client, err := b.manager.getPool(b.pool)
	if err != nil {
		return nil, err
	}

	conn, err := client.Proxy().Connect(ctx, b.containerID, proxy.ConnectOptions{
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
func (b *Box) Workspace() workspace.FS {
	b.touch()
	if b.ws != nil {
		return b.ws
	}
	client, err := b.manager.getPool(b.pool)
	if err != nil {
		return nil
	}
	b.ws = client.Workspace(b.id)
	return b.ws
}

// VNC returns the VNC WebSocket URL.
func (b *Box) VNC(ctx context.Context) (string, error) {
	b.touch()
	client, err := b.manager.getPool(b.pool)
	if err != nil {
		return "", err
	}
	return client.VNC().URL(ctx, b.containerID)
}

// Proxy returns the HTTP URL for a service on the given port inside the sandbox.
func (b *Box) Proxy(ctx context.Context, port int, path string) (string, error) {
	b.touch()
	client, err := b.manager.getPool(b.pool)
	if err != nil {
		return "", err
	}
	return client.Proxy().URL(ctx, b.containerID, port, path)
}

// Start starts a stopped sandbox.
func (b *Box) Start(ctx context.Context) error {
	client, err := b.manager.getPool(b.pool)
	if err != nil {
		return err
	}
	return client.Sandbox().Start(ctx, b.containerID)
}

// Stop stops the sandbox without removing it.
func (b *Box) Stop(ctx context.Context) error {
	client, err := b.manager.getPool(b.pool)
	if err != nil {
		return err
	}
	return client.Sandbox().Stop(ctx, b.containerID, 10*time.Second)
}

// Remove stops and removes the sandbox.
func (b *Box) Remove(ctx context.Context) error {
	return b.manager.Remove(ctx, b.id)
}

// Info returns current sandbox status.
func (b *Box) Info(ctx context.Context) (*BoxInfo, error) {
	client, err := b.manager.getPool(b.pool)
	if err != nil {
		return nil, err
	}

	info, err := client.Sandbox().Inspect(ctx, b.containerID)
	if err != nil {
		return nil, err
	}

	return &BoxInfo{
		ID:           b.id,
		ContainerID:  b.containerID,
		Pool:         b.pool,
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

func (b *Box) idleTimeout() time.Duration {
	if b.idleTimeoutD > 0 {
		return b.idleTimeoutD
	}
	pd := b.manager.findPoolDef(b.pool)
	if pd != nil {
		return pd.IdleTimeout
	}
	return 0
}

func (b *Box) maxLifetime() time.Duration {
	pd := b.manager.findPoolDef(b.pool)
	if pd != nil {
		return pd.MaxLifetime
	}
	return 0
}
