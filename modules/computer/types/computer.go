// Package types contains the public Computer module data contracts.
package types

import "time"

const (
	LeaseSourceTao      = "tao"
	LeaseSourceAdmin    = "admin"
	LeaseSourceInternal = "internal"

	ScopeSystem = "system"
	ScopeTeam   = "team"
	ScopeUser   = "user"

	VisibilityPrivate = "private"
	VisibilityMembers = "members"
	VisibilityTeam    = "team"
	VisibilityGlobal  = "global"

	StatusPending      = "pending"
	StatusProvisioning = "provisioning"
	StatusOnline       = "online"
	StatusOffline      = "offline"
	StatusDisconnected = "disconnected"
	StatusBusy         = "busy"
	StatusServing      = "serving"
	StatusSuspended    = "suspended"
	StatusExpired      = "expired"
	StatusError        = "error"
	StatusReleased     = "released"
)

// Computer is one Tai node registration.
type Computer struct {
	ID             string                 `json:"id"`
	NodeID         string                 `json:"node_id"`
	InstanceID     string                 `json:"instance_id,omitempty"`
	Name           string                 `json:"name"`
	ProviderType   string                 `json:"provider_type"`
	Scope          string                 `json:"scope"`
	Visibility     string                 `json:"visibility"`
	Status         string                 `json:"status"`
	UserID         string                 `json:"user_id,omitempty"`
	TeamID         string                 `json:"team_id,omitempty"`
	DataDir        string                 `json:"data_dir,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	LeaseSource    string                 `json:"lease_source,omitempty"`
	LeaseExpiresAt *time.Time             `json:"lease_expires_at,omitempty"`
	RenewedAt      *time.Time             `json:"renewed_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at,omitempty"`
	UpdatedAt      time.Time              `json:"updated_at,omitempty"`
}

// CreateComputerRequest is the internal creation payload.
type CreateComputerRequest struct {
	ID             string
	NodeID         string
	InstanceID     string
	Name           string
	ProviderType   string
	Scope          string
	Visibility     string
	Status         string
	UserID         string
	TeamID         string
	DataDir        string
	Metadata       map[string]interface{}
	LeaseSource    string
	LeaseExpiresAt *time.Time
}

// ConnectRequest starts a local node and returns its token.
type ConnectRequest struct {
	Name           string     `json:"name"`
	DataDir        string     `json:"data_dir"`
	Visibility     string     `json:"visibility"`
	Members        []string   `json:"members,omitempty"`
	LeaseSource    string     `json:"lease_source,omitempty"`
	LeaseExpiresAt *time.Time `json:"lease_expires_at,omitempty"`
	Scope          string     `json:"-"`
	UserID         string     `json:"-"`
	TeamID         string     `json:"-"`
}

// ConnectResult is returned once by POST /computers/connect.
type ConnectResult struct {
	Computer      *Computer `json:"computer"`
	ComputerID    string    `json:"computer_id"`
	NodeID        string    `json:"node_id"`
	NodeToken     string    `json:"node_token"`
	ExpiresAt     time.Time `json:"expires_at"`
	Server        string    `json:"server,omitempty"`
	TransportPort int       `json:"transport_port,omitempty"`
	Protocols     []string  `json:"protocols,omitempty"`
}

// ListFilter filters computer lists by owner/visibility/status.
type ListFilter struct {
	Scope      string
	Visibility string
	Status     string
	UserID     string
	TeamID     string
	Page       int
	PageSize   int
}

// UpdateRequest updates mutable computer fields.
type UpdateRequest struct {
	Name       *string `json:"name,omitempty"`
	Visibility *string `json:"visibility,omitempty"`
}

// CreateOrUpdateComputerRequest upserts a Computer by computer_id or instance_id.
type CreateOrUpdateComputerRequest struct {
	ID             string
	InstanceID     string
	NodeID         string
	Name           string
	ProviderType   string
	Scope          string
	Visibility     string
	Status         string
	UserID         string
	TeamID         string
	DataDir        string
	Metadata       map[string]interface{}
	LeaseSource    string
	LeaseExpiresAt *time.Time
	IssueNodeToken bool
}

// UpsertResult is the result of CreateOrUpdate.
type UpsertResult struct {
	Computer  *Computer
	NodeToken string
	ExpiresAt time.Time
}

// RenewLeaseRequest extends or restores a computer lease.
type RenewLeaseRequest struct {
	ExpiresAt      *time.Time    `json:"expires_at,omitempty"`
	Duration       time.Duration `json:"-"`
	LeaseSource    string        `json:"lease_source,omitempty"`
	IssueNodeToken bool          `json:"issue_node_token,omitempty"`
}

// RenewLeaseResult is returned by Renew.
type RenewLeaseResult struct {
	Computer  *Computer
	NodeToken string
	ExpiresAt time.Time
}

// Runtime describes a runnable container runtime.
type Runtime struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Ready   bool   `json:"ready"`
}

// Container describes a container exposed by the node runtime.
type Container struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image,omitempty"`
	Status string `json:"status,omitempty"`
}

// WorkspaceRef references a workspace available on a computer.
type WorkspaceRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
