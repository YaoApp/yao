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
