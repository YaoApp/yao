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

type Pool struct {
	Name        string
	Addr        string
	Options     []tai.Option
	MaxPerUser  int
	MaxTotal    int
	IdleTimeout time.Duration
	MaxLifetime time.Duration
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
