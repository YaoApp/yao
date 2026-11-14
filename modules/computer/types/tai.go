package types

import (
	"context"
	"time"
)

// SSHTarget identifies a cloud host controlled over SSH.
type SSHTarget struct {
	Host       string
	Port       int
	User       string
	PrivateKey string
	KnownHosts string
}

// BootstrapRequest contains the node credentials written before Tai starts.
type BootstrapRequest struct {
	NodeID    string
	NodeToken string
	Server    string
	DataDir   string
}

// TaiStatus is the normalized Tai process status.
type TaiStatus struct {
	Running bool
	Version string
	PID     int
}

// TaiController controls Tai on a remote host.
// StartTaiRequest mirrors the Tai v2 `tai start` config.
type StartTaiRequest struct {
	NodeID          string
	DataDir         string
	LogLevel        string
	LogFile         string
	LogFormat       string
	ShutdownTimeout time.Duration
	ServerURL       string
	NodeToken       string
	SandboxImage    string
}

type TaiController interface {
	Bootstrap(ctx context.Context, target SSHTarget, req BootstrapRequest) error
	Start(ctx context.Context, target SSHTarget, req StartTaiRequest) error
	Stop(ctx context.Context, target SSHTarget) error
	Status(ctx context.Context, target SSHTarget) (*TaiStatus, error)
	Logs(ctx context.Context, target SSHTarget, lines int) (string, error)
}

// TaiNodeProvider exposes runtime data from registered Tai nodes.
type TaiNodeProvider interface {
	Node(nodeID string) (TaiNode, bool)
	ListWorkspaces(ctx context.Context, nodeID string) ([]TaiWorkspace, error)
}

// TaiNodeController is an optional capability for node control.
type TaiNodeController interface {
	GoAway(ctx context.Context, nodeID, reason string) error
}

// TaiNode is the neutral runtime snapshot used by Computer APIs.
type TaiNode struct {
	ID                string
	Version           string
	OS                string
	Arch              string
	Capabilities      []string
	ContainerRuntimes []Runtime
}

// TaiWorkspace is a workspace reference returned by Tai.
type TaiWorkspace struct {
	ID   string
	Name string
}
