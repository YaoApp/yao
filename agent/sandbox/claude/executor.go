package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	goujson "github.com/yaoapp/gou/json"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/attachment"
	infraSandbox "github.com/yaoapp/yao/sandbox"
	"github.com/yaoapp/yao/sandbox/ipc"
)

// Options for Claude executor (copied from parent package to avoid import cycle)
type Options struct {
	Command          string
	Image            string
	MaxMemory        string
	MaxCPU           float64
	Timeout          time.Duration
	Arguments        map[string]interface{}
	UserID           string
	ChatID           string
	MCPConfig        []byte
	MCPTools         map[string]*ipc.MCPTool // MCP tools to expose via IPC
	SkillsDir        string
	SystemPrompt     string // System prompt from assistant prompts.yml
	ConnectorHost    string
	ConnectorKey     string
	Model            string
	ConnectorOptions map[string]interface{} // Extra connector options (e.g., thinking, max_tokens)
	Secrets          map[string]string      // Secrets to pass to container (e.g., GITHUB_TOKEN)
}

// Executor implements the sandbox.Executor interface for Claude CLI
type Executor struct {
	manager       *infraSandbox.Manager
	containerName string
	opts          *Options
	workDir       string
	loadingMsgID  string // Loading message ID for tool execution updates
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
	// Note: IPC session is created by manager.createContainer, socket is already bind mounted
	ctx := context.Background()
	createOpts := infraSandbox.CreateOptions{
		UserID: execOpts.UserID,
		ChatID: execOpts.ChatID,
		Image:  execOpts.Image,
	}
	container, err := manager.GetOrCreate(ctx, execOpts.UserID, execOpts.ChatID, createOpts)
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

// SetLoadingMsgID sets the loading message ID for tool execution updates
func (e *Executor) SetLoadingMsgID(id string) {
	e.loadingMsgID = id
}

// Stream runs the Claude CLI with streaming output
func (e *Executor) Stream(ctx *agentContext.Context, messages []agentContext.Message, handler message.StreamFunc) (*agentContext.CompletionResponse, error) {
	// Create a cancellable context for this stream operation
	// We need to handle both:
	// 1. HTTP context cancellation (client disconnect)
	// 2. InterruptController cancellation (user clicks "stop" button)
	//
	// Note on InterruptController:
	// - ctx.Interrupt.Context() is only cancelled when InterruptForce && len(Messages) == 0
	// - When user sends messages with the interrupt, the context is NOT cancelled
	// - We use ctx.Interrupt.IsInterrupted() to check for any interrupt signal
	stdCtx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Start a goroutine to monitor for interrupts and HTTP context cancellation
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stdCtx.Done():
				// Already cancelled, exit
				return
			case <-ticker.C:
				// Check if there's a pending interrupt signal using Peek()
				// This works even when Messages are included (which doesn't cancel the context)
				if ctx != nil && ctx.Interrupt != nil {
					if signal := ctx.Interrupt.Peek(); signal != nil {
						cancelFunc()
						return
					}
				}
				// Check InterruptController.IsInterrupted() (for context-cancelled interrupts)
				if ctx != nil && ctx.Interrupt != nil && ctx.Interrupt.IsInterrupted() {
					cancelFunc()
					return
				}
				// Check HTTP context
				if ctx != nil && ctx.Context != nil {
					select {
					case <-ctx.Context.Done():
						cancelFunc()
						return
					default:
					}
				}
			}
		}
	}()

	// Set MCP tools for this request (dynamic, runtime configuration)
	if len(e.opts.MCPTools) > 0 {
		ipcManager := e.manager.GetIPCManager()
		if ipcManager != nil {
			if session, ok := ipcManager.Get(e.opts.ChatID); ok {
				session.SetMCPTools(e.opts.MCPTools)
			}
		}
	}

	// Prepare environment: write configs and copy skills
	if err := e.prepareEnvironment(stdCtx); err != nil {
		return nil, fmt.Errorf("failed to prepare environment: %w", err)
	}

	// Resolve attachment URLs and write files to container
	// This converts __yao.attachment:// URLs to local file paths in /workspace/.attachments/
	if resolved, attErr := e.prepareAttachments(stdCtx, messages); attErr != nil {
		// Non-fatal: log warning and continue with original messages
		log.Printf("[sandbox] Warning: failed to prepare attachments: %v", attErr)
	} else {
		messages = resolved
	}

	// Check if we should skip Claude CLI execution
	// Skip if no prompts, no skills, and no MCP config
	if e.shouldSkipClaudeCLI() {
		// Return empty response - hooks can use sandbox API to do their work
		return &agentContext.CompletionResponse{
			ID:           fmt.Sprintf("sandbox-skip-%d", time.Now().UnixNano()),
			Model:        "sandbox",
			Created:      time.Now().Unix(),
			Role:         "assistant",
			Content:      "",
			FinishReason: agentContext.FinishReasonStop,
		}, nil
	}

	// Check if this is a continuation (Claude CLI session exists in workspace)
	isContinuation := e.hasExistingSession(stdCtx)

	// Build Claude CLI command using stored options
	cmd, env, err := BuildCommandWithContinuation(messages, e.opts, isContinuation)
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

	// Ensure reader is closed when context is cancelled or function returns
	// This is important for cleanup when user clicks "stop"
	done := make(chan struct{})
	defer func() {
		close(done)
		reader.Close()
	}()

	// Monitor for context cancellation and forcefully kill Claude CLI process
	go func() {
		// Also start a ticker to periodically check context status for debugging
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stdCtx.Done():
				// First, kill the Claude CLI process inside the container
				// This is important because closing the reader/connection alone may not stop the process
				// Use a background context since stdCtx is already cancelled
				killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Kill claude process (the Claude CLI binary)
				e.manager.KillProcess(killCtx, e.containerName, "claude")

				// Also close the reader to unblock any pending reads
				reader.Close()
				return
			case <-done:
				// Normal completion, nothing to do
				return
			case <-ticker.C:
				// Periodic check - no action needed
			}
		}
	}()

	// DEBUG: Tee the reader to write raw output to a log file for debugging
	debugLogPath := e.workDir + "/claude-cli-raw.log"
	debugReader := e.createDebugReader(stdCtx, reader, debugLogPath)

	// Parse streaming output (uses e.loadingMsgID set via SetLoadingMsgID)
	return e.parseStream(ctx, debugReader, handler)
}

