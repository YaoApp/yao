package assistant

import (
	stdContext "context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output/message"
	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	sandboxTypes "github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/config"
	infraV2 "github.com/yaoapp/yao/sandbox/v2"
	traceTypes "github.com/yaoapp/yao/trace/types"
	"github.com/yaoapp/yao/workspace"
)

// HasSandboxV2 returns true if the assistant has a V2 sandbox configuration.
func (ast *Assistant) HasSandboxV2() bool {
	return ast.SandboxV2 != nil
}

// initSandboxV2 initializes the V2 sandbox: obtains a Computer, gets a Runner,
// runs Prepare, and returns the runner, computer, cleanup closure, loading
// message ID, and any error.
func (ast *Assistant) initSandboxV2(ctx *context.Context, opts *context.Options) (
	sandboxTypes.Runner, infraV2.Computer, func(), string, error,
) {
	cfg := ast.SandboxV2
	manager := infraV2.M()

	loadingMsg := &message.Message{
		Type: message.TypeLoading,
		Props: map[string]any{
			"message": i18n.T(ctx.Locale, "sandbox.preparing"),
		},
	}
	loadingMsgID, _ := ctx.SendStream(loadingMsg)

	stdCtx := ctx.Context

	// 1. Resolve connector (before Computer so proxy env vars can be injected).
	conn, _, err := ast.GetConnector(ctx, opts)
	if err != nil && cfg.Runner.Name != "yao" {
		closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
		return nil, nil, nil, "", fmt.Errorf("get connector: %w", err)
	}

	// 2. Build human-readable DisplayName from real Agent name + Workspace name.
	cfg.DisplayName = buildBoxDisplayName(ctx, ast.ID, ast.Name)

	// 2.5. Image existence check + pull (for box mode).
	if cfg.Computer.Image != "" && manager != nil {
		nodeID, kind, _ := sandboxv2.ResolveNodeID(ctx, cfg, manager)
		if kind == "box" && nodeID != "" {
			updateLoadingV2(ctx, loadingMsgID, "sandbox.starting")
			exists, existsErr := manager.ImageExists(stdCtx, nodeID, cfg.Computer.Image)
			if existsErr != nil {
				log.Printf("[sandbox/v2] image exists check failed on node %s: %v", nodeID, existsErr)
			}
			if existsErr == nil && !exists {
				updateLoadingV2(ctx, loadingMsgID, "sandbox.pulling_image")
				ch, pullErr := manager.PullImage(stdCtx, nodeID, cfg.Computer.Image, infraV2.ImagePullOptions{})
				if pullErr != nil {
					log.Printf("[sandbox/v2] image pull failed on node %s: %v (will retry in Create)", nodeID, pullErr)
				} else if ch != nil {
					for p := range ch {
						if p.Error != "" {
							log.Printf("[sandbox/v2] image pull progress error: %s", p.Error)
							break
						}
					}
				}
			}
		}
	}

	// 3. Obtain Computer.
	updateLoadingV2(ctx, loadingMsgID, "sandbox.starting")
	computer, identifier, err := sandboxv2.GetComputer(ctx, cfg, manager)
	if err != nil {
		closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
		return nil, nil, nil, "", fmt.Errorf("getComputer failed: %w", err)
	}
	_ = identifier

	// 4. Get Runner.
	runner, err := sandboxv2.Get(cfg.Runner.Name)
	if err != nil {
		sandboxv2.LifecycleAction(stdCtx, cfg, computer, manager)
		closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
		return nil, nil, nil, "", fmt.Errorf("get runner %q: %w", cfg.Runner.Name, err)
	}

	// 5. Resolve assistant directory and skills subdirectory.
	assistantDir := ""
	skillsDir := ""
	if ast.Path != "" {
		assistantDir = filepath.Join(config.Conf.AppSource, ast.Path)
		dir := filepath.Join(assistantDir, "skills")
		if info, e := os.Stat(dir); e == nil && info.IsDir() {
			skillsDir = dir
		}
	}

	// 6. Convert MCP servers.
	var mcpServers []sandboxTypes.MCPServer
	if ast.MCP != nil {
		for _, s := range ast.MCP.Servers {
			mcpServers = append(mcpServers, sandboxTypes.MCPServer{
				ServerID:  s.ServerID,
				Resources: s.Resources,
				Tools:     s.Tools,
			})
		}
	}

	// 7. Runner.Prepare (standard context).
	err = runner.Prepare(stdCtx, &sandboxTypes.PrepareRequest{
		Computer:     computer,
		Config:       cfg,
		Connector:    conn,
		SkillsDir:    skillsDir,
		AssistantDir: assistantDir,
		MCPServers:   mcpServers,
		ConfigHash:   ast.ConfigHash,
		RunSteps:     sandboxv2.RunPrepareSteps,
	})
	if err != nil {
		runner.Cleanup(stdCtx, computer)
		sandboxv2.LifecycleAction(stdCtx, cfg, computer, manager)
		closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
		return nil, nil, nil, "", fmt.Errorf("runner.Prepare: %w", err)
	}

	// Inject computer + workspace into context so Create/Next hooks
	// can access ctx.computer and ctx.workspace.
	ctx.SetComputer(computer)

	cleanup := func() {
		cleanCtx, cancel := stdContext.WithTimeout(stdContext.Background(), 5*time.Second)
		defer cancel()
		runner.Cleanup(cleanCtx, computer)
		sandboxv2.LifecycleAction(cleanCtx, cfg, computer, manager)
	}

	return runner, computer, cleanup, loadingMsgID, nil
}

