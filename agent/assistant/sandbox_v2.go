package assistant

import (
	stdContext "context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output/message"
	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	sandboxTypes "github.com/yaoapp/yao/agent/sandbox/v2/types"
	store "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/llmprovider"
	infraV2 "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/setting"
	traceTypes "github.com/yaoapp/yao/trace/types"
	"github.com/yaoapp/yao/workspace"
)

// HasSandboxV2 returns true if the assistant has a V2 sandbox configuration.
func (ast *Assistant) HasSandboxV2() bool {
	return ast.SandboxV2 != nil
}

// sandboxV2InitResult bundles everything returned by initSandboxV2.
type sandboxV2InitResult struct {
	Runner       sandboxTypes.Runner
	Computer     infraV2.Computer
	Config       *sandboxTypes.SandboxConfig
	Cleanup      func()
	LoadingMsgID string
	Roles        map[string]connector.Connector
}

// initSandboxV2 initializes the V2 sandbox: obtains a Computer, gets a Runner,
// resolves the role matrix, runs Prepare, and returns the result.
//
// A shallow copy of ast.SandboxV2 is made so that concurrent requests to the
// same assistant each get their own mutable config (Owner, ID, NodeID, etc.).
func (ast *Assistant) initSandboxV2(ctx *context.Context, opts *context.Options) (*sandboxV2InitResult, error) {
	cfgCopy := *ast.SandboxV2
	cfg := &cfgCopy
	manager := infraV2.M()

	loadingMsg := &message.Message{
		Type: message.TypeLoading,
		Props: map[string]any{
			"message": i18n.T(ctx.Locale, "sandbox.preparing"),
		},
	}
	loadingMsgID, _ := ctx.SendStream(loadingMsg)

	stdCtx := ctx.Context

	// 1. Runner resolution (three-layer dispatch).
	globalRunner := ""
	if sandboxv2.GlobalRunnerFunc != nil {
		globalRunner = sandboxv2.GlobalRunnerFunc()
	}
	runnerName, err := sandboxv2.ResolveRunner("", &cfg.Runner, globalRunner)
	if err != nil {
		closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
		return nil, fmt.Errorf("resolve runner: %w", err)
	}
	mode := sandboxv2.ResolveMode(runnerName, cfg.Computer.Image)

	// 2. Local short-circuit: yaocode runner runs in-process, no Computer needed.
	if mode == "local" {
		r, rErr := sandboxv2.Get(runnerName)
		if rErr != nil {
			closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
			return nil, fmt.Errorf("get runner %q: %w", runnerName, rErr)
		}

		conn, _, _ := ast.GetConnector(ctx, opts)
		roles := resolveRoles(conn, ctx.Authorized)

		assistantDir := ""
		skillsDir := ""
		if ast.Path != "" {
			assistantDir = filepath.Join(config.Conf.AppSource, ast.Path)
			dir := filepath.Join(assistantDir, "skills")
			if info, e := os.Stat(dir); e == nil && info.IsDir() {
				skillsDir = dir
			}
		}

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

		if err := r.Prepare(stdCtx, &sandboxTypes.PrepareRequest{
			Computer:     nil,
			Config:       cfg,
			Connector:    conn,
			Roles:        roles,
			AssistantID:  ast.ID,
			SkillsDir:    skillsDir,
			AssistantDir: assistantDir,
			MCPServers:   mcpServers,
			ConfigHash:   ast.ConfigHash,
			RunSteps:     sandboxv2.RunPrepareSteps,
		}); err != nil {
			closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
			return nil, fmt.Errorf("runner.Prepare: %w", err)
		}

		return &sandboxV2InitResult{
			Runner:       r,
			Config:       cfg,
			Cleanup:      func() {},
			LoadingMsgID: loadingMsgID,
			Roles:        roles,
		}, nil
	}

	// 3. Remote runner path: resolve connector.
	conn, _, err := ast.GetConnector(ctx, opts)
	if err != nil {
		closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
		return nil, fmt.Errorf("get connector: %w", err)
	}

	roles := resolveRoles(conn, ctx.Authorized)

	cfg.DisplayName = buildBoxDisplayName(ctx, ast.ID, ast.Name)

	// 4. Image existence check + pull (for box mode).
	if cfg.Computer.Image != "" && manager != nil {
		nodeID, kind, resolveErr := sandboxv2.ResolveNodeID(ctx, cfg, manager, runnerName, mode)
		if resolveErr != nil {
			log.Printf("[sandbox/v2] ResolveNodeID pre-check failed (will retry in GetComputer): %v", resolveErr)
		}
		if resolveErr == nil && kind == "box" && nodeID != "" {
			updateLoadingV2(ctx, loadingMsgID, "sandbox.starting")
			exists, existsErr := manager.ImageExists(stdCtx, nodeID, cfg.Computer.Image)
			if existsErr != nil {
				log.Printf("[sandbox/v2] image exists check failed on node %s: %v", nodeID, existsErr)
			}
			if existsErr == nil && !exists {
				updateLoadingV2(ctx, loadingMsgID, "sandbox.pulling_image")
				var pullUserID, pullTeamID string
				if ctx.Authorized != nil {
					pullUserID = ctx.Authorized.UserID
					pullTeamID = ctx.Authorized.TeamID
				}
				pullOpts := buildImagePullOptions(pullUserID, pullTeamID)
				ch, pullErr := manager.PullImage(stdCtx, nodeID, cfg.Computer.Image, pullOpts)
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

	// 5. Obtain Computer.
	updateLoadingV2(ctx, loadingMsgID, "sandbox.starting")
	computer, identifier, err := sandboxv2.GetComputer(ctx, cfg, manager, runnerName, mode)
	if err != nil {
		closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
		return nil, fmt.Errorf("getComputer failed: %w", err)
	}
	_ = identifier

	if computer == nil {
		closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
		return nil, fmt.Errorf("GetComputer returned nil computer for runner=%s mode=%s", runnerName, mode)
	}

	// 6. Get Runner.
	runner, err := sandboxv2.Get(runnerName)
	if err != nil {
		sandboxv2.LifecycleAction(stdCtx, cfg, computer, manager)
		closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
		return nil, fmt.Errorf("get runner %q: %w", runnerName, err)
	}

	// 7. Resolve assistant directory and skills subdirectory.
	assistantDir := ""
	skillsDir := ""
	if ast.Path != "" {
		assistantDir = filepath.Join(config.Conf.AppSource, ast.Path)
		dir := filepath.Join(assistantDir, "skills")
		if info, e := os.Stat(dir); e == nil && info.IsDir() {
			skillsDir = dir
		}
	}

	// 8. Convert MCP servers.
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

	// 9. Runner.Prepare (standard context).
	err = runner.Prepare(stdCtx, &sandboxTypes.PrepareRequest{
		Computer:     computer,
		Config:       cfg,
		Connector:    conn,
		Roles:        roles,
		AssistantID:  ast.ID,
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
		return nil, fmt.Errorf("runner.Prepare: %w", err)
	}

	ctx.SetComputer(computer)

	cleanup := func() {
		cleanCtx, cancel := stdContext.WithTimeout(stdContext.Background(), 5*time.Second)
		defer cancel()
		runner.Cleanup(cleanCtx, computer)
		sandboxv2.LifecycleAction(cleanCtx, cfg, computer, manager)
	}

	return &sandboxV2InitResult{
		Runner:       runner,
		Computer:     computer,
		Config:       cfg,
		Cleanup:      cleanup,
		LoadingMsgID: loadingMsgID,
		Roles:        roles,
	}, nil
}

// sandboxV2StreamParams groups arguments for executeSandboxV2Stream.
type sandboxV2StreamParams struct {
	Messages     []context.Message
	AgentNode    traceTypes.Node
	Handler      message.StreamFunc
	Runner       sandboxTypes.Runner
	Computer     infraV2.Computer
	Config       *sandboxTypes.SandboxConfig
	LoadingMsgID string
	Options      *context.Options
	Roles        map[string]connector.Connector
}

// executeSandboxV2Stream calls the V2 Runner.Stream and wraps it in the
// standard completion response.
func (ast *Assistant) executeSandboxV2Stream(
	ctx *context.Context, p *sandboxV2StreamParams,
) (*context.CompletionResponse, error) {
	_ = p.AgentNode

	cfg := p.Config
	manager := infraV2.M()

	// Build system prompt (parse $CTX variables the same way as buildSystemPrompts).
	var systemPrompt string
	if len(ast.Prompts) > 0 {
		ctxVars := ast.buildContextVariables(ctx)
		parsed := store.Prompts(ast.Prompts).Parse(ctxVars)
		for _, pr := range parsed {
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
		Roles:        p.Roles,
		AssistantID:  ast.ID,
		Messages:     p.Messages,
		SystemPrompt: systemPrompt,
		ChatID:       ctx.ChatID,
		Token:        tok,
		Logger:       ctx.Logger,
		UserExplicit: p.Options != nil && p.Options.Connector != "",
		Locale:       ctx.Locale,
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

// resolveRoles builds the role → connector map using the llmprovider role system.
// The primary connector (user-selected or system default) becomes "default";
// other roles (heavy, light, vision) are fetched from llmprovider settings.
func resolveRoles(conn connector.Connector, identity llmprovider.Identity) map[string]connector.Connector {
	roles := map[string]connector.Connector{}
	if conn != nil {
		roles["default"] = conn
	}
	if llmprovider.Global == nil || identity == nil {
		return roles
	}
	for _, role := range []string{"heavy", "light", "vision"} {
		if c, err := llmprovider.Global.GetRoleModelBy(role, identity); err == nil {
			roles[role] = c
		}
	}
	return roles
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

// buildImagePullOptions returns ImagePullOptions with registry credentials
// injected from the global SandboxRegistryConfig if available.
func buildImagePullOptions(userID, teamID string) infraV2.ImagePullOptions {
	opts := infraV2.ImagePullOptions{}
	if setting.Global == nil {
		return opts
	}

	saved, err := setting.Global.GetMerged(userID, teamID, "sandbox.registry")
	if err != nil || len(saved) == 0 {
		return opts
	}

	url, _ := saved["registry_url"].(string)
	user, _ := saved["username"].(string)
	pass, _ := saved["password"].(string)
	if url != "" && user != "" {
		opts.Auth = &infraV2.RegistryAuth{
			Server:   url,
			Username: user,
			Password: pass,
		}
	}
	return opts
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