// shouldSkipClaudeCLI checks if Claude CLI execution should be skipped
// Skip when: no system prompt, no skills, and no MCP config
func (e *Executor) shouldSkipClaudeCLI() bool {
	hasPrompts := e.opts.SystemPrompt != ""
	hasSkills := e.opts.SkillsDir != ""
	hasMCP := len(e.opts.MCPConfig) > 0

	// If any of these are present, execute Claude CLI
	return !hasPrompts && !hasSkills && !hasMCP
}

// hasExistingSession checks if Claude CLI has an existing session in the workspace
// Claude CLI stores session data in $HOME/.claude/projects/ (which is /workspace/.claude/projects/)
// If session data exists, we should use --continue to resume the session
func (e *Executor) hasExistingSession(ctx context.Context) bool {
	// Check if /workspace/.claude/projects/ directory has any content
	// This indicates a previous session exists
	sessionDir := e.workDir + "/.claude/projects"
	files, err := e.manager.ListDir(ctx, e.containerName, sessionDir)
	if err != nil {
		// Directory doesn't exist or error reading - no existing session
		return false
	}
	// If there are any files/directories in the projects folder, session exists
	return len(files) > 0
}

// prepareEnvironment prepares the container environment before execution
// This includes: claude-proxy config, MCP config, and Skills directory
func (e *Executor) prepareEnvironment(ctx context.Context) error {
	// 1. Write claude-proxy config and start the proxy
	if err := e.startClaudeProxy(ctx); err != nil {
		return fmt.Errorf("failed to start claude-proxy: %w", err)
	}

	// 2. Write MCP config if provided
	if len(e.opts.MCPConfig) > 0 {
		if err := e.writeMCPConfig(ctx); err != nil {
			return fmt.Errorf("failed to write MCP config: %w", err)
		}
	}

	// 3. Copy Skills directory if provided
	if e.opts.SkillsDir != "" {
		if err := e.copySkillsDirectory(ctx); err != nil {
			// Non-fatal: log warning but continue
			// Skills might not exist or be optional
			_ = err // Ignore error, skills are optional
		}
	}

	return nil
}

