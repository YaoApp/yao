package sandboxv2

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mathrand "math/rand"
	"strings"

	"github.com/yaoapp/kun/log"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/workspace"
)

// BuildIdentifier determines the Computer identifier based on lifecycle policy
// and optional metadata override. Returns "" for oneshot (always new).
func BuildIdentifier(cfg *types.SandboxConfig, ownerID, chatID, assistantID, workspaceID string, metadata map[string]any) string {
	if cfg.Lifecycle == "oneshot" {
		return ""
	}

	switch cfg.Lifecycle {
	case "session":
		return fmt.Sprintf("%s-%s-%s", ownerID, assistantID, chatID)
	case "longrunning", "persistent":
		return fmt.Sprintf("%s-%s.%s", ownerID, assistantID, workspaceID)
	default:
		return ""
	}
}

// ResolveNodeID determines the target nodeID and computer kind based on
// metadata and DSL configuration, without creating or acquiring a container.
// Returns (nodeID, kind, error). kind is "box" or "host".
func ResolveNodeID(ctx *agentContext.Context, cfg *types.SandboxConfig, manager *infra.Manager) (string, string, error) {
	computerID := ""
	if ctx.Metadata != nil {
		if cid, ok := ctx.Metadata["computer_id"].(string); ok && cid != "" {
			computerID = cid
		}
	}

	workspaceID := ""
	if ctx.Metadata != nil {
		if ws, ok := ctx.Metadata["workspace_id"].(string); ok && ws != "" {
			workspaceID = ws
		}
	}
	ownerID := resolveOwnerID(ctx)

	log.Trace("[sandbox/v2] ResolveNodeID: computerID=%q workspaceID=%q ownerID=%q image=%q", computerID, workspaceID, ownerID, cfg.Computer.Image)

	if workspaceID != "" {
		wsNode, err := workspace.M().NodeForWorkspace(context.Background(), workspaceID)
		if err == nil && wsNode != "" {
			log.Trace("[sandbox/v2] ResolveNodeID: workspace %s -> node %s", workspaceID, wsNode)
			computerID = wsNode
		}
	}

	if computerID == "" {
		pickedID, err := pickNodeByFilter(cfg.Filter, cfg.Computer.Image)
		if err != nil {
			return "", "", fmt.Errorf("auto-select node for ResolveNodeID: %w", err)
		}
		log.Trace("[sandbox/v2] ResolveNodeID: pickNodeByFilter -> %s", pickedID)
		computerID = pickedID
		cfg.NodeID = pickedID
	}

	if node, ok := tai.GetNodeMeta(computerID); ok {
		hasContainerRuntime := node.Capabilities.Docker || node.Capabilities.K8s
		log.Trace("[sandbox/v2] ResolveNodeID: node=%q HostExec=%v Docker=%v K8s=%v hasContainer=%v", computerID, node.Capabilities.HostExec, node.Capabilities.Docker, node.Capabilities.K8s, hasContainerRuntime)
		if node.Capabilities.HostExec && !hasContainerRuntime {
			log.Trace("[sandbox/v2] ResolveNodeID: -> host (host-only node)")
			return computerID, "host", nil
		}
		if node.Capabilities.HostExec && hasContainerRuntime && cfg.Computer.Image == "" {
			log.Trace("[sandbox/v2] ResolveNodeID: -> host (dual-capable, no image)")
			return computerID, "host", nil
		}
		if !hasContainerRuntime {
			return "", "", fmt.Errorf("node %q has no container runtime and no host_exec capability", computerID)
		}
		log.Trace("[sandbox/v2] ResolveNodeID: -> box")
		return computerID, "box", nil
	}
	log.Trace("[sandbox/v2] ResolveNodeID: node %q not found in registry, assuming box", computerID)
	return computerID, "box", nil
}

