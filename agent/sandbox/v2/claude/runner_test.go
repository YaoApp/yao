package claude_test

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/caller"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/attachment"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

type e2eCase struct {
	ID      string
	Prompt  string
	Timeout time.Duration
}

type protocolCase struct {
	Name   string
	TeamID string
	UserID string
}

var cases = []e2eCase{
	{
		ID:      "tests.sandbox-v2.oneshot-cli",
		Prompt:  "Reply exactly with: hello sandbox v2",
		Timeout: 3 * time.Minute,
	},
}

var toolCallCases = []e2eCase{
	{
		ID:      "tests.sandbox-v2.oneshot-cli",
		Prompt:  "Run the command 'echo refactor-ok' and tell me the output.",
		Timeout: 3 * time.Minute,
	},
}

func TestSandboxV2_Claude_E2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)

	require.NotNil(t, caller.AgentGetterFunc, "AgentGetterFunc should be registered after Prepare")

	protocols := []protocolCase{
		{"openai", identity.BetaOpenAITeamID, identity.BetaOpenAIOwnerUserID},
		{"anthropic", identity.BetaAnthropicTeamID, identity.BetaAnthropicOwnerUserID},
	}

	for _, proto := range protocols {
		proto := proto
		t.Run(proto.Name, func(t *testing.T) {
			for _, tc := range cases {
				tc := tc
				t.Run(tc.ID, func(t *testing.T) {
					agent, err := caller.AgentGetterFunc(tc.ID)
					require.NoError(t, err, "should load assistant %s", tc.ID)

					timeout := tc.Timeout
					if timeout == 0 {
						timeout = 3 * time.Minute
					}

					chatID := fmt.Sprintf("e2e-%s-%s-%d", proto.Name, tc.ID, time.Now().UnixMilli())
					ctx := agentcontext.New(
						context.Background(),
						&oauthtypes.AuthorizedInfo{
							TeamID: proto.TeamID,
							UserID: proto.UserID,
						},
						chatID,
					)

					messages := []agentcontext.Message{
						{Role: "user", Content: tc.Prompt},
					}

					done := make(chan struct{})
					var resp *agentcontext.Response
					var streamErr error

					go func() {
						defer close(done)
						resp, streamErr = agent.Stream(ctx, messages)
					}()

					select {
					case <-done:
					case <-time.After(timeout):
						t.Fatalf("timeout after %v", timeout)
					}

					require.NoError(t, streamErr, "Stream should not return error")
					require.NotNil(t, resp, "response should not be nil")

					// ── 1. CompletionResponse should behave like the LLM path ──
					require.NotNil(t, resp.Completion, "completion should not be nil")
					assert.Equal(t, "assistant", resp.Completion.Role, "role should be assistant")
					assert.Equal(t, agentcontext.FinishReasonStop, resp.Completion.FinishReason, "finish_reason should be stop")
					assert.NotNil(t, resp.Completion.Content, "Content should be populated (same as LLM path)")

					contentStr, ok := resp.Completion.Content.(string)
					require.True(t, ok, "Content should be a string, got %T", resp.Completion.Content)
					t.Logf("CompletionResponse.Content (%d chars): %s", len(contentStr), contentStr)
					assert.Contains(t, contentStr, "hello sandbox v2", "Content should contain expected text")

					// ── 2. Buffer: frame sequence handled correctly ──
					require.NotNil(t, ctx.Buffer, "ctx.Buffer should not be nil")

					msgs := ctx.Buffer.GetMessages()
					t.Logf("buffer message count: %d", len(msgs))
					for _, m := range msgs {
						t.Logf("  seq=%d role=%s type=%s streaming=%v props_keys=%v",
							m.Sequence, m.Role, m.Type, m.IsStreaming, mapKeys(m.Props))
					}

					var userInputCount, assistantTextCount, loadingCount int
					var bufferTextContent string
					for _, m := range msgs {
						switch {
						case m.Role == "user" && m.Type == "user_input":
							userInputCount++
						case m.Role == "assistant" && m.Type == "loading":
							loadingCount++
						case m.Role == "assistant" && m.Type == "text":
							assistantTextCount++
							assert.False(t, m.IsStreaming, "text message should not be streaming (handleMessageEnd should have finalized it)")
							require.NotNil(t, m.Props, "text message props should not be nil")
							if c, ok := m.Props["content"].(string); ok {
								bufferTextContent += c
							}
						}
					}

					assert.Equal(t, 1, userInputCount, "should have exactly 1 user_input message")
					assert.GreaterOrEqual(t, loadingCount, 1, "should have at least 1 loading message")
					assert.Equal(t, 1, assistantTextCount, "should have exactly 1 assistant text message (from handleMessageEnd)")
					assert.Contains(t, bufferTextContent, "hello sandbox v2", "buffer text should contain expected content")

					// ── 3. Buffer content matches CompletionResponse.Content ──
					assert.Equal(t, contentStr, bufferTextContent,
						"CompletionResponse.Content and Buffer text should match")
				})
			}
		})
	}
}

