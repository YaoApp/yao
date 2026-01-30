package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	infraSandbox "github.com/yaoapp/yao/sandbox"
)

// Options for Claude executor (copied from parent package to avoid import cycle)
type Options struct {
	Command       string
	Image         string
	MaxMemory     string
	MaxCPU        float64
	Timeout       time.Duration
	Arguments     map[string]interface{}
	UserID        string
	ChatID        string
	MCPConfig     []byte
	SkillsDir     string
	ConnectorHost string
	ConnectorKey  string
	Model         string
}

// Executor implements the sandbox.Executor interface for Claude CLI
type Executor struct {
	manager       *infraSandbox.Manager
	containerName string
	opts          *Options
	workDir       string
}

// NewExecutor creates a new Claude executor
func NewExecutor(manager *infraSandbox.Manager, opts interface{}) (*Executor, error) {
	if manager == nil {
		return nil, fmt.Errorf("manager is required")
	}

	// Type assertion to get options
	var execOpts *Options
	switch o := opts.(type) {
	case *Options:
		execOpts = o
	default:
		// Try to convert from map or other struct
		return nil, fmt.Errorf("invalid options type: %T", opts)
	}

	if execOpts == nil {
		return nil, fmt.Errorf("options is required")
	}
	if execOpts.UserID == "" {
		return nil, fmt.Errorf("UserID is required")
	}
	if execOpts.ChatID == "" {
		return nil, fmt.Errorf("ChatID is required")
	}

	// Create or get container
	ctx := context.Background()
	container, err := manager.GetOrCreate(ctx, execOpts.UserID, execOpts.ChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Get workspace directory from config
	config := manager.GetConfig()
	workDir := config.ContainerWorkDir
	if workDir == "" {
		workDir = "/workspace"
	}

	return &Executor{
		manager:       manager,
		containerName: container.Name,
		opts:          execOpts,
		workDir:       workDir,
	}, nil
}

// Stream runs the Claude CLI with streaming output
func (e *Executor) Stream(ctx *agentContext.Context, messages []agentContext.Message, handler message.StreamFunc) (*agentContext.CompletionResponse, error) {
	stdCtx := context.Background()
	if ctx != nil && ctx.Context != nil {
		stdCtx = ctx.Context
	}

	// Write CCR config file to container before executing
	if err := e.writeCCRConfig(stdCtx); err != nil {
		return nil, fmt.Errorf("failed to write CCR config: %w", err)
	}

	// Build Claude CLI command using stored options
	cmd, env, err := BuildCommand(messages, e.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	// Prepare execution options
	execOpts := &infraSandbox.ExecOptions{
		WorkDir: e.workDir,
		Env:     env,
	}

	if e.opts != nil && e.opts.Timeout > 0 {
		execOpts.Timeout = e.opts.Timeout
	}

	reader, err := e.manager.Stream(stdCtx, e.containerName, cmd, execOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}
	defer reader.Close()

	// Parse streaming output
	return e.parseStream(reader, handler)
}

// writeCCRConfig writes the CCR configuration file to the container
func (e *Executor) writeCCRConfig(ctx context.Context) error {
	// Build CCR config
	configJSON, err := BuildCCRConfig(e.opts)
	if err != nil {
		return fmt.Errorf("failed to build CCR config: %w", err)
	}

	// Write config to container's CCR directory
	configPath := "/home/sandbox/.claude-code-router/config.json"
	if err := e.manager.WriteFile(ctx, e.containerName, configPath, configJSON); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", configPath, err)
	}

	return nil
}

// Execute runs the Claude CLI and returns the response
func (e *Executor) Execute(ctx *agentContext.Context, messages []agentContext.Message) (*agentContext.CompletionResponse, error) {
	return e.Stream(ctx, messages, nil)
}

// parseStream parses Claude CLI streaming output
func (e *Executor) parseStream(reader io.Reader, handler message.StreamFunc) (*agentContext.CompletionResponse, error) {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size for potentially large outputs
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var textContent strings.Builder
	var toolCalls []agentContext.ToolCall
	var model string
	var usage *message.UsageInfo

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Note: Docker stream demuxing is handled by sandbox.Manager.Stream()
		// which uses stdcopy.StdCopy to properly separate stdout/stderr

		// Try to parse as JSON (Claude CLI --output-format stream-json)
		var msg StreamMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Not JSON, might be plain text output
			textContent.WriteString(line)
			textContent.WriteString("\n")
			continue
		}

		// Process different message types
		switch msg.Type {
		case "content_block_delta":
			// Streaming text content
			if delta, ok := msg.Content.(map[string]interface{}); ok {
				if text, ok := delta["text"].(string); ok {
					textContent.WriteString(text)
					// Send to stream handler if available
					if handler != nil {
						handler(message.ChunkText, []byte(text))
					}
				}
			}

		case "message_delta":
			// Message completion with usage
			if content, ok := msg.Content.(map[string]interface{}); ok {
				if usageData, ok := content["usage"].(map[string]interface{}); ok {
					usage = &message.UsageInfo{}
					if v, ok := usageData["input_tokens"].(float64); ok {
						usage.PromptTokens = int(v)
					}
					if v, ok := usageData["output_tokens"].(float64); ok {
						usage.CompletionTokens = int(v)
					}
					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
				}
			}

		case "message_start":
			// Extract model from message_start
			if content, ok := msg.Content.(map[string]interface{}); ok {
				if m, ok := content["model"].(string); ok {
					model = m
				}
			}

		case "content_block_start":
			// Might contain tool use blocks
			if block, ok := msg.Content.(map[string]interface{}); ok {
				if block["type"] == "tool_use" {
					toolCall := agentContext.ToolCall{
						ID:   getString(block, "id"),
						Type: agentContext.ToolTypeFunction,
						Function: agentContext.Function{
							Name:      getString(block, "name"),
							Arguments: "{}",
						},
					}
					toolCalls = append(toolCalls, toolCall)
				}
			}

		case "error":
			return nil, fmt.Errorf("Claude CLI error: %s", msg.Error)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	// Build response
	response := &agentContext.CompletionResponse{
		ID:           fmt.Sprintf("sandbox-%d", time.Now().UnixNano()),
		Model:        model,
		Created:      time.Now().Unix(),
		Role:         "assistant",
		Content:      textContent.String(),
		FinishReason: agentContext.FinishReasonStop,
	}

	// Add tool calls if any
	if len(toolCalls) > 0 {
		response.ToolCalls = toolCalls
		response.FinishReason = agentContext.FinishReasonToolCalls
	}

	// Add usage if available
	if usage != nil {
		response.Usage = usage
	}

	return response, nil
}

// ReadFile reads a file from the container
func (e *Executor) ReadFile(ctx context.Context, path string) ([]byte, error) {
	// Make path absolute if not
	if !strings.HasPrefix(path, "/") {
		path = e.workDir + "/" + path
	}
	return e.manager.ReadFile(ctx, e.containerName, path)
}

// WriteFile writes content to a file in the container
func (e *Executor) WriteFile(ctx context.Context, path string, content []byte) error {
	// Make path absolute if not
	if !strings.HasPrefix(path, "/") {
		path = e.workDir + "/" + path
	}
	return e.manager.WriteFile(ctx, e.containerName, path, content)
}

// ListDir lists directory contents in the container
func (e *Executor) ListDir(ctx context.Context, path string) ([]infraSandbox.FileInfo, error) {
	// Make path absolute if not
	if !strings.HasPrefix(path, "/") {
		path = e.workDir + "/" + path
	}

	return e.manager.ListDir(ctx, e.containerName, path)
}

// Exec executes a command in the container
func (e *Executor) Exec(ctx context.Context, cmd []string) (string, error) {
	result, err := e.manager.Exec(ctx, e.containerName, cmd, &infraSandbox.ExecOptions{
		WorkDir: e.workDir,
	})
	if err != nil {
		return "", err
	}

	if result.ExitCode != 0 {
		return result.Stdout, fmt.Errorf("command exited with code %d: %s", result.ExitCode, result.Stderr)
	}

	return result.Stdout, nil
}

// GetWorkDir returns the container workspace directory
func (e *Executor) GetWorkDir() string {
	return e.workDir
}

// Close releases the executor resources and removes the container
func (e *Executor) Close() error {
	if e.manager != nil && e.containerName != "" {
		ctx := context.Background()
		return e.manager.Remove(ctx, e.containerName)
	}
	return nil
}

// Helper function to get string from map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