// sandboxV2StreamParams groups arguments for executeSandboxV2Stream.
type sandboxV2StreamParams struct {
	Messages     []context.Message
	AgentNode    traceTypes.Node
	Handler      message.StreamFunc
	Runner       sandboxTypes.Runner
	Computer     infraV2.Computer
	LoadingMsgID string
	Options      *context.Options
}

// executeSandboxV2Stream calls the V2 Runner.Stream and wraps it in the
// standard completion response.
func (ast *Assistant) executeSandboxV2Stream(
	ctx *context.Context, p *sandboxV2StreamParams,
) (*context.CompletionResponse, error) {
	_ = p.AgentNode

	cfg := ast.SandboxV2
	manager := infraV2.M()

	// Build system prompt.
	var systemPrompt string
	if len(ast.Prompts) > 0 {
		for _, pr := range ast.Prompts {
			if pr.Role == "system" && pr.Content != "" {
				systemPrompt = pr.Content
				break
			}
		}
	}

	// Resolve connector for Stream (respects user-selected connector via opts).
	conn, _, _ := ast.GetConnector(ctx, p.Options)

	var tok *sandboxTypes.SandboxToken
	if ctx.Authorized != nil {
		var err error
		tok, err = sandboxv2.IssueSandboxToken(ctx.Authorized.TeamID, ctx.Authorized.UserID)
		if err != nil {
			return nil, fmt.Errorf("issue sandbox token: %w", err)
		}
	}

	streamReq := &sandboxTypes.StreamRequest{
		Computer:     p.Computer,
		Config:       cfg,
		Connector:    conn,
		Messages:     p.Messages,
		SystemPrompt: systemPrompt,
		ChatID:       ctx.ChatID,
		Token:        tok,
		Logger:       ctx.Logger,
	}

	execReq := &sandboxv2.ExecuteRequest{
		Computer:     p.Computer,
		Runner:       p.Runner,
		Config:       cfg,
		StreamReq:    streamReq,
		Manager:      manager,
		LoadingMsgID: p.LoadingMsgID,
	}

	return sandboxv2.ExecuteSandboxStream(ctx, execReq, p.Handler)
}

// initStandaloneWorkspace loads the workspace FS into context when no sandbox
// is configured but the user selected a workspace (metadata["workspace_id"]).
func (ast *Assistant) initStandaloneWorkspace(ctx *context.Context) {
	if ctx.Metadata == nil {
		return
	}
	wsID, _ := ctx.Metadata["workspace_id"].(string)
	if wsID == "" {
		return
	}

	stdCtx := ctx.Context
	wsFS, err := workspace.M().FS(stdCtx, wsID)
	if err != nil {
		log.Printf("[assistant] initStandaloneWorkspace: failed to load workspace %s: %v", wsID, err)
		return
	}
	ctx.SetWorkspace(wsFS)
}

// buildBoxDisplayName constructs a human-readable display name for a Box
// using the locale-resolved Agent name and Workspace name (matching the UI list pages).
func buildBoxDisplayName(ctx *context.Context, assistantID, rawName string) string {
	agentName := i18n.Tr(assistantID, ctx.Locale, rawName)

	wsName := ""
	if ctx.Metadata != nil {
		if wsID, ok := ctx.Metadata["workspace_id"].(string); ok && wsID != "" {
			if wsm := workspace.M(); wsm != nil {
				if ws, err := wsm.Get(ctx.Context, wsID); err == nil && ws != nil {
					wsName = ws.Name
				}
			}
		}
	}

	if agentName != "" && wsName != "" {
		return agentName + " / " + wsName
	}
	if agentName != "" {
		return agentName
	}
	if wsName != "" {
		return wsName
	}
	return ""
}

func updateLoadingV2(ctx *context.Context, loadingMsgID, msgKey string) {
	if loadingMsgID == "" || ctx == nil || msgKey == "" {
		return
	}
	msg := &message.Message{
		MessageID:   loadingMsgID,
		Delta:       true,
		DeltaAction: message.DeltaReplace,
		Type:        message.TypeLoading,
		Props: map[string]any{
			"message": i18n.T(ctx.Locale, msgKey),
		},
	}
	ctx.Send(msg)
}

func closeLoadingV2(ctx *context.Context, loadingMsgID, msgKey string) {
	if loadingMsgID == "" || ctx == nil {
		return
	}
	props := map[string]any{"done": true}
	if msgKey != "" {
		props["message"] = i18n.T(ctx.Locale, msgKey)
	} else {
		props["message"] = ""
	}
	doneMsg := &message.Message{
		MessageID:   loadingMsgID,
		Delta:       true,
		DeltaAction: message.DeltaReplace,
		Type:        message.TypeLoading,
		Props:       props,
	}
	ctx.Send(doneMsg)
}