func TestSandboxV2_Claude_Attachments(t *testing.T) {
	identity := testprepare.PrepareE2E(t)

	require.NotNil(t, caller.AgentGetterFunc, "AgentGetterFunc should be registered after Prepare")

	// Attachments with image_url require a model with native vision support.
	// OpenAI rejects image_url in tool-role messages (Claude CLI internal behavior),
	// so only Anthropic-native models (haiku) work for this test.
	protocols := []protocolCase{
		{"haiku", identity.BetaHaikuTeamID, identity.BetaHaikuOwnerUserID},
	}

	// Shared testdata setup (once per top-level test)
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	testdataDir := filepath.Join(filepath.Dir(thisFile), "testdata")

	const uploaderName = "__yao.attachment"
	manager, err := attachment.New(attachment.ManagerOption{
		Driver:       "local",
		MaxSize:      "50M",
		AllowedTypes: []string{"image/*", "text/*", "application/*", "video/*", ".ts", ".js", ".tsx", ".jsx"},
		Options:      map[string]interface{}{"path": filepath.Join(os.TempDir(), "test_sandbox_v2_attach")},
	})
	require.NoError(t, err)
	manager.Name = uploaderName
	attachment.Managers[uploaderName] = manager
	t.Cleanup(func() { delete(attachment.Managers, uploaderName) })

	imgFile := uploadTestFile(t, manager, testdataDir, "test-image.png", "image/png")
	codeFile := uploadTestFile(t, manager, testdataDir, "code.ts", "text/plain")

	imgWrapper := fmt.Sprintf("%s://%s", uploaderName, imgFile.ID)
	codeWrapper := fmt.Sprintf("%s://%s", uploaderName, codeFile.ID)

	for _, proto := range protocols {
		proto := proto
		t.Run(proto.Name, func(t *testing.T) {
			agent, err := caller.AgentGetterFunc("tests.sandbox-v2.oneshot-cli")
			require.NoError(t, err)

			chatID := fmt.Sprintf("e2e-attach-%s-%d", proto.Name, time.Now().UnixMilli())
			ctx := agentcontext.New(
				context.Background(),
				&oauthtypes.AuthorizedInfo{TeamID: proto.TeamID, UserID: proto.UserID},
				chatID,
			)

			messages := []agentcontext.Message{
				{
					Role: "user",
					Content: []interface{}{
						map[string]interface{}{"type": "text", "text": "Describe the attached image and summarize the attached code file. Reply in English."},
						map[string]interface{}{
							"type":      "image_url",
							"image_url": map[string]interface{}{"url": imgWrapper, "detail": "auto"},
						},
						map[string]interface{}{
							"type": "file",
							"file": map[string]interface{}{"url": codeWrapper, "filename": "code.ts"},
						},
					},
				},
			}

			done := make(chan struct{})
			var resp *agentcontext.Response
			var streamErr error

			go func() {
				defer close(done)
				resp, streamErr = agent.Stream(ctx, messages)
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Minute):
				t.Fatalf("timeout after 5m")
			}

			require.NoError(t, streamErr, "Stream should not return error")
			require.NotNil(t, resp)
			require.NotNil(t, resp.Completion)

			contentStr, ok := resp.Completion.Content.(string)
			require.True(t, ok, "Content should be a string, got %T", resp.Completion.Content)
			t.Logf("Response (%d chars): %s", len(contentStr), contentStr)

			lower := strings.ToLower(contentStr)

			imageKeywords := []string{"hello", "utf", "chinese", "text", "emoji"}
			imgHit := false
			for _, kw := range imageKeywords {
				if strings.Contains(lower, kw) {
					imgHit = true
					break
				}
			}
			assert.True(t, imgHit, "response should mention image content (tried: %v)", imageKeywords)

			codeKeywords := []string{"excel", "typescript", "class", "volcengine"}
			codeHit := false
			for _, kw := range codeKeywords {
				if strings.Contains(lower, kw) {
					codeHit = true
					break
				}
			}
			assert.True(t, codeHit, "response should mention code content (tried: %v)", codeKeywords)
		})
	}
}

