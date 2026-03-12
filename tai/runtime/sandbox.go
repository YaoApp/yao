package runtime

import (
	"context"
	"io"
	"time"
)

// Runtime manages container lifecycle.
// Local connects directly to a Docker daemon; Docker/Containerd/K8s connect via Tai proxy.
type Runtime interface {
	Create(ctx context.Context, opts CreateOptions) (string, error)
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string, timeout time.Duration) error
	Remove(ctx context.Context, id string, force bool) error
	Exec(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error)
	ExecStream(ctx context.Context, id string, cmd []string, opts ExecOptions) (*StreamHandle, error)
	Inspect(ctx context.Context, id string) (*ContainerInfo, error)
	List(ctx context.Context, opts ListOptions) ([]ContainerInfo, error)
	Close() error
}

// StreamHandle provides real-time I/O access to a running exec process.
type StreamHandle struct {
	Stdin  io.WriteCloser
	Stdout io.Reader
	Stderr io.Reader
	// Wait blocks until the exec process finishes and returns the exit code.
	Wait func() (int, error)
	// Cancel aborts the exec process.
	Cancel func()
}

// CreateOptions configures a new container.
type CreateOptions struct {
	Name       string
	Image      string
	Cmd        []string
	Env        map[string]string
	Binds      []string
	WorkingDir string
	Memory     int64   // bytes, 0 = no limit
	CPUs       float64 // 0 = no limit
	VNC        bool
	Ports      []PortMapping
	Labels     map[string]string // container/pod labels for discovery and management
	User       string            // container user, e.g. "1000:1000" or "sandbox"
}

// PortMapping maps a container port to a host port.
type PortMapping struct {
	ContainerPort int
	HostPort      int    // 0 = random
	HostIP        string // default "127.0.0.1"
	Protocol      string // "tcp" (default) or "udp"
}

// ContainerInfo describes a running or stopped container.
type ContainerInfo struct {
	ID     string
	Name   string
	Image  string
	Status string // "created", "running", "exited", "removing"
	IP     string
	Ports  []PortMapping
	Labels map[string]string
}

// ExecOptions configures a command execution inside a container.
type ExecOptions struct {
	WorkDir string
	Env     map[string]string
}

// ExecResult holds output from an exec command.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// ListOptions filters container listing.
type ListOptions struct {
	All    bool              // include stopped containers
	Labels map[string]string // filter by labels
}

func envSlice(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	s := make([]string, 0, len(m))
	for k, v := range m {
		s = append(s, k+"="+v)
	}
	return s
}

func proto(p string) string {
	if p == "" {
		return "tcp"
	}
	return p
}

func hostIP(ip string) string {
	if ip == "" {
		return "127.0.0.1"
	}
	return ip
}
