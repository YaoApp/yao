package assistant

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output/message"
	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	sandboxTypes "github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/config"
	infraV2 "github.com/yaoapp/yao/sandbox/v2"
	traceTypes "github.com/yaoapp/yao/trace/types"
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

	// 2. Obtain Computer (passes connector for OPENAI_PROXY_* env injection).
	computer, identifier, err := sandboxv2.GetComputer(ctx, cfg, manager, conn)
	if err != nil {
		closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
		return nil, nil, nil, "", fmt.Errorf("getComputer failed: %w", err)
	}
	_ = identifier

	// 3. Get Runner.
	runner, err := sandboxv2.Get(cfg.Runner.Name)
	if err != nil {
		sandboxv2.LifecycleAction(stdCtx, cfg, computer, manager)
		closeLoadingV2(ctx, loadingMsgID, "sandbox.failed")
		return nil, nil, nil, "", fmt.Errorf("get runner %q: %w", cfg.Runner.Name, err)
	}

	// 4. Resolve skills directory.
	skillsDir := ""
	if ast.Path != "" {
		dir := filepath.Join(config.Conf.AppSource, ast.Path, "skills")
		if info, e := os.Stat(dir); e == nil && info.IsDir() {
			skillsDir = dir
		}
	}

	// 5. Convert MCP servers.
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

	// 6. Runner.Prepare (standard context).
	err = runner.Prepare(stdCtx, &sandboxTypes.PrepareRequest{
		Computer:   computer,
		Config:     cfg,
		Connector:  conn,
		SkillsDir:  skillsDir,
		MCPServers: mcpServers,
		ConfigHash: ast.ConfigHash,
		RunSteps:   sandboxv2.RunPrepareSteps,
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
		// Defensive fallback — executeSandboxV2Stream defer handles the
		// normal case; this covers paths that never reach execution.
	}

	return runner, computer, cleanup, loadingMsgID, nil
}

// executeSandboxV2Stream calls the V2 Runner.Stream and wraps it in the
// standard completion response.
func (ast *Assistant) executeSandboxV2Stream(
	ctx *context.Context,
	completionMessages []context.Message,
	agentNode traceTypes.Node,
	streamHandler message.StreamFunc,
	runner sandboxTypes.Runner,
	computer infraV2.Computer,
	loadingMsgID string,
) (*context.CompletionResponse, error) {
	_ = agentNode

	cfg := ast.SandboxV2
	manager := infraV2.M()

	// Close the "preparing" loading on first output.
	if loadingMsgID != "" {
		closeLoadingV2(ctx, loadingMsgID, "")
	}

	// Build system prompt.
	var systemPrompt string
	if len(ast.Prompts) > 0 {
		for _, p := range ast.Prompts {
			if p.Role == "system" && p.Content != "" {
				systemPrompt = p.Content
				break
			}
		}
	}

	// Resolve connector for Stream.
	conn, _, _ := ast.GetConnector(ctx)

	streamReq := &sandboxTypes.StreamRequest{
		Computer:     computer,
		Config:       cfg,
		Connector:    conn,
		Messages:     completionMessages,
		SystemPrompt: systemPrompt,
		ChatID:       ctx.ChatID,
	}

	execReq := &sandboxv2.ExecuteRequest{
		Computer:  computer,
		Runner:    runner,
		Config:    cfg,
		StreamReq: streamReq,
		Manager:   manager,
	}

	return sandboxv2.ExecuteSandboxStream(ctx, execReq, streamHandler)
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
