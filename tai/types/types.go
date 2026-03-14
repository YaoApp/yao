package types

import "time"

// Runtime selects which container runtime to use via Tai.
type Runtime int

const (
	Docker Runtime = iota
	K8s
)

// Ports configures service ports for Tai server.
type Ports struct {
	GRPC   int `json:"grpc"`
	HTTP   int `json:"http"`
	VNC    int `json:"vnc"`
	Docker int `json:"docker"`
	K8s    int `json:"k8s"`
}

// Capabilities describes what features a Tai node supports.
type Capabilities struct {
	Docker   bool `json:"docker"`
	K8s      bool `json:"k8s"`
	HostExec bool `json:"host_exec"`
	VNC      bool `json:"vnc"`
}

// SystemInfo describes the host machine running Tai.
type SystemInfo struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
	NumCPU   int    `json:"num_cpu"`
	TotalMem int64  `json:"total_mem,omitempty"`
	Shell    string `json:"shell,omitempty"`
	TempDir  string `json:"temp_dir,omitempty"`
}

// AuthInfo holds Yao user authorization extracted from OAuth token.
type AuthInfo struct {
	Subject  string
	UserID   string
	ClientID string
	Scope    string
	TeamID   string
	TenantID string
}

// NodeMeta is the read-only metadata snapshot of a registered Tai node.
// Carries no runtime resource references.
type NodeMeta struct {
	TaiID        string
	MachineID    string
	Version      string
	Auth         AuthInfo
	System       SystemInfo
	Mode         string // "direct" | "tunnel" | "local"
	Addr         string
	YaoBase      string
	Ports        Ports
	Capabilities Capabilities
	Status       string // "online" | "offline" | "connecting"
	ConnectedAt  time.Time
	LastPing     time.Time
	DisplayName  string
}
