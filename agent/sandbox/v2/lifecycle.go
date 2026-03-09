package sandboxv2

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// BuildIdentifier determines the Computer identifier based on lifecycle policy
// and optional metadata override. Returns "" for oneshot (always new).
func BuildIdentifier(cfg *types.SandboxConfig, ownerID, chatID, assistantID string, metadata map[string]any) string {
	if cfg.Lifecycle == "oneshot" {
		return ""
	}

	// Custom identifier from metadata takes precedence.
	if metadata != nil {
		if cid, ok := metadata["computer_id"].(string); ok && cid != "" {
			return fmt.Sprintf("%s-%s", ownerID, cid)
		}
	}

	switch cfg.Lifecycle {
	case "session":
		return fmt.Sprintf("%s-%s", ownerID, chatID)
	case "longrunning", "persistent":
		return fmt.Sprintf("%s-%s", ownerID, assistantID)
	default:
		return ""
	}
}

// GetComputer obtains or creates a Computer for the current request.
// Returns the Computer, the resolved identifier, and any error.
func GetComputer(ctx *agentContext.Context, cfg *types.SandboxConfig, manager *infra.Manager) (infra.Computer, string, error) {
	ownerID := resolveOwnerID(ctx)
	identifier := BuildIdentifier(cfg, ownerID, ctx.ChatID, ctx.AssistantID, ctx.Metadata)

	// Fill runtime fields.
	cfg.Owner = ownerID
	cfg.ID = identifier

	workspaceID := ""
	if ctx.Metadata != nil {
		if ws, ok := ctx.Metadata["workspace_id"].(string); ok && ws != "" {
			workspaceID = ws
		}
	}
	if workspaceID == "" {
		workspaceID = ownerID
	}
	cfg.WorkspaceID = workspaceID

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

	// Reuse: non-empty identifier → try Get first.
	if identifier != "" {
		box, err := manager.Get(context.Background(), identifier)
		if err == nil && box != nil {
			box.BindWorkplace(workspaceID)
			return box, identifier, nil
		}
	}

	// Create new box.
	createOpts, err := BuildCreateOptions(cfg, identifier, ownerID, workspaceID)
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