// GetComputer obtains or creates a Computer for the current request.
// Connector config is injected per-execution inside ClaudeRunner.Stream
// via "tai a2o config put".
// Returns the Computer, the resolved identifier, and any error.
func GetComputer(ctx *agentContext.Context, cfg *types.SandboxConfig, manager *infra.Manager) (infra.Computer, string, error) {
	ownerID := resolveOwnerID(ctx)

	workspaceID := ""
	if ctx.Metadata != nil {
		if ws, ok := ctx.Metadata["workspace_id"].(string); ok && ws != "" {
			workspaceID = ws
		}
	}

	identifier := BuildIdentifier(cfg, ownerID, ctx.ChatID, ctx.AssistantID, workspaceID, ctx.Metadata)

	// Fill runtime fields.
	cfg.Owner = ownerID
	cfg.ID = identifier
	cfg.WorkspaceID = workspaceID

	// Resolve computer_id from metadata to determine kind and nodeID.
	computerID := ""
	if ctx.Metadata != nil {
		if cid, ok := ctx.Metadata["computer_id"].(string); ok && cid != "" {
			computerID = cid
		}
	}

	// Workspace-wins rule: when workspace_id is present,
	// the workspace's bound node takes precedence over computer_id.
	if workspaceID != "" {
		wsNode, err := workspace.M().NodeForWorkspace(context.Background(), workspaceID)
		if err == nil && wsNode != "" {
			if computerID != "" && computerID != wsNode {
				log.Trace("[sandbox/v2] workspace %s bound to node %s overrides computer_id %s", workspaceID, wsNode, computerID)
			}
			computerID = wsNode
		}
	}

	log.Trace("[sandbox/v2] GetComputer: computerID=%q workspaceID=%q ownerID=%q cfgNodeID=%q image=%q", computerID, workspaceID, ownerID, cfg.NodeID, cfg.Computer.Image)

	if computerID != "" {
		log.Trace("[sandbox/v2] GetComputer: -> resolveComputerByID(%s)", computerID)
		return resolveComputerByID(cfg, manager, computerID, ownerID, identifier, workspaceID)
	}

	log.Trace("[sandbox/v2] GetComputer: -> resolveComputerByDSL (no computerID)")
	return resolveComputerByDSL(cfg, manager, ownerID, identifier, workspaceID)
}

// resolveComputerByID dispatches based on the runtime computer_id from metadata.
// It queries the registry and sandbox manager to determine the computer kind.
func resolveComputerByID(
	cfg *types.SandboxConfig, manager *infra.Manager,
	computerID, ownerID, identifier, workspaceID string,
) (infra.Computer, string, error) {

	// 1) Check if computer_id is a known Tai node (host or node kind).
	if node, ok := tai.GetNodeMeta(computerID); ok {
		cfg.NodeID = computerID
		hasContainerRuntime := node.Capabilities.Docker || node.Capabilities.K8s
		log.Trace("[sandbox/v2] resolveComputerByID: node=%q found=true HostExec=%v Docker=%v K8s=%v hasContainer=%v image=%q", computerID, node.Capabilities.HostExec, node.Capabilities.Docker, node.Capabilities.K8s, hasContainerRuntime, cfg.Computer.Image)

		if node.Capabilities.HostExec && !hasContainerRuntime {
			log.Trace("[sandbox/v2] resolveComputerByID: -> host (host-only node)")
			cfg.Kind = "host"
			host, err := manager.Host(context.Background(), computerID)
			if err != nil {
				return nil, identifier, fmt.Errorf("get host computer: %w", err)
			}
			host.BindWorkplace(workspaceID)
			return host, identifier, nil
		}

		if node.Capabilities.HostExec && hasContainerRuntime && cfg.Computer.Image == "" {
			// Dual-capable node with no image in DSL: prefer host mode.
			cfg.Kind = "host"
			host, err := manager.Host(context.Background(), computerID)
			if err != nil {
				return nil, identifier, fmt.Errorf("get host computer: %w", err)
			}
			host.BindWorkplace(workspaceID)
			return host, identifier, nil
		}

		if !hasContainerRuntime {
			return nil, identifier, fmt.Errorf("node %q has no container runtime and no host_exec capability", computerID)
		}

		// Node with container runtime and DSL has image: create/reuse a box.
		cfg.Kind = "box"
		return resolveBox(cfg, manager, ownerID, identifier, workspaceID)
	}

	// 2) Check if computer_id is an existing box ID.
	if manager != nil {
		box, err := manager.Get(context.Background(), computerID)
		if err == nil && box != nil {
			cfg.Kind = "box"
			box.BindWorkplace(workspaceID)
			return box, computerID, nil
		}
	}

	return nil, identifier, fmt.Errorf("computer %q not found in registry or sandbox manager", computerID)
}

// resolveComputerByDSL dispatches based on DSL static configuration (cfg.Computer.Image).
func resolveComputerByDSL(
	cfg *types.SandboxConfig, manager *infra.Manager,
	ownerID, identifier, workspaceID string,
) (infra.Computer, string, error) {

	log.Trace("[sandbox/v2] resolveComputerByDSL: cfgNodeID=%q image=%q", cfg.NodeID, cfg.Computer.Image)

	if cfg.NodeID == "" {
		pickedID, err := pickNodeByFilter(cfg.Filter, cfg.Computer.Image)
		if err != nil {
			return nil, identifier, fmt.Errorf("auto-select node: %w", err)
		}
		log.Trace("[sandbox/v2] resolveComputerByDSL: pickNodeByFilter -> %s", pickedID)
		cfg.NodeID = pickedID
	}

	log.Trace("[sandbox/v2] resolveComputerByDSL: -> resolveComputerByID(%s)", cfg.NodeID)
	return resolveComputerByID(cfg, manager, cfg.NodeID, ownerID, identifier, workspaceID)
}

