// Package types contains the public Computer module data contracts.
package types

import "time"

const (
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
	StatusError        = "error"
	StatusReleased     = "released"
)

// Computer is one Tai node registration.
type Computer struct {
	ID           string                 `json:"id"`
	NodeID       string                 `json:"node_id"`
	Name         string                 `json:"name"`
	ProviderType string                 `json:"provider_type"`
	Scope        string                 `json:"scope"`
	Visibility   string                 `json:"visibility"`
	Status       string                 `json:"status"`
	UserID       string                 `json:"user_id,omitempty"`
	TeamID       string                 `json:"team_id,omitempty"`
	DataDir      string                 `json:"data_dir,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at,omitempty"`
	UpdatedAt    time.Time              `json:"updated_at,omitempty"`
}

// CreateComputerRequest is the internal creation payload.
type CreateComputerRequest struct {
	ID           string
	NodeID       string
	Name         string
	ProviderType string
	Scope        string
	Visibility   string
	Status       string
	UserID       string
	TeamID       string
	DataDir      string
	Metadata     map[string]interface{}
}

// ConnectRequest starts a local node and returns its token.
type ConnectRequest struct {
	Name       string   `json:"name"`
	DataDir    string   `json:"data_dir"`
	Visibility string   `json:"visibility"`
	Members    []string `json:"members,omitempty"`
	Scope      string   `json:"-"`
	UserID     string   `json:"-"`
	TeamID     string   `json:"-"`
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
