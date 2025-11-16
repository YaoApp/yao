package context

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestGetCompletionRequest(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

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
		expectedAccept      Accept
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
			expectedReferer:     RefererProcess,
			expectedAccept:      AcceptWebCUI,
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
			expectedReferer:     RefererAPI,
			expectedAccept:      AcceptStandard,
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
			expectedReferer:     RefererMCP,
			expectedAccept:      AcceptDesktopCUI,
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
			expectedReferer:     RefererAPI,
			expectedAccept:      AcceptStandard,
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

			// Build request
			bodyBytes, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "http://example.com/chat/completions", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add query params
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			// Add headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			c.Request = req

			// Call GetCompletionRequest
			completionReq, ctx, err := GetCompletionRequest(c, cache)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, completionReq)
			assert.NotNil(t, ctx)

			// Verify CompletionRequest
			assert.Equal(t, tt.expectedModel, completionReq.Model)
			assert.Equal(t, tt.expectedMsgCount, len(completionReq.Messages))
			if tt.expectedTemp != nil {
				assert.NotNil(t, completionReq.Temperature)
				assert.Equal(t, *tt.expectedTemp, *completionReq.Temperature)
			}
			if tt.expectedStream != nil {
				assert.NotNil(t, completionReq.Stream)
				assert.Equal(t, *tt.expectedStream, *completionReq.Stream)
			}

			// Verify Context
			assert.Equal(t, tt.expectedLocale, ctx.Locale)
			assert.Equal(t, tt.expectedTheme, ctx.Theme)
			assert.Equal(t, tt.expectedReferer, ctx.Referer)
			assert.Equal(t, tt.expectedAccept, ctx.Accept)
			assert.Equal(t, tt.expectedAssistantID, ctx.AssistantID)
			assert.NotNil(t, ctx.Space)
			assert.NotNil(t, ctx.Cache)
		})
	}
}

func TestParseClientType(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  string
	}{
		{"Empty user agent", "", "web"},
		{"Standard web browser", "Mozilla/5.0", "web"},
		{"Android", "Mozilla/5.0 (Linux; Android 10)", "android"},
		{"iPhone", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0)", "ios"},
		{"iPad", "Mozilla/5.0 (iPad; CPU OS 14_0)", "ios"},
		{"Windows", "Mozilla/5.0 (Windows NT 10.0)", "windows"},
		{"macOS", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)", "macos"},
		{"Linux", "Mozilla/5.0 (X11; Linux x86_64)", "linux"},
		{"Yao Agent", "Yao-Agent/1.0", "agent"},
		{"JSSDK", "Yao-JSSDK/2.0", "jssdk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getClientType(tt.userAgent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseAccept(t *testing.T) {
	tests := []struct {
		name       string
		clientType string
		expected   Accept
	}{
		{"Web client", "web", AcceptWebCUI},
		{"Android client", "android", AccepNativeCUI},
		{"iOS client", "ios", AccepNativeCUI},
		{"Windows client", "windows", AcceptDesktopCUI},
		{"macOS client", "macos", AcceptDesktopCUI},
		{"Linux client", "linux", AcceptDesktopCUI},
		{"Agent client", "agent", AcceptStandard},
		{"JSSDK client", "jssdk", AcceptStandard},
		{"Unknown client", "unknown", AcceptStandard},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAccept(tt.clientType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateAccept(t *testing.T) {
	tests := []struct {
		name     string
		accept   string
		expected Accept
	}{
		{"Valid standard", "standard", AcceptStandard},
		{"Valid cui-web", "cui-web", AcceptWebCUI},
		{"Valid cui-native", "cui-native", AccepNativeCUI},
		{"Valid cui-desktop", "cui-desktop", AcceptDesktopCUI},
		{"Invalid value", "invalid", AcceptStandard},
		{"Empty string", "", AcceptStandard},
		{"Random string", "random-accept", AcceptStandard},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateAccept(tt.accept)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateReferer(t *testing.T) {
	tests := []struct {
		name     string
		referer  string
		expected string
	}{
		{"Valid api", "api", RefererAPI},
		{"Valid process", "process", RefererProcess},
		{"Valid mcp", "mcp", RefererMCP},
		{"Valid jssdk", "jssdk", RefererJSSDK},
		{"Valid agent", "agent", RefererAgent},
		{"Valid tool", "tool", RefererTool},
		{"Valid hook", "hook", RefererHook},
		{"Valid schedule", "schedule", RefererSchedule},
		{"Valid script", "script", RefererScript},
		{"Valid internal", "internal", RefererInternal},
		{"Invalid value", "invalid", RefererAPI},
		{"Empty string", "", RefererAPI},
		{"Random string", "random-referer", RefererAPI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateReferer(tt.referer)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions
func floatPtr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}
