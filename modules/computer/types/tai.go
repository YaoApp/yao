package types

import "context"

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
type TaiController interface {
	Bootstrap(ctx context.Context, target SSHTarget, req BootstrapRequest) error
	Start(ctx context.Context, target SSHTarget) error
	Stop(ctx context.Context, target SSHTarget) error
	Status(ctx context.Context, target SSHTarget) (*TaiStatus, error)
	Logs(ctx context.Context, target SSHTarget, lines int) (string, error)
}
