package sandbox

import (
	"context"
	"io"
	"time"

	"github.com/yaoapp/yao/tai/workspace"
)

// ---------------------------------------------------------------------------
// Computer — unified interface for execution environments
// ---------------------------------------------------------------------------

// Computer is the unified interface for remote execution environments.
// Both Box (container) and Host (bare metal) implement it.
type Computer interface {
	ComputerInfo() ComputerInfo
	Exec(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecResult, error)
	Stream(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecStream, error)
	VNC(ctx context.Context) (string, error)
	Proxy(ctx context.Context, port int, path string) (string, error)
	BindWorkplace(workspaceID string)
	Workplace() workspace.FS
	GetWorkDir() string
	ListPorts(ctx context.Context) ([]*PortInfo, error)
	ListProcesses(ctx context.Context, opts ...ListProcessesOption) ([]*ProcessInfo, *SystemLoad, error)
}

// ComputerInfo holds identity and registry information for a Computer.
type ComputerInfo struct {
	Kind         string // "box" | "host"
	NodeID       string
	TaiID        string
	MachineID    string
	Version      string
	System       SystemInfo
	Mode         string // "direct" | "tunnel"
	Capabilities map[string]bool
	Status       string

	// Box-specific fields (zero values for Host)
	BoxID       string
	ContainerID string
	Owner       string
	Image       string
	Policy      LifecyclePolicy
	Labels      map[string]string
	DisplayName string
}

// SystemInfo describes the hardware and environment of a Tai node.
type SystemInfo struct {
	OS       string
	Arch     string
	Hostname string
	NumCPU   int
	TotalMem int64
	Shell    string // preferred shell: "sh", "pwsh", "powershell", "cmd.exe"
	TempDir  string // system temp directory
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

type LifecyclePolicy string

const (
	OneShot     LifecyclePolicy = "oneshot"
	Session     LifecyclePolicy = "session"
	LongRunning LifecyclePolicy = "longrunning"
	Persistent  LifecyclePolicy = "persistent"
)

const (
	DefaultStopTimeout            = 2 * time.Second
	DefaultSessionIdleTimeout     = 30 * time.Minute
	DefaultLongRunningIdleTimeout = 2 * time.Hour
	DefaultOneShotMaxAge          = 8 * time.Hour
)

// ---------------------------------------------------------------------------
// Create / List options
// ---------------------------------------------------------------------------

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
	NodeID      string
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
	MaxLifetime time.Duration
	StopTimeout time.Duration

	WorkspaceID string
	ChatID      string
	MountMode   string
	MountPath   string
	DisplayName string
	Locale      string
}

type ListOptions struct {
	Owner  string
	NodeID string
	Labels map[string]string
}

// ---------------------------------------------------------------------------
// Unified ExecOption / ExecResult / ExecStream
// ---------------------------------------------------------------------------

type execConfig struct {
	WorkDir        string
	Env            map[string]string
	Timeout        time.Duration
	Stdin          []byte
	MaxOutputBytes int64
}

// ExecOption configures an Exec or Stream call on any Computer.
type ExecOption func(*execConfig)

func WithWorkDir(dir string) ExecOption {
	return func(c *execConfig) { c.WorkDir = dir }
}

func WithEnv(env map[string]string) ExecOption {
	return func(c *execConfig) { c.Env = env }
}

func WithTimeout(timeout time.Duration) ExecOption {
	return func(c *execConfig) { c.Timeout = timeout }
}

func WithStdin(data []byte) ExecOption {
	return func(c *execConfig) { c.Stdin = data }
}

func WithMaxOutput(bytes int64) ExecOption {
	return func(c *execConfig) { c.MaxOutputBytes = bytes }
}

// ExecResult holds the outcome of a command executed on any Computer.
type ExecResult struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	DurationMs int64
	Error      string
	Truncated  bool
}

// ExecStream provides real-time streaming I/O for a running command.
type ExecStream struct {
	Stdout io.ReadCloser
	Stderr io.ReadCloser
	Stdin  io.WriteCloser
	Wait   func() (int, error)
	Cancel func()
}

// ---------------------------------------------------------------------------
// Attach (Box-specific, not part of Computer interface)
// ---------------------------------------------------------------------------

type attachConfig struct {
	Protocol string
	Path     string
	Headers  map[string]string
}

type AttachOption func(*attachConfig)

func WithProtocol(protocol string) AttachOption {
	return func(c *attachConfig) { c.Protocol = protocol }
}

func WithPath(path string) AttachOption {
	return func(c *attachConfig) { c.Path = path }
}

func WithHeaders(headers map[string]string) AttachOption {
	return func(c *attachConfig) { c.Headers = headers }
}

// ImagePullOptions configures an image pull operation.
type ImagePullOptions struct {
	Auth *RegistryAuth
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

// BoxInfo is a snapshot of a Box's runtime state (used by Manager.List).
type BoxInfo struct {
	ID           string
	ContainerID  string
	NodeID       string
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

// ---------------------------------------------------------------------------
// SystemQuery types
// ---------------------------------------------------------------------------

// PortInfo represents a listening network port.
type PortInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Process  string `json:"process"`
	PID      int    `json:"pid"`
	State    string `json:"state"`
	Address  string `json:"address"`
	Command  string `json:"command"`
}

// ProcessInfo represents a running process.
type ProcessInfo struct {
	PID        int     `json:"pid"`
	PPID       int     `json:"ppid"`
	User       string  `json:"user"`
	Command    string  `json:"command"`
	State      string  `json:"state"`
	CPUPercent float32 `json:"cpuPercent"`
	MemPercent float32 `json:"memPercent"`
	RSSBytes   int64   `json:"rssBytes"`
	VSZBytes   int64   `json:"vszBytes"`
	StartTime  int64   `json:"startTime"`
	CPUTimeMs  int64   `json:"cpuTimeMs"`
	Threads    int     `json:"threads"`
	OpenFiles  int     `json:"openFiles"`
}

// SystemLoad represents overall system resource usage.
type SystemLoad struct {
	Load1        float32 `json:"load1"`
	Load5        float32 `json:"load5"`
	Load15       float32 `json:"load15"`
	MemTotal     int64   `json:"memTotal"`
	MemUsed      int64   `json:"memUsed"`
	MemAvailable int64   `json:"memAvailable"`
	SwapTotal    int64   `json:"swapTotal"`
	SwapUsed     int64   `json:"swapUsed"`
	CPUCount     int     `json:"cpuCount"`
	CPUUsage     float32 `json:"cpuUsage"`
	UptimeSec    int64   `json:"uptimeSec"`
}

// ListProcessesOption configures a ListProcesses call.
type ListProcessesOption func(*listProcessesConfig)

type listProcessesConfig struct {
	SkipCPU bool
}

// WithSkipCPU skips CPU sampling for faster process listing.
func WithSkipCPU() ListProcessesOption {
	return func(c *listProcessesConfig) { c.SkipCPU = true }
}

func applyListProcessesOpts(opts []ListProcessesOption) listProcessesConfig {
	var cfg listProcessesConfig
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}
