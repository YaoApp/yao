package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
)

type dockerSandbox struct {
	core dockerCore
}

// NewDocker creates a Runtime backed by Docker SDK through Tai's Docker API proxy.
// addr should be "tcp://tai-host:12375".
func NewDocker(addr string) (Runtime, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(addr),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("docker via tai: %w", err)
	}
	return &dockerSandbox{core: dockerCore{cli: cli}}, nil
}

func (d *dockerSandbox) Create(ctx context.Context, opts CreateOptions) (string, error) {
	return d.core.create(ctx, opts, true)
}

func (d *dockerSandbox) Start(ctx context.Context, id string) error {
	return d.core.start(ctx, id)
}

func (d *dockerSandbox) Stop(ctx context.Context, id string, timeout time.Duration) error {
	return d.core.stop(ctx, id, int(timeout.Seconds()))
}

func (d *dockerSandbox) Remove(ctx context.Context, id string, force bool) error {
	return d.core.remove(ctx, id, force)
}

func (d *dockerSandbox) Exec(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
	return d.core.exec(ctx, id, cmd, opts)
}

func (d *dockerSandbox) ExecStream(ctx context.Context, id string, cmd []string, opts ExecOptions) (*StreamHandle, error) {
	return d.core.execStream(ctx, id, cmd, opts)
}

func (d *dockerSandbox) Inspect(ctx context.Context, id string) (*ContainerInfo, error) {
	return d.core.inspect(ctx, id)
}

func (d *dockerSandbox) List(ctx context.Context, opts ListOptions) ([]ContainerInfo, error) {
	return d.core.list(ctx, opts)
}

func (d *dockerSandbox) Close() error {
	return d.core.cli.Close()
}
