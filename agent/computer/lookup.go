package computer

import (
	"context"
	"errors"
	"fmt"

	"github.com/yaoapp/yao/agent/sandbox/v2/resolve"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai/registry"
	taiTypes "github.com/yaoapp/yao/tai/types"
	"github.com/yaoapp/yao/workspace"
)

// LookupOpts carries the parameters for Lookup.
//
// Image and Lifecycle should always be provided by callers (fetched from
// assistant config). This keeps the computer package free of assistant-package
// imports, avoiding an import cycle.
type LookupOpts struct {
	Auth        *oauthtypes.AuthorizedInfo
	AssistantID string
	WorkspaceID string
	ChatID      string
	Image       string // container image (determines box vs host mode)
	Lifecycle   string // "oneshot" | "session" | "longrunning" | "persistent"

	// --- Optional overrides ---
	NodeID        string // skip workspace→node lookup when pre-resolved
	BoxIdentifier string // skip BuildIdentifier recomputation (oneshot needs this)
}

// Lookup is the single entry point for resolving parameters into a Computer
// (Box or Host). Both the agent pipeline (GetComputer) and API handlers
// (resolveComputer) use this function.
//
// Return semantics (consistent with the original resolveComputer):
//   - (nil, nil)  — box mode, sandbox not running (not found / stopped / oneshot unresolvable)
//   - (comp, nil) — computer found
//   - (nil, err)  — real error (assistant missing, workspace invalid, node offline, etc.)
func Lookup(ctx context.Context, opts *LookupOpts) (sandbox.Computer, error) {
	image, lifecycle := opts.Image, opts.Lifecycle
	nodeID := opts.NodeID

	// 1. Workspace is required.
	if opts.WorkspaceID == "" {
		return nil, errors.New("computer.Lookup: workspace ID is empty")
	}

	// 2. Resolve node from workspace when not overridden.
	if nodeID == "" {
		var err error
		nodeID, err = workspace.M().NodeForWorkspace(ctx, opts.WorkspaceID)
		if err != nil {
			return nil, fmt.Errorf("computer.Lookup: cannot resolve node for workspace %q: %w", opts.WorkspaceID, err)
		}
		if nodeID == "" {
			return nil, fmt.Errorf("computer.Lookup: workspace %q has no bound node", opts.WorkspaceID)
		}
	}

	// 3. Registry → NodeCandidate → ResolveMode
	meta, ok := registry.Global().Get(nodeID)
	if !ok {
		return nil, fmt.Errorf("computer.Lookup: node %q offline or removed", nodeID)
	}
	candidate := toNodeCandidate(meta)
	mode := resolve.ResolveMode(&candidate, "", image)

	// 4. ownerID (TeamID > UserID > "anonymous")
	ownerID := deriveOwnerID(opts.Auth)

	// 5. Dispatch by mode + BindWorkplace
	mgr := sandbox.M()
	switch mode {
	case "host":
		host, err := mgr.Host(ctx, nodeID)
		if err != nil {
			return nil, fmt.Errorf("computer.Lookup: host %q: %w", nodeID, err)
		}
		host.BindWorkplace(opts.WorkspaceID)
		return host, nil

	case "box":
		identifier := opts.BoxIdentifier
		if identifier == "" {
			identifier = resolve.BuildIdentifier(lifecycle, ownerID, opts.ChatID, opts.AssistantID, opts.WorkspaceID)
		}
		if identifier == "" {
			return nil, nil // oneshot without override: cannot look up
		}
		box, err := mgr.Get(ctx, identifier)
		if err != nil {
			if errors.Is(err, sandbox.ErrNotFound) {
				return nil, nil
			}
			return nil, fmt.Errorf("computer.Lookup: box %q: %w", identifier, err)
		}
		if box.IsStopped() {
			return nil, nil
		}
		box.BindWorkplace(opts.WorkspaceID)
		return box, nil

	default:
		return nil, fmt.Errorf("computer.Lookup: no feasible mode for node %q (canBox=%v canHost=%v image=%q)",
			nodeID, candidate.CanBox, candidate.CanHost, image)
	}
}

// toNodeCandidate converts registry NodeMeta to resolve.NodeCandidate.
func toNodeCandidate(n *taiTypes.NodeMeta) resolve.NodeCandidate {
	return resolve.NodeCandidate{
		ID:      n.TaiID,
		IsLocal: n.TaiID == "local" && n.Mode == "local",
		Runners: n.Capabilities.Runners,
		CanBox:  n.Capabilities.Docker || n.Capabilities.K8s,
		CanHost: n.Capabilities.HostExec,
		OS:      n.System.OS,
		Arch:    n.System.Arch,
	}
}

// deriveOwnerID mirrors resolveOwnerID from lifecycle.go.
func deriveOwnerID(auth *oauthtypes.AuthorizedInfo) string {
	if auth != nil {
		if auth.TeamID != "" {
			return auth.TeamID
		}
		if auth.UserID != "" {
			return auth.UserID
		}
	}
	return "anonymous"
}