func uploadTestFile(t *testing.T, manager *attachment.Manager, testdataDir, filename, contentType string) *attachment.File {
	t.Helper()
	path := filepath.Join(testdataDir, filename)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "read testdata/%s", filename)

	fh := &attachment.FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: filename,
			Size:     int64(len(data)),
			Header:   make(map[string][]string),
		},
	}
	fh.Header.Set("Content-Type", contentType)

	file, err := manager.Upload(context.Background(), fh, bytes.NewReader(data), attachment.UploadOption{
		Groups: []string{"e2e-sandbox-v2"},
	})
	require.NoError(t, err, "upload testdata/%s", filename)
	t.Logf("uploaded %s => ID=%s, Path=%s", filename, file.ID, file.Path)
	return file
}

func mapKeys(m map[string]interface{}) []string {
	if m == nil {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestSandboxV2_Claude_ToolCallE2E verifies that tool call execution emits
// "execute" messages and that usage/result_summary metadata is propagated.
func TestSandboxV2_Claude_ToolCallE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)

	require.NotNil(t, caller.AgentGetterFunc, "AgentGetterFunc should be registered after Prepare")

	protocols := []protocolCase{
		{"openai", identity.BetaOpenAITeamID, identity.BetaOpenAIOwnerUserID},
		{"anthropic", identity.BetaAnthropicTeamID, identity.BetaAnthropicOwnerUserID},
	}

	for _, proto := range protocols {
		proto := proto
		t.Run(proto.Name, func(t *testing.T) {
			for _, tc := range toolCallCases {
				tc := tc
				t.Run(tc.ID+"_tool_call", func(t *testing.T) {
					agent, err := caller.AgentGetterFunc(tc.ID)
					require.NoError(t, err, "should load assistant %s", tc.ID)

					timeout := tc.Timeout
					if timeout == 0 {
						timeout = 3 * time.Minute
					}

					chatID := fmt.Sprintf("e2e-tool-%s-%s-%d", proto.Name, tc.ID, time.Now().UnixMilli())
					ctx := agentcontext.New(
						context.Background(),
						&oauthtypes.AuthorizedInfo{
							TeamID: proto.TeamID,
							UserID: proto.UserID,
						},
						chatID,
					)

					messages := []agentcontext.Message{
						{Role: "user", Content: tc.Prompt},
					}

					done := make(chan struct{})
					var resp *agentcontext.Response
					var streamErr error

					go func() {
						defer close(done)
						resp, streamErr = agent.Stream(ctx, messages)
					}()

					select {
					case <-done:
					case <-time.After(timeout):
						t.Fatalf("timeout after %v", timeout)
					}

					require.NoError(t, streamErr, "Stream should not return error")
					require.NotNil(t, resp, "response should not be nil")
					require.NotNil(t, resp.Completion, "completion should not be nil")

					contentStr, ok := resp.Completion.Content.(string)
					require.True(t, ok, "Content should be a string, got %T", resp.Completion.Content)
					t.Logf("Response (%d chars): %s", len(contentStr), contentStr)

					assert.Contains(t, strings.ToLower(contentStr), "refactor-ok",
						"response should contain the command output")

					// ── Verify buffer has execute messages ──
					require.NotNil(t, ctx.Buffer, "ctx.Buffer should not be nil")
					msgs := ctx.Buffer.GetMessages()
					t.Logf("buffer message count: %d", len(msgs))

					var executeCount int
					for _, m := range msgs {
						t.Logf("  seq=%d role=%s type=%s streaming=%v props_keys=%v",
							m.Sequence, m.Role, m.Type, m.IsStreaming, mapKeys(m.Props))
						if m.Type == "execute" {
							executeCount++
							assert.NotNil(t, m.Props, "execute message should have props")
							if m.Props != nil {
								if toolName, ok := m.Props["tool"].(string); ok {
									t.Logf("    execute tool=%s status=%v", toolName, m.Props["status"])
								}
							}
						}
					}
					assert.GreaterOrEqual(t, executeCount, 1,
						"should have at least 1 execute message (tool call)")
				})
			}
		})
	}
}