// startClaudeProxy writes proxy config and starts claude-proxy
func (e *Executor) startClaudeProxy(ctx context.Context) error {
	// Skip if no connector configured (e.g., test containers without claude-proxy)
	if e.opts.ConnectorHost == "" || e.opts.ConnectorKey == "" {
		return nil
	}

	// Build proxy config
	configJSON, err := BuildProxyConfig(e.opts)
	if err != nil {
		return fmt.Errorf("failed to build proxy config: %w", err)
	}

	// Create config directory (outside workspace for security - user can't see api_key/secrets)
	// /tmp/.yao/ is not visible to user's file manager
	configDir := "/tmp/.yao"
	if _, err := e.manager.Exec(ctx, e.containerName, []string{"mkdir", "-p", configDir}, nil); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	// Write config to secure location (not in /workspace/)
	configPath := configDir + "/proxy.json"
	if err := e.manager.WriteFile(ctx, e.containerName, configPath, configJSON); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", configPath, err)
	}

	// Start the proxy (only if start-claude-proxy exists in the image)
	result, err := e.manager.Exec(ctx, e.containerName, []string{"which", "start-claude-proxy"}, &infraSandbox.ExecOptions{
		WorkDir: e.workDir,
	})
	if err != nil || result.ExitCode != 0 {
		// start-claude-proxy not available (e.g., alpine test image), skip
		return nil
	}

	// Start the proxy
	result, err = e.manager.Exec(ctx, e.containerName, []string{"start-claude-proxy"}, &infraSandbox.ExecOptions{
		WorkDir: e.workDir,
		Env: map[string]string{
			"WORKSPACE": e.workDir,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start claude-proxy: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("claude-proxy failed to start: %s", result.Stderr)
	}

	return nil
}

// writeMCPConfig writes the MCP configuration file to the container workspace
func (e *Executor) writeMCPConfig(ctx context.Context) error {
	if len(e.opts.MCPConfig) == 0 {
		return nil
	}

	// Write MCP config to workspace (.mcp.json)
	mcpPath := e.workDir + "/.mcp.json"
	if err := e.manager.WriteFile(ctx, e.containerName, mcpPath, e.opts.MCPConfig); err != nil {
		return fmt.Errorf("failed to write MCP config to %s: %w", mcpPath, err)
	}

	return nil
}

// copySkillsDirectory copies the skills directory to the container
func (e *Executor) copySkillsDirectory(ctx context.Context) error {
	if e.opts.SkillsDir == "" {
		return nil
	}

	// Target path in container: /workspace/.claude/skills/
	// This follows Claude CLI's expected skills location
	claudeDir := e.workDir + "/.claude"

	// Create .claude directory first
	if _, err := e.manager.Exec(ctx, e.containerName, []string{"mkdir", "-p", claudeDir}, nil); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Copy skills from host to container
	// CopyToContainer extracts tar to containerPath, and createTarFromPath uses
	// filepath.Dir(hostPath) as base, so if hostPath is /path/to/skills,
	// tar entries are like "skills/skill-name/SKILL.md"
	// Extracting to /workspace/.claude/ gives us /workspace/.claude/skills/skill-name/SKILL.md
	if err := e.manager.CopyToContainer(ctx, e.containerName, e.opts.SkillsDir, claudeDir); err != nil {
		return fmt.Errorf("failed to copy skills to container: %w", err)
	}

	return nil
}

// prepareAttachments resolves __yao.attachment:// URLs in messages,
// writes the actual files to the container's /workspace/.attachments/ directory,
// and replaces the attachment content parts with text references to the file paths.
// This allows Claude CLI to read the files using its built-in Read/Bash tools.
func (e *Executor) prepareAttachments(ctx context.Context, messages []agentContext.Message) ([]agentContext.Message, error) {
	// Track used filenames to handle duplicates
	usedNames := make(map[string]int)
	attachmentDir := e.workDir + "/.attachments"
	dirCreated := false
	hasAttachments := false

	result := make([]agentContext.Message, len(messages))
	copy(result, messages)

	for i, msg := range result {
		if msg.Role != "user" {
			continue
		}

		// Handle content array (multimodal messages come as []interface{} from JSON)
		parts, ok := msg.Content.([]interface{})
		if !ok {
			// Try typed content parts
			if typedParts, ok := msg.Content.([]agentContext.ContentPart); ok {
				iparts := make([]interface{}, len(typedParts))
				for j, p := range typedParts {
					// Convert to map for uniform handling
					m := map[string]interface{}{"type": string(p.Type)}
					if p.Text != "" {
						m["text"] = p.Text
					}
					if p.ImageURL != nil {
						m["image_url"] = map[string]interface{}{
							"url":    p.ImageURL.URL,
							"detail": string(p.ImageURL.Detail),
						}
					}
					if p.File != nil {
						m["file"] = map[string]interface{}{
							"url":      p.File.URL,
							"filename": p.File.Filename,
						}
					}
					iparts[j] = m
				}
				parts = iparts
			} else {
				continue
			}
		}

		if len(parts) == 0 {
			continue
		}

		// Process each content part
		var textParts []string
		modified := false

		for _, item := range parts {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			partType, _ := m["type"].(string)

			switch partType {
			case "text":
				if text, ok := m["text"].(string); ok && text != "" {
					textParts = append(textParts, text)
				}

			case "image_url":
				imgData, _ := m["image_url"].(map[string]interface{})
				if imgData == nil {
					continue
				}
				url, _ := imgData["url"].(string)
				if url == "" {
					continue
				}

				uploaderName, fileID, isWrapper := attachment.Parse(url)
				if !isWrapper {
					// Not an attachment URL, keep as text reference
					textParts = append(textParts, fmt.Sprintf("[Image: %s]", url))
					modified = true
					continue
				}

				// Resolve the attachment
				ref, err := e.resolveAttachment(ctx, uploaderName, fileID, "", attachmentDir, usedNames, &dirCreated)
				if err != nil {
					log.Printf("[sandbox] Warning: failed to resolve image attachment %s: %v", fileID, err)
					textParts = append(textParts, "[Attached image: failed to load]")
					modified = true
					continue
				}

				textParts = append(textParts, ref)
				hasAttachments = true
				modified = true

			case "file":
				fileData, _ := m["file"].(map[string]interface{})
				if fileData == nil {
					continue
				}
				url, _ := fileData["url"].(string)
				hintName, _ := fileData["filename"].(string)
				if url == "" {
					continue
				}

				uploaderName, fileID, isWrapper := attachment.Parse(url)
				if !isWrapper {
					textParts = append(textParts, fmt.Sprintf("[File: %s]", url))
					modified = true
					continue
				}

				ref, err := e.resolveAttachment(ctx, uploaderName, fileID, hintName, attachmentDir, usedNames, &dirCreated)
				if err != nil {
					log.Printf("[sandbox] Warning: failed to resolve file attachment %s: %v", fileID, err)
					textParts = append(textParts, "[Attached file: failed to load]")
					modified = true
					continue
				}

				textParts = append(textParts, ref)
				hasAttachments = true
				modified = true

			default:
				// Keep other types as-is (shouldn't happen normally)
				continue
			}
		}

		if modified && len(textParts) > 0 {
			newMsg := result[i]
			newMsg.Content = strings.Join(textParts, "\n\n")
			result[i] = newMsg
		}
	}

	if !hasAttachments {
		return result, nil
	}

	return result, nil
}

// resolveAttachment reads an attachment from the attachment manager and writes it
// to the container's .attachments directory. Returns a text reference string.
func (e *Executor) resolveAttachment(
	ctx context.Context,
	uploaderName, fileID, hintName, attachmentDir string,
	usedNames map[string]int,
	dirCreated *bool,
) (string, error) {
	// Get attachment manager
	manager, exists := attachment.Managers[uploaderName]
	if !exists {
		return "", fmt.Errorf("attachment manager not found: %s", uploaderName)
	}

	// Get file info
	fileInfo, err := manager.Info(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Read file data
	data, err := manager.Read(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Determine filename
	filename := fileInfo.Filename
	if filename == "" && hintName != "" {
		filename = hintName
	}
	if filename == "" {
		// Fallback: use fileID with extension from content type
		ext := extensionFromContentType(fileInfo.ContentType)
		filename = fileID + ext
	}

	// Handle duplicate filenames
	baseName := filename
	if count, exists := usedNames[baseName]; exists {
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)
		filename = fmt.Sprintf("%s_%d%s", name, count+1, ext)
		usedNames[baseName] = count + 1
	} else {
		usedNames[baseName] = 0
	}

	// Create attachments directory if not yet created
	if !*dirCreated {
		if err := e.manager.WriteFile(ctx, e.containerName, attachmentDir+"/.keep", []byte("")); err != nil {
			return "", fmt.Errorf("failed to create attachments directory: %w", err)
		}
		*dirCreated = true
	}

	// Write file to container
	containerPath := attachmentDir + "/" + filename
	if err := e.manager.WriteFile(ctx, e.containerName, containerPath, data); err != nil {
		return "", fmt.Errorf("failed to write file to container: %w", err)
	}

	// Build human-readable size string
	sizeStr := formatFileSize(fileInfo.Bytes)

	// Return text reference
	return fmt.Sprintf("[Attached file: %s (%s, %s)]", containerPath, fileInfo.ContentType, sizeStr), nil
}

// extensionFromContentType returns a file extension for a given content type
func extensionFromContentType(contentType string) string {
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "application/pdf":
		return ".pdf"
	case "text/plain":
		return ".txt"
	case "text/html":
		return ".html"
	case "text/css":
		return ".css"
	case "text/javascript", "application/javascript":
		return ".js"
	case "application/json":
		return ".json"
	case "application/zip":
		return ".zip"
	default:
		return ""
	}
}

// formatFileSize returns a human-readable file size string
func formatFileSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
}