// resolveBox reuses or creates a box container.
func resolveBox(
	cfg *types.SandboxConfig, manager *infra.Manager,
	ownerID, identifier, workspaceID string,
) (infra.Computer, string, error) {

	if workspaceID == "" && cfg.NodeID != "" {
		workspaceID = workspace.DefaultWorkspaceID(ownerID, cfg.NodeID)
		cfg.WorkspaceID = workspaceID
		if dot := strings.LastIndex(identifier, "."); dot >= 0 {
			identifier = identifier[:dot+1] + workspaceID
			cfg.ID = identifier
		}
	}

	// Reuse: non-empty identifier → try Get first.
	if identifier != "" {
		box, err := manager.Get(context.Background(), identifier)
		if err == nil && box != nil {
			if box.IsStopped() {
				if startErr := manager.StartBox(context.Background(), identifier); startErr != nil {
					log.Trace("[sandbox/v2] auto-start stopped box %s failed: %v, creating new", identifier, startErr)
				} else {
					box.BindWorkplace(workspaceID)
					return box, identifier, nil
				}
			} else {
				box.BindWorkplace(workspaceID)
				return box, identifier, nil
			}
		}
	}

	// Create new box.
	createOpts, err := BuildCreateOptions(cfg, identifier, ownerID, workspaceID)
	if err != nil {
		return nil, identifier, fmt.Errorf("build create options: %w", err)
	}
	log.Trace("[sandbox/v2] resolveBox: createOpts NodeID=%q Image=%q WorkspaceID=%q ID=%q Owner=%q", createOpts.NodeID, createOpts.Image, createOpts.WorkspaceID, createOpts.ID, createOpts.Owner)

	// Oneshot with empty identifier: generate a random one.
	if createOpts.ID == "" {
		createOpts.ID = randomID()
		identifier = createOpts.ID
		cfg.ID = identifier
	}

	box, err := manager.Create(context.Background(), createOpts)
	if err != nil {
		return nil, identifier, fmt.Errorf("create computer: %w", err)
	}
	return box, identifier, nil
}

// LifecycleAction performs the post-request lifecycle operation based on policy.
// Called in defer after executeSandboxStream completes.
func LifecycleAction(ctx context.Context, cfg *types.SandboxConfig, computer infra.Computer, manager *infra.Manager) {
	if computer == nil || cfg == nil {
		return
	}

	info := computer.ComputerInfo()

	switch cfg.Lifecycle {
	case "oneshot":
		if info.Kind == "box" && manager != nil {
			if err := manager.Remove(ctx, cfg.ID); err != nil {
				log.Trace("[sandbox/v2] oneshot remove %s: %v", cfg.ID, err)
			}
		}

	case "session", "longrunning":
		if info.Kind == "box" && manager != nil {
			manager.Heartbeat(cfg.ID, false, 0) // active=false: request finished, start idle timer
		}

	case "persistent":
		// No action — persistent boxes survive indefinitely.
	}
}

// resolveOwnerID returns teamID if available, otherwise userID.
func resolveOwnerID(ctx *agentContext.Context) string {
	if ctx.Authorized != nil {
		if ctx.Authorized.TeamID != "" {
			return ctx.Authorized.TeamID
		}
		if ctx.Authorized.UserID != "" {
			return ctx.Authorized.UserID
		}
	}
	return "anonymous"
}

// pickNodeByFilter selects a random online node that satisfies the given filter
// and image requirement. If image is non-empty, candidate nodes must have a
// container runtime (Docker or K8s).
func pickNodeByFilter(filter *types.ComputerFilter, image string) (string, error) {
	reg := registry.Global()
	if reg == nil {
		return "", fmt.Errorf("tai registry not initialized")
	}

	nodes := reg.List()
	var candidates []string
	for _, n := range nodes {
		if n.Status != "online" && n.Status != "" {
			continue
		}

		if filter != nil {
			if filter.OS != "" && !strings.EqualFold(n.System.OS, filter.OS) {
				continue
			}
			if filter.Arch != "" && !strings.EqualFold(n.System.Arch, filter.Arch) {
				continue
			}
			if len(filter.Kind) > 0 {
				matched := false
				for _, k := range filter.Kind {
					switch strings.ToLower(k) {
					case "host":
						if n.Capabilities.HostExec {
							matched = true
						}
					case "box":
						if n.Capabilities.Docker || n.Capabilities.K8s {
							matched = true
						}
					}
				}
				if !matched {
					continue
				}
			}
		}

		if image != "" && !(n.Capabilities.Docker || n.Capabilities.K8s) {
			continue
		}

		candidates = append(candidates, n.TaiID)
	}

	if len(candidates) == 0 {
		kind := ""
		os := ""
		arch := ""
		if filter != nil {
			kind = fmt.Sprintf("%v", []string(filter.Kind))
			os = filter.OS
			arch = filter.Arch
		}
		return "", fmt.Errorf("no online node matches filter (kind=%s os=%s arch=%s image=%s)", kind, os, arch, image)
	}

	return candidates[mathrand.Intn(len(candidates))], nil
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
