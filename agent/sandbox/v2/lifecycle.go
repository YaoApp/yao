package sandboxv2

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/yaoapp/gou/connector"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai"
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

// GetComputer obtains or creates a Computer for the current request.
// An optional connector may be passed to inject OPENAI_PROXY_* env vars.
// Returns the Computer, the resolved identifier, and any error.
func GetComputer(ctx *agentContext.Context, cfg *types.SandboxConfig, manager *infra.Manager, conn ...connector.Connector) (infra.Computer, string, error) {
	ownerID := resolveOwnerID(ctx)

	workspaceID := ""
	if ctx.Metadata != nil {
		if ws, ok := ctx.Metadata["workspace_id"].(string); ok && ws != "" {
			workspaceID = ws
		}
	}
	if workspaceID == "" {
		workspaceID = ownerID
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

	// Workspace-wins rule: when both workspace_id and computer_id are present,
	// the workspace's bound node takes precedence over computer_id.
	if workspaceID != "" && workspaceID != ownerID {
		wsNode, err := workspace.M().NodeForWorkspace(context.Background(), workspaceID)
		if err == nil && wsNode != "" {
			if computerID != "" && computerID != wsNode {
				log.Printf("[sandbox/v2] workspace %s bound to node %s overrides computer_id %s", workspaceID, wsNode, computerID)
			}
			computerID = wsNode
		}
	}

	if computerID != "" {
		return resolveComputerByID(cfg, manager, computerID, ownerID, identifier, workspaceID, conn...)
	}

	// No computer_id: fall back to DSL-based dispatch (original logic).
	return resolveComputerByDSL(cfg, manager, ownerID, identifier, workspaceID, conn...)
}

// resolveComputerByID dispatches based on the runtime computer_id from metadata.
// It queries the registry and sandbox manager to determine the computer kind.
func resolveComputerByID(
	cfg *types.SandboxConfig, manager *infra.Manager,
	computerID, ownerID, identifier, workspaceID string,
	conn ...connector.Connector,
) (infra.Computer, string, error) {

	// 1) Check if computer_id is a known Tai node (host or node kind).
	if node, ok := tai.GetNodeMeta(computerID); ok {
		cfg.NodeID = computerID
		hasContainerRuntime := node.Capabilities.Docker || node.Capabilities.K8s

		if node.Capabilities.HostExec && !hasContainerRuntime {
			// Host-only node: must use host mode regardless of DSL image config.
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
		return resolveBox(cfg, manager, ownerID, identifier, workspaceID, conn...)
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
	conn ...connector.Connector,
) (infra.Computer, string, error) {

	// Host mode: no image → host computer.
	if cfg.Computer.Image == "" {
		cfg.Kind = "host"
		nodeID := cfg.NodeID
		if nodeID == "" {
			return nil, identifier, fmt.Errorf("host mode requires a nodeID (set in sandbox.yao or workspace)")
		}
		host, err := manager.Host(context.Background(), nodeID)
		if err != nil {
			return nil, identifier, fmt.Errorf("get host computer: %w", err)
		}
		host.BindWorkplace(workspaceID)
		return host, identifier, nil
	}

	cfg.Kind = "box"
	return resolveBox(cfg, manager, ownerID, identifier, workspaceID, conn...)
}

// resolveBox reuses or creates a box container.
func resolveBox(
	cfg *types.SandboxConfig, manager *infra.Manager,
	ownerID, identifier, workspaceID string,
	conn ...connector.Connector,
) (infra.Computer, string, error) {

	// Reuse: non-empty identifier → try Get first.
	if identifier != "" {
		box, err := manager.Get(context.Background(), identifier)
		if err == nil && box != nil {
			if box.IsStopped() {
				if startErr := manager.StartBox(context.Background(), identifier); startErr != nil {
					log.Printf("[sandbox/v2] auto-start stopped box %s failed: %v, creating new", identifier, startErr)
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
	var c connector.Connector
	if len(conn) > 0 {
		c = conn[0]
	}
	createOpts, err := BuildCreateOptions(cfg, identifier, ownerID, workspaceID, c)
	if err != nil {
		return nil, identifier, fmt.Errorf("build create options: %w", err)
	}

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
				log.Printf("[sandbox/v2] oneshot remove %s: %v", cfg.ID, err)
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

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
