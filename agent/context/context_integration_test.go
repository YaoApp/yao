//go:build integration

package context_test

import (
	"bytes"
	stdContext "context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/store"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestGetCompletionRequest(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	require.NoError(t, err)

	tests := []struct {
		name                string
		requestBody         map[string]interface{}
		queryParams         map[string]string
		headers             map[string]string
		expectedModel       string
		expectedMsgCount    int
		expectedTemp        *float64
		expectedStream      *bool
		expectedLocale      string
		expectedTheme       string
		expectedReferer     string
		expectedAccept      agentctx.Accept
		expectedAssistantID string
		expectError         bool
	}{
		{
			name: "Complete request from body with metadata",
			requestBody: map[string]interface{}{
				"model": "gpt-4-yao_assistant123",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "Hello"},
				},
				"temperature": 0.7,
				"stream":      true,
				"metadata": map[string]string{
					"locale":  "zh-cn",
					"theme":   "dark",
					"referer": "process",
					"accept":  "cui-web",
					"chat_id": "chat-from-metadata",
				},
			},
			expectedModel:       "gpt-4-yao_assistant123",
			expectedMsgCount:    1,
			expectedTemp:        floatPtr(0.7),
			expectedStream:      boolPtr(true),
			expectedLocale:      "zh-cn",
			expectedTheme:       "dark",
			expectedReferer:     agentctx.RefererProcess,
			expectedAccept:      agentctx.AcceptWebCUI,
			expectedAssistantID: "assistant123",
			expectError:         false,
		},
		{
			name: "Query params override payload metadata",
			requestBody: map[string]interface{}{
				"model": "gpt-4-yao_test456",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "Test"},
				},
				"metadata": map[string]string{
					"locale": "en-us",
					"theme":  "light",
				},
			},
			queryParams: map[string]string{
				"locale": "fr-FR",
				"theme":  "auto",
			},
			expectedModel:       "gpt-4-yao_test456",
			expectedMsgCount:    1,
			expectedLocale:      "fr-fr",
			expectedTheme:       "auto",
			expectedReferer:     agentctx.RefererAPI,
			expectedAccept:      agentctx.AcceptStandard,
			expectedAssistantID: "test456",
			expectError:         false,
		},
		{
			name: "Headers override payload metadata",
			requestBody: map[string]interface{}{
				"model": "gpt-3.5-turbo-yao_header789",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "Test"},
				},
				"metadata": map[string]string{
					"referer": "process",
					"accept":  "cui-web",
				},
			},
			headers: map[string]string{
				"X-Yao-Referer": "mcp",
				"X-Yao-Accept":  "cui-desktop",
			},
			expectedModel:       "gpt-3.5-turbo-yao_header789",
			expectedMsgCount:    1,
			expectedLocale:      "",
			expectedTheme:       "",
			expectedReferer:     agentctx.RefererMCP,
			expectedAccept:      agentctx.AcceptDesktopCUI,
			expectedAssistantID: "header789",
			expectError:         false,
		},
		{
			name: "Minimal request without metadata",
			requestBody: map[string]interface{}{
				"model": "gpt-4o-yao_minimal",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "Hello"},
				},
			},
			expectedModel:       "gpt-4o-yao_minimal",
			expectedMsgCount:    1,
			expectedLocale:      "",
			expectedTheme:       "",
			expectedReferer:     agentctx.RefererAPI,
			expectedAccept:      agentctx.AcceptStandard,
			expectedAssistantID: "minimal",
			expectError:         false,
		},
		{
			name: "Missing model",
			requestBody: map[string]interface{}{
				"messages": []map[string]interface{}{
					{"role": "user", "content": "Hello"},
				},
			},
			expectError: true,
		},
		{
			name: "Missing messages",
			requestBody: map[string]interface{}{
				"model": "gpt-4",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "http://example.com/chat/completions", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			c.Request = req

			completionReq, ctx, opts, err := agentctx.GetCompletionRequest(c, cache)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, completionReq)
			require.NotNil(t, ctx)
			require.NotNil(t, opts)
			defer ctx.Release()

			assert.Equal(t, tt.expectedModel, completionReq.Model)
			assert.Equal(t, tt.expectedMsgCount, len(completionReq.Messages))
			if tt.expectedTemp != nil {
				require.NotNil(t, completionReq.Temperature)
				assert.Equal(t, *tt.expectedTemp, *completionReq.Temperature)
			}
			if tt.expectedStream != nil {
				require.NotNil(t, completionReq.Stream)
				assert.Equal(t, *tt.expectedStream, *completionReq.Stream)
			}

			assert.Equal(t, tt.expectedLocale, ctx.Locale)
			assert.Equal(t, tt.expectedTheme, ctx.Theme)
			assert.Equal(t, tt.expectedReferer, ctx.Referer)
			assert.Equal(t, tt.expectedAccept, ctx.Accept)
			assert.Equal(t, tt.expectedAssistantID, ctx.AssistantID)
			assert.NotNil(t, ctx.Memory)
			assert.NotNil(t, ctx.Cache)
		})
	}
}

func TestContextNew_WithAuthorized(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	defer ctx.Release()

	require.NotNil(t, ctx)
	assert.Equal(t, "test-chat-id", ctx.ChatID)
	assert.NotNil(t, ctx.Memory)
	assert.NotNil(t, ctx.IDGenerator)
}

func floatPtr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}
