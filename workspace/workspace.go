package workspace

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MountMode controls read-write or read-only access when a workspace is
// bind-mounted into a container.
type MountMode string

const (
	MountRW MountMode = "rw"
	MountRO MountMode = "ro"
)

const metadataFile = ".workspace.json"

// Workspace is a persistent, user-managed storage entity.
// It is pinned to a specific Tai node (host machine) at creation time;
// containers referencing this workspace are automatically routed to that node.
type Workspace struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Owner     string            `json:"owner"`
	Node      string            `json:"node"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// CreateOptions configures a new workspace.
type CreateOptions struct {
	ID     string            // explicit ID; empty = auto-generate (uuid)
	Name   string            // human-readable name
	Owner  string            // user ID
	Node   string            // target Tai node (required)
	Labels map[string]string // arbitrary metadata
}

// ListOptions filters workspace listing.
type ListOptions struct {
	Owner string // filter by owner; empty = all
	Node  string // filter by node; empty = all
}

// UpdateOptions specifies which metadata fields to change.
// nil fields are left unchanged. Node and Owner are immutable.
type UpdateOptions struct {
	Name   *string           // nil = no change
	Labels map[string]string // nil = no change; non-nil replaces all labels
}

// NodeInfo describes a Tai node available for workspace storage.
type NodeInfo struct {
	Name   string // pool name = node name
	Addr   string // tai:// address
	Online bool   // tai client is connected
}

func generateID() string {
	return fmt.Sprintf("ws-%s", uuid.New().String()[:12])
}

func marshalMeta(ws *Workspace) ([]byte, error) {
	return json.MarshalIndent(ws, "", "  ")
}

func unmarshalMeta(data []byte) (*Workspace, error) {
	var ws Workspace
	if err := json.Unmarshal(data, &ws); err != nil {
		return nil, fmt.Errorf("workspace: invalid metadata: %w", err)
	}
	return &ws, nil
}