// Execute runs the Claude CLI and returns the response
func (e *Executor) Execute(ctx *agentContext.Context, messages []agentContext.Message) (*agentContext.CompletionResponse, error) {
	return e.Stream(ctx, messages, nil)
}

// debugWriter wraps an io.Reader to write all data to a debug log file
type debugWriter struct {
	reader  io.Reader
	logFile *os.File
	buffer  []byte
}

func (d *debugWriter) Read(p []byte) (n int, err error) {
	n, err = d.reader.Read(p)
	if n > 0 && d.logFile != nil {
		// Write raw bytes to log file
		d.logFile.Write(p[:n])
		d.logFile.Sync()
	}
	return n, err
}

func (d *debugWriter) Close() error {
	if d.logFile != nil {
		d.logFile.Close()
	}
	return nil
}

// createDebugReader creates a tee reader that writes to a debug log file
// The log file is written to the container's workspace for inspection
func (e *Executor) createDebugReader(ctx context.Context, reader io.ReadCloser, logPath string) io.Reader {
	// Create a local temp file for debug logging
	// We write to a local file first, then copy to container when done
	localLogPath := "/tmp/claude-cli-debug-" + e.containerName + ".log"
	logFile, err := os.Create(localLogPath)
	if err != nil {
		return reader
	}

	// Write header
	logFile.WriteString("=== Claude CLI Raw Output Debug Log ===\n")
	logFile.WriteString(fmt.Sprintf("Container: %s\n", e.containerName))
	logFile.WriteString(fmt.Sprintf("Time: %s\n", time.Now().Format(time.RFC3339)))
	logFile.WriteString(fmt.Sprintf("WorkDir: %s\n", e.workDir))
	logFile.WriteString("=== BEGIN OUTPUT ===\n")
	logFile.Sync()

	return &debugWriter{
		reader:  reader,
		logFile: logFile,
	}
}

