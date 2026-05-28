package sandboxv2

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/yaoapp/kun/log"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
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

// ResolveRunner determines which runner to use.
// Priority: userPref > sandbox.yao runner.name > globalRunner > fallback "yaocode".
// The special value "use::default" is treated as "not set" at each level,
// allowing the next level in the chain to take effect.
func ResolveRunner(userPref string, runnerCfg *types.RunnerConfig, globalRunner string) (string, error) {
	var runner string

	// 1. User explicit preference (skip "use::default")
	if userPref != "" && userPref != "use::default" {
		runner = userPref
	}

	// 2. sandbox.yao runner.name (skip "use::default")
	if runner == "" && runnerCfg != nil && runnerCfg.Name != "" && runnerCfg.Name != "use::default" {
		runner = runnerCfg.Name
	}

	// 3. agent.yml global runner (skip "use::default")
	if runner == "" && globalRunner != "" && globalRunner != "use::default" {
		runner = globalRunner
	}

	// 4. Fallback
	if runner == "" {
		runner = "yaocode"
	}

	// Backward-compat alias mapping
	runner = canonicalRunner(runner)

	if !containsRunner(SupportedRunners, runner) {
		return "", fmt.Errorf("unknown runner %q (supported: %v)", runner, SupportedRunners)
	}

	// Validate against sandbox.yao supports list
	if runnerCfg != nil && len(runnerCfg.Supports) > 0 {
		if !containsRunner(runnerCfg.Supports, runner) {
			return "", fmt.Errorf("runner %q not supported by this assistant (supports: %v)", runner, runnerCfg.Supports)
		}
	}

	return runner, nil
}

// GetComputer obtains or creates a Computer for the current request using the
// pre-computed SelectionResult from SelectNode. It trusts sel.Mode to dispatch
// directly, avoiding re-derivation from node capabilities.
// Returns the Computer, the resolved identifier, and any error.
func GetComputer(ctx *agentContext.Context, cfg *types.SandboxConfig, manager *infra.Manager, sel *SelectionResult) (infra.Computer, string, error) {
	ownerID := resolveOwnerID(ctx)

	workspaceID := ""
	if ctx.Metadata != nil {
		if ws, ok := ctx.Metadata["workspace_id"].(string); ok && ws != "" {
			workspaceID = ws
		}
	}

	identifier := BuildIdentifier(cfg, ownerID, ctx.ChatID, ctx.AssistantID, workspaceID, ctx.Metadata)

	cfg.Owner = ownerID
	cfg.ID = identifier
	cfg.WorkspaceID = workspaceID
	cfg.NodeID = sel.NodeID

	// Inject user identity into sandbox environment for ownership tracking
	if ctx.Authorized != nil {
		if cfg.Environment == nil {
			cfg.Environment = make(map[string]string)
		}
		if ctx.Authorized.UserID != "" {
			cfg.Environment["YAO_USER_ID"] = ctx.Authorized.UserID
		}
		if ctx.Authorized.TeamID != "" {
			cfg.Environment["YAO_TEAM_ID"] = ctx.Authorized.TeamID
		}
	}

	log.Trace("[sandbox/v2] GetComputer: nodeID=%q runner=%q mode=%q workspaceID=%q ownerID=%q image=%q",
		sel.NodeID, sel.Runner, sel.Mode, workspaceID, ownerID, cfg.Computer.Image)

	switch sel.Mode {
	case "host":
		cfg.Kind = "host"
		workspaceID, identifier = ensureHostWorkspace(cfg, ownerID, workspaceID, identifier)

		host, err := manager.Host(context.Background(), sel.NodeID)
		if err != nil {
			return nil, identifier, fmt.Errorf("get host computer on node %q: %w", sel.NodeID, err)
		}
		host.BindWorkplace(workspaceID)
		return host, identifier, nil

	case "box":
		cfg.Kind = "box"
		return resolveBox(cfg, manager, ownerID, identifier, workspaceID)

	default:
		return nil, identifier, fmt.Errorf("unsupported mode %q for node %q", sel.Mode, sel.NodeID)
	}
}

// ensureHostWorkspace generates a default workspace ID when none is provided
// and ensures the workspace directory + metadata exist on disk (idempotent).
// Mirrors the default-workspace logic in resolveBox for parity.
func ensureHostWorkspace(cfg *types.SandboxConfig, ownerID, workspaceID, identifier string) (string, string) {
	if workspaceID == "" && cfg.NodeID != "" {
		workspaceID = workspace.DefaultWorkspaceID(ownerID, cfg.NodeID)
		cfg.WorkspaceID = workspaceID
		if dot := strings.LastIndex(identifier, "."); dot >= 0 {
			identifier = identifier[:dot+1] + workspaceID
			cfg.ID = identifier
		}
	}

	if workspaceID != "" {
		wsm := workspace.M()
		if _, err := wsm.Get(context.Background(), workspaceID); err != nil {
			if _, createErr := wsm.Create(context.Background(), workspace.CreateOptions{
				ID:    workspaceID,
				Owner: ownerID,
				Node:  cfg.NodeID,
				Name:  "Default",
			}); createErr != nil {
				log.Trace("[sandbox/v2] auto-create workspace %s failed: %v", workspaceID, createErr)
			}
		}
	}

	return workspaceID, identifier
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
//
// NOTE: cfg.ID must NOT be used here — it is a shared mutable field on the
// Assistant struct and is overwritten by concurrent requests. The authoritative
// box ID is computer.ComputerInfo().BoxID, which is set once when the Box is
// created and never changes.
func LifecycleAction(ctx context.Context, cfg *types.SandboxConfig, computer infra.Computer, manager *infra.Manager) {
	if computer == nil || cfg == nil {
		return
	}

	info := computer.ComputerInfo()
	boxID := info.BoxID // use the box's own immutable ID, not cfg.ID

	switch cfg.Lifecycle {
	case "oneshot":
		if info.Kind == "box" && manager != nil {
			if err := manager.Remove(ctx, boxID); err != nil {
				log.Trace("[sandbox/v2] oneshot remove %s: %v", boxID, err)
			}
		}

	case "session", "longrunning":
		if info.Kind == "box" && manager != nil {
			manager.Heartbeat(boxID, false, 0) // active=false: request finished, start idle timer
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
