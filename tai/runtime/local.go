package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
)

type local struct {
	core dockerCore
}

// NewLocal creates a Runtime backed by a direct Docker daemon connection.
// addr can be "unix:///var/run/docker.sock", "tcp://host:port", or "" for platform default.
func NewLocal(addr string) (Runtime, error) {
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if addr != "" {
		opts = append(opts, client.WithHost(addr))
	} else {
		opts = append(opts, client.FromEnv)
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("docker ping: %w", err)
	}
	return &local{core: dockerCore{cli: cli}}, nil
}

func (l *local) Create(ctx context.Context, opts CreateOptions) (string, error) {
	return l.core.create(ctx, opts, opts.VNC)
}

func (l *local) Start(ctx context.Context, id string) error {
	return l.core.start(ctx, id)
}

func (l *local) Stop(ctx context.Context, id string, timeout time.Duration) error {
	return l.core.stop(ctx, id, int(timeout.Seconds()))
}

func (l *local) Remove(ctx context.Context, id string, force bool) error {
	return l.core.remove(ctx, id, force)
}

func (l *local) Exec(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
	return l.core.exec(ctx, id, cmd, opts)
}

func (l *local) ExecStream(ctx context.Context, id string, cmd []string, opts ExecOptions) (*StreamHandle, error) {
	return l.core.execStream(ctx, id, cmd, opts)
}

func (l *local) Inspect(ctx context.Context, id string) (*ContainerInfo, error) {
	return l.core.inspect(ctx, id)
}

func (l *local) List(ctx context.Context, opts ListOptions) ([]ContainerInfo, error) {
	return l.core.list(ctx, opts)
}

func (l *local) Close() error {
	return l.core.cli.Close()
}

func portStr(p int) string {
	if p == 0 {
		return ""
	}
	return fmt.Sprintf("%d", p)
}
