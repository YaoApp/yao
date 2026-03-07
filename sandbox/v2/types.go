package sandbox

import (
	"io"
	"time"

	"github.com/yaoapp/yao/tai"
)

type LifecyclePolicy string

const (
	OneShot     LifecyclePolicy = "oneshot"
	Session     LifecyclePolicy = "session"
	LongRunning LifecyclePolicy = "longrunning"
	Persistent  LifecyclePolicy = "persistent"
)

const DefaultStopTimeout = 2 * time.Second

type Pool struct {
	Name        string
	Addr        string
	Options     []tai.Option
	MaxPerUser  int
	MaxTotal    int
	IdleTimeout time.Duration
	MaxLifetime time.Duration
	StopTimeout time.Duration // SIGTERM grace period before SIGKILL; 0 = DefaultStopTimeout
}

type PoolInfo struct {
	Name        string
	Addr        string
	Connected   bool
	Boxes       int
	MaxPerUser  int
	MaxTotal    int
	IdleTimeout time.Duration
	MaxLifetime time.Duration
}

type PortMapping struct {
	ContainerPort int
	HostPort      int
	HostIP        string
	Protocol      string
}

type CreateOptions struct {
	ID          string
	Owner       string
	Labels      map[string]string
	Pool        string
	Image       string
	WorkDir     string
	User        string
	Env         map[string]string
	Memory      int64
	CPUs        float64
	VNC         bool
	Ports       []PortMapping
	Policy      LifecyclePolicy
	IdleTimeout time.Duration

	StopTimeout time.Duration // SIGTERM grace period; 0 = pool default or DefaultStopTimeout

	WorkspaceID string // workspace to mount; empty = no workspace
	MountMode   string // "rw" (default) or "ro"
	MountPath   string // container path; default "/workspace"
}

type ListOptions struct {
	Owner  string
	Pool   string
	Labels map[string]string
}

type execConfig struct {
	WorkDir string
	Env     map[string]string
	Timeout time.Duration
}

type ExecOption func(*execConfig)

func WithWorkDir(dir string) ExecOption {
	return func(c *execConfig) {
		c.WorkDir = dir
	}
}

func WithEnv(env map[string]string) ExecOption {
	return func(c *execConfig) {
		c.Env = env
	}
}

func WithTimeout(timeout time.Duration) ExecOption {
	return func(c *execConfig) {
		c.Timeout = timeout
	}
}

type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type ExecStream struct {
	Stdout io.ReadCloser
	Stderr io.ReadCloser
	Stdin  io.WriteCloser
	Wait   func() (int, error)
	Cancel func()
}

type attachConfig struct {
	Protocol string
	Path     string
	Headers  map[string]string
}

type AttachOption func(*attachConfig)

func WithProtocol(protocol string) AttachOption {
	return func(c *attachConfig) {
		c.Protocol = protocol
	}
}

func WithPath(path string) AttachOption {
	return func(c *attachConfig) {
		c.Path = path
	}
}

func WithHeaders(headers map[string]string) AttachOption {
	return func(c *attachConfig) {
		c.Headers = headers
	}
}

// ImagePullOptions configures an image pull operation.
type ImagePullOptions struct {
	Auth *RegistryAuth // nil = anonymous / public
}

// RegistryAuth holds credentials for a private container registry.
type RegistryAuth struct {
	Username string
	Password string
	Server   string
}

type ServiceConn struct {
	Read   func() ([]byte, error)
	Write  func(data []byte) error
	Events <-chan []byte
	URL    string
	Close  func() error
}

type BoxInfo struct {
	ID           string
	ContainerID  string
	Pool         string
	Owner        string
	Status       string
	Policy       LifecyclePolicy
	Labels       map[string]string
	Image        string
	CreatedAt    time.Time
	LastActive   time.Time
	ProcessCount int
	VNC          bool
}

// HostExecResult holds the outcome of a command executed on the Tai host.
type HostExecResult struct {
	ExitCode   int
	Stdout     []byte
	Stderr     []byte
	DurationMs int64
	Error      string
	Truncated  bool
}

// HostExecStream provides real-time streaming output from a command running
// on the Tai host machine via HostExec gRPC ExecStream.
type HostExecStream struct {
	Stdout <-chan []byte
	Stderr <-chan []byte
	Wait   func() (int, error) // blocks until exit; returns exit code
	Cancel func()              // cancels the stream context
}

type hostExecConfig struct {
	WorkDir        string
	Env            map[string]string
	Stdin          []byte
	TimeoutMs      int64
	MaxOutputBytes int64
}

// HostExecOption configures an ExecOnHost call.
type HostExecOption func(*hostExecConfig)

func WithHostWorkDir(dir string) HostExecOption {
	return func(c *hostExecConfig) { c.WorkDir = dir }
}

func WithHostEnv(env map[string]string) HostExecOption {
	return func(c *hostExecConfig) { c.Env = env }
}

func WithHostStdin(data []byte) HostExecOption {
	return func(c *hostExecConfig) { c.Stdin = data }
}

func WithHostTimeout(ms int64) HostExecOption {
	return func(c *hostExecConfig) { c.TimeoutMs = ms }
}

func WithHostMaxOutput(bytes int64) HostExecOption {
	return func(c *hostExecConfig) { c.MaxOutputBytes = bytes }
}