// parseStream parses Claude CLI streaming output (stream-json format)
// Claude CLI output format with --include-partial-messages:
// - {"type":"system","subtype":"init",...} - initialization
// - {"type":"stream_event","event":{"delta":{"type":"text_delta","text":"..."}}} - real-time text deltas
// - {"type":"assistant","message":{...,"content":[{"type":"text","text":"..."}],...}} - complete messages
// - {"type":"result","subtype":"success",...,"result":"..."} - final result
func (e *Executor) parseStream(ctx *agentContext.Context, reader io.Reader, handler message.StreamFunc) (*agentContext.CompletionResponse, error) {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size for potentially large outputs
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var textContent strings.Builder
	var toolCalls []agentContext.ToolCall
	var model string
	var usage *message.UsageInfo
	var finalResult string
	messageStarted := false    // Track if we've sent ChunkMessageStart
	prepLoadingClosed := false // Track if "preparing sandbox" loading has been closed

	// Tool input accumulation state
	type toolState struct {
		name      string
		index     int
		inputJSON strings.Builder
		loadingID string // Each tool has its own loading message
	}
	var currentTool *toolState
	var lastToolLoadingID string // Track the last tool loading ID to close it

	// Helper function to close "preparing sandbox" loading on first output
	closePrepLoading := func() {
		if !prepLoadingClosed && e.loadingMsgID != "" && ctx != nil {
			doneMsg := &message.Message{
				MessageID:   e.loadingMsgID,
				Delta:       true,
				DeltaAction: message.DeltaReplace,
				Type:        message.TypeLoading,
				Props: map[string]interface{}{
					"message": "",
					"done":    true,
				},
			}
			ctx.Send(doneMsg)
			prepLoadingClosed = true
		}
	}

	lineCount := 0

	// Get the underlying context for cancellation checks
	var stdCtx context.Context
	if ctx != nil && ctx.Context != nil {
		stdCtx = ctx.Context
	} else {
		stdCtx = context.Background()
	}

	for scanner.Scan() {
		// Check for context cancellation on each iteration
		select {
		case <-stdCtx.Done():
			return nil, stdCtx.Err()
		default:
			// Continue processing
		}

		line := scanner.Text()
		lineCount++
		if line == "" {
			continue
		}

		// Try to parse as JSON (Claude CLI --output-format stream-json)
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Not JSON, might be plain text output
			textContent.WriteString(line)
			textContent.WriteString("\n")
			continue
		}

		msgType, _ := msg["type"].(string)

		// Process Claude CLI stream-json message types
		switch msgType {
		case "system":
			// Initialization message - extract model if available
			if m, ok := msg["model"].(string); ok {
				model = m
			}

		case "stream_event":
			// Real-time streaming event (from --include-partial-messages)
			// Format: {"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"..."}}}
			if event, ok := msg["event"].(map[string]interface{}); ok {
				eventType, _ := event["type"].(string)

				switch eventType {
				case "content_block_start":
					// Handle new content blocks
					// Format: {"event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use"|"text",...}}}
					if contentBlock, ok := event["content_block"].(map[string]interface{}); ok {
						blockType, _ := contentBlock["type"].(string)
						switch blockType {
						case "text":
							// New text block starting - add paragraph separator if we already have content
							// This ensures proper separation between text blocks across tool-use rounds
							if textContent.Len() > 0 {
								textContent.WriteString("\n\n")
								if handler != nil && messageStarted {
									handler(message.ChunkText, []byte("\n\n"))
								}
							}
						case "tool_use":
							toolName, _ := contentBlock["name"].(string)
							blockIndex := 0
							if idx, ok := event["index"].(float64); ok {
								blockIndex = int(idx)
							}
							if toolName != "" && ctx != nil {
								// Close "preparing sandbox" loading on first tool
								closePrepLoading()

								// Close previous tool loading if exists
								if lastToolLoadingID != "" {
									doneMsg := &message.Message{
										MessageID:   lastToolLoadingID,
										Delta:       true,
										DeltaAction: message.DeltaReplace,
										Type:        message.TypeLoading,
										Props: map[string]interface{}{
											"message": "",
											"done":    true,
										},
									}
									ctx.Send(doneMsg)
								}

								// Create new loading message for this tool
								locale := ctx.Locale
								toolLoadingMsg := &message.Message{
									Type: message.TypeLoading,
									Props: map[string]interface{}{
										"message": getToolDescription(toolName, locale),
									},
								}
								newLoadingID, _ := ctx.SendStream(toolLoadingMsg)

								// Initialize tool state for input accumulation
								currentTool = &toolState{
									name:      toolName,
									index:     blockIndex,
									loadingID: newLoadingID,
								}
								lastToolLoadingID = newLoadingID

								log.Printf("[Sandbox] Tool started: %s", toolName)
							}
						}
					}

				case "content_block_delta":
					if delta, ok := event["delta"].(map[string]interface{}); ok {
						deltaType, _ := delta["type"].(string)
						switch deltaType {
						case "text_delta":
							if text, ok := delta["text"].(string); ok && text != "" {
								// Close "preparing sandbox" loading on first text output
								closePrepLoading()

								// Send to stream handler for real-time output
								if handler != nil {
									// Send ChunkMessageStart first if not already started
									if !messageStarted {
										startData := message.EventMessageStartData{
											MessageID: fmt.Sprintf("sandbox-%d", time.Now().UnixNano()),
											Type:      "text",
											Timestamp: time.Now().UnixMilli(),
										}
										startDataJSON, _ := json.Marshal(startData)
										handler(message.ChunkMessageStart, startDataJSON)
										messageStarted = true
									}
									handler(message.ChunkText, []byte(text))
								}
								// Also accumulate for final response
								textContent.WriteString(text)
							}

						case "input_json_delta":
							// Accumulate tool input JSON fragments
							if currentTool != nil {
								if partialJSON, ok := delta["partial_json"].(string); ok {
									currentTool.inputJSON.WriteString(partialJSON)
								}
							}
						}
					}

				case "content_block_stop":
					// Tool input complete - parse and update loading with detailed info
					if currentTool != nil && currentTool.loadingID != "" && ctx != nil {
						inputStr := currentTool.inputJSON.String()
						if inputStr != "" {
							// Use gou/json.Parse for fault-tolerant parsing
							locale := ctx.Locale
							detailedMsg := getToolDetailedDescription(currentTool.name, inputStr, locale)
							if detailedMsg != "" {
								toolMsg := &message.Message{
									MessageID:   currentTool.loadingID,
									Delta:       true,
									DeltaAction: message.DeltaReplace,
									Type:        message.TypeLoading,
									Props: map[string]interface{}{
										"message": detailedMsg,
									},
								}
								ctx.Send(toolMsg)
								log.Printf("[Sandbox] Tool: %s -> %s", currentTool.name, detailedMsg)
							}
						}
						// Note: Don't close loading here - it will be closed when next tool starts or at end
						// Reset tool state but keep lastToolLoadingID to close it later
						currentTool = nil
					}
				}
			}

		case "assistant":
			// Assistant message - extract content
			// With --include-partial-messages, we receive real-time text via stream_event
			// The assistant message contains the full accumulated content
			if msgData, ok := msg["message"].(map[string]interface{}); ok {
				// Get model from message
				if m, ok := msgData["model"].(string); ok && model == "" {
					model = m
				}

				// Check if this is the final message (has stop_reason)
				stopReason, hasStopReason := msgData["stop_reason"].(string)
				isFinalMessage := hasStopReason && stopReason != ""

				// Extract content from final message
				// This serves as a fallback if stream_event wasn't received
				if isFinalMessage {
					if contentArr, ok := msgData["content"].([]interface{}); ok {
						for _, item := range contentArr {
							if contentItem, ok := item.(map[string]interface{}); ok {
								itemType, _ := contentItem["type"].(string)

								switch itemType {
								case "text":
									// Only use this if we haven't already accumulated text from stream_event
									if textContent.Len() == 0 {
										if text, ok := contentItem["text"].(string); ok && text != "" {
											textContent.WriteString(text)
											// Send to stream handler if available
											if handler != nil {
												if !messageStarted {
													startData := message.EventMessageStartData{
														MessageID: fmt.Sprintf("sandbox-%d", time.Now().UnixNano()),
														Type:      "text",
														Timestamp: time.Now().UnixMilli(),
													}
													startDataJSON, _ := json.Marshal(startData)
													handler(message.ChunkMessageStart, startDataJSON)
													messageStarted = true
												}
												handler(message.ChunkText, []byte(text))
											}
										}
									}

								case "tool_use":
									toolName := getString(contentItem, "name")
									toolCall := agentContext.ToolCall{
										ID:   getString(contentItem, "id"),
										Type: agentContext.ToolTypeFunction,
										Function: agentContext.Function{
											Name: toolName,
										},
									}
									// Get input as JSON string
									var inputJSONStr string
									if input, ok := contentItem["input"]; ok {
										if inputJSON, err := json.Marshal(input); err == nil {
											inputJSONStr = string(inputJSON)
											toolCall.Function.Arguments = inputJSONStr
										}
									}
									toolCalls = append(toolCalls, toolCall)

									// Create tool loading message (from complete assistant message)
									// This is a fallback for when stream_event wasn't received
									if toolName != "" && ctx != nil {
										// Close previous tool loading if exists
										if lastToolLoadingID != "" {
											doneMsg := &message.Message{
												MessageID:   lastToolLoadingID,
												Delta:       true,
												DeltaAction: message.DeltaReplace,
												Type:        message.TypeLoading,
												Props: map[string]interface{}{
													"message": "",
													"done":    true,
												},
											}
											ctx.Send(doneMsg)
										}

										// Create new loading for this tool
										locale := ctx.Locale
										detailedMsg := getToolDetailedDescription(toolName, inputJSONStr, locale)
										if detailedMsg != "" {
											toolLoadingMsg := &message.Message{
												Type: message.TypeLoading,
												Props: map[string]interface{}{
													"message": detailedMsg,
												},
											}
											newLoadingID, _ := ctx.SendStream(toolLoadingMsg)
											lastToolLoadingID = newLoadingID
											log.Printf("[Sandbox] Tool: %s -> %s", toolName, detailedMsg)
										}
									}
								}
							}
						}
					}
				}

				// Extract usage (from any message that has it)
				if usageData, ok := msgData["usage"].(map[string]interface{}); ok {
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

		case "result":
			// Final result message
			// Check if this is an error result (is_error: true)
			isError, _ := msg["is_error"].(bool)
			if result, ok := msg["result"].(string); ok {
				if isError {
					// This is an error - return it as an error
					return nil, fmt.Errorf("Claude CLI error: %s", result)
				}
				finalResult = result
			}
			// Send done signal to handler (only if message was started and not an error)
			if handler != nil && messageStarted && !isError {
				handler(message.ChunkMessageEnd, nil)
			}

		case "error":
			// Error message
			if errMsg, ok := msg["error"].(string); ok {
				return nil, fmt.Errorf("Claude CLI error: %s", errMsg)
			}
			if errObj, ok := msg["error"].(map[string]interface{}); ok {
				if errMsg, ok := errObj["message"].(string); ok {
					return nil, fmt.Errorf("Claude CLI error: %s", errMsg)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	// Close the last tool loading message if exists
	if lastToolLoadingID != "" && ctx != nil {
		doneMsg := &message.Message{
			MessageID:   lastToolLoadingID,
			Delta:       true,
			DeltaAction: message.DeltaReplace,
			Type:        message.TypeLoading,
			Props: map[string]interface{}{
				"message": "",
				"done":    true,
			},
		}
		ctx.Send(doneMsg)
	}

	// Use final result if available, otherwise use accumulated text content
	content := textContent.String()
	if finalResult != "" && content == "" {
		content = finalResult
	}

	// Build response
	response := &agentContext.CompletionResponse{
		ID:           fmt.Sprintf("sandbox-%d", time.Now().UnixNano()),
		Model:        model,
		Created:      time.Now().Unix(),
		Role:         "assistant",
		Content:      content,
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

// truncateStr truncates a string to maxLen characters
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// getToolDescription returns a human-readable, localized description for a Claude CLI tool
func getToolDescription(toolName string, locale string) string {
	// Map tool names to i18n keys
	toolKeys := map[string]string{
		"Read":         "sandbox.tool.read",
		"Write":        "sandbox.tool.write",
		"Edit":         "sandbox.tool.edit",
		"StrReplace":   "sandbox.tool.edit",
		"Bash":         "sandbox.tool.bash",
		"Shell":        "sandbox.tool.bash",
		"Glob":         "sandbox.tool.glob",
		"Grep":         "sandbox.tool.grep",
		"LS":           "sandbox.tool.ls",
		"Task":         "sandbox.tool.task",
		"WebSearch":    "sandbox.tool.web_search",
		"WebFetch":     "sandbox.tool.web_fetch",
		"TodoWrite":    "sandbox.tool.todo_write",
		"AskQuestion":  "sandbox.tool.ask_question",
		"SwitchMode":   "sandbox.tool.switch_mode",
		"ReadLints":    "sandbox.tool.read_lints",
		"EditNotebook": "sandbox.tool.edit_notebook",
	}

	if key, ok := toolKeys[toolName]; ok {
		return i18n.T(locale, key)
	}
	// For unknown tools, use the unknown key and replace {{name}} manually
	template := i18n.T(locale, "sandbox.tool.unknown")
	return strings.Replace(template, "{{name}}", toolName, 1)
}

// getToolDetailedDescription returns a detailed description with specific parameters
// It parses the tool input JSON and extracts key information to show users
func getToolDetailedDescription(toolName string, inputJSON string, locale string) string {
	// Parse the input JSON using fault-tolerant parser
	parsed, err := goujson.Parse(inputJSON)
	if err != nil {
		// Fall back to basic description if parsing fails
		return getToolDescription(toolName, locale)
	}

	input, ok := parsed.(map[string]interface{})
	if !ok {
		return getToolDescription(toolName, locale)
	}

	// Extract key information based on tool type
	var detail string
	switch toolName {
	case "Bash", "Shell":
		// Show the command being executed
		if cmd, ok := input["command"].(string); ok && cmd != "" {
			// Truncate long commands
			if len(cmd) > 50 {
				cmd = cmd[:47] + "..."
			}
			detail = cmd
		}

	case "Read":
		// Show the file being read
		if path, ok := input["path"].(string); ok && path != "" {
			detail = filepath.Base(path)
		}

	case "Write":
		// Show the file being written
		// Note: Claude CLI uses "file_path" for Write tool, not "path"
		if path, ok := input["file_path"].(string); ok && path != "" {
			detail = filepath.Base(path)
		} else if path, ok := input["path"].(string); ok && path != "" {
			detail = filepath.Base(path)
		}

	case "Edit", "StrReplace":
		// Show the file being edited
		if path, ok := input["path"].(string); ok && path != "" {
			detail = filepath.Base(path)
		}

	case "Glob":
		// Show the glob pattern
		if pattern, ok := input["glob_pattern"].(string); ok && pattern != "" {
			detail = pattern
		} else if pattern, ok := input["pattern"].(string); ok && pattern != "" {
			detail = pattern
		}

	case "Grep":
		// Show the search pattern
		if pattern, ok := input["pattern"].(string); ok && pattern != "" {
			if len(pattern) > 30 {
				pattern = pattern[:27] + "..."
			}
			detail = pattern
		}

	case "LS":
		// Show the directory
		if path, ok := input["target_directory"].(string); ok && path != "" {
			detail = filepath.Base(path)
		} else if path, ok := input["path"].(string); ok && path != "" {
			detail = filepath.Base(path)
		}

	case "WebSearch":
		// Show the search query
		if query, ok := input["search_term"].(string); ok && query != "" {
			if len(query) > 40 {
				query = query[:37] + "..."
			}
			detail = query
		} else if query, ok := input["query"].(string); ok && query != "" {
			if len(query) > 40 {
				query = query[:37] + "..."
			}
			detail = query
		}

	case "WebFetch":
		// Show the URL
		if url, ok := input["url"].(string); ok && url != "" {
			// Extract domain from URL
			if len(url) > 50 {
				url = url[:47] + "..."
			}
			detail = url
		}

	case "Task":
		// Show the task description
		if desc, ok := input["description"].(string); ok && desc != "" {
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			detail = desc
		}
	}

	// Build the message with detail
	baseMsg := getToolDescription(toolName, locale)
	if detail != "" {
		return baseMsg + ": " + detail
	}
	return baseMsg
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

// GetSandboxID returns the sandbox ID (userID-chatID)
func (e *Executor) GetSandboxID() string {
	if e.opts == nil {
		return ""
	}
	return fmt.Sprintf("%s-%s", e.opts.UserID, e.opts.ChatID)
}

// GetVNCUrl returns the VNC preview URL path
// Returns empty string if VNC is not enabled for this sandbox image
func (e *Executor) GetVNCUrl() string {
	if e.opts == nil {
		return ""
	}

	imageName := e.opts.Image
	if imageName == "" {
		return ""
	}

	// Check if the image supports VNC using the shared keyword list in sandbox package
	if !infraSandbox.IsVNCImage(imageName) {
		return ""
	}

	// Return only the sandbox ID, the full URL is constructed by openapi/sandbox.GetVNCClientURL()
	return e.GetSandboxID()
}

// Close releases the executor resources and removes the container
// Note: IPC session is managed by sandbox.Manager.Remove()
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
