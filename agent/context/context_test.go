package context

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewGin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name               string
		queryParams        map[string]string
		routeParams        map[string]string
		headers            map[string]string
		expectedChatID     string
		expectedAssistant  string
		expectedLocale     string
		expectedTheme      string
		expectedClientType string
		expectedReferer    string
		expectedAccept     Accept
	}{
		{
			name: "Parse all query parameters",
			queryParams: map[string]string{
				"chat_id": "chat123",
				"locale":  "zh-CN",
				"theme":   "dark",
				"referer": RefererProcess,
				"accept":  string(AcceptStandard),
			},
			routeParams: map[string]string{
				"assistant_id": "ast456",
			},
			headers: map[string]string{
				"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			},
			expectedChatID:     "chat123",
			expectedAssistant:  "ast456",
			expectedLocale:     "zh-CN",
			expectedTheme:      "dark",
			expectedClientType: "macos",
			expectedReferer:    RefererProcess,
			expectedAccept:     AcceptStandard,
		},
		{
			name:        "Default values with no parameters",
			queryParams: map[string]string{},
			headers: map[string]string{
				"User-Agent": "Mozilla/5.0",
			},
			expectedChatID:     "",
			expectedAssistant:  "",
			expectedLocale:     "",
			expectedTheme:      "",
			expectedClientType: "web",
			expectedReferer:    RefererAPI,
			expectedAccept:     AcceptWebCUI,
		},
		{
			name:        "Android client type detection",
			queryParams: map[string]string{},
			headers: map[string]string{
				"User-Agent": "Mozilla/5.0 (Linux; Android 10)",
			},
			expectedClientType: "android",
			expectedReferer:    RefererAPI,
			expectedAccept:     AccepNativeCUI,
		},
		{
			name:        "iOS client type detection",
			queryParams: map[string]string{},
			headers: map[string]string{
				"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0)",
			},
			expectedClientType: "ios",
			expectedReferer:    RefererAPI,
			expectedAccept:     AccepNativeCUI,
		},
		{
			name:        "Windows desktop client type detection",
			queryParams: map[string]string{},
			headers: map[string]string{
				"User-Agent": "Mozilla/5.0 (Windows NT 10.0)",
			},
			expectedClientType: "windows",
			expectedReferer:    RefererAPI,
			expectedAccept:     AcceptDesktopCUI,
		},
		{
			name:        "Agent client type detection",
			queryParams: map[string]string{},
			headers: map[string]string{
				"User-Agent": "Yao-Agent/1.0",
			},
			expectedClientType: "agent",
			expectedReferer:    RefererAPI,
			expectedAccept:     AcceptStandard,
		},
		{
			name:        "JSSDK client type detection",
			queryParams: map[string]string{},
			headers: map[string]string{
				"User-Agent": "Yao-JSSDK/2.0",
			},
			expectedClientType: "jssdk",
			expectedReferer:    RefererAPI,
			expectedAccept:     AcceptStandard,
		},
		{
			name:        "Custom headers for referer and accept",
			queryParams: map[string]string{},
			headers: map[string]string{
				"User-Agent":    "Mozilla/5.0",
				"X-Yao-Referer": RefererMCP,
				"X-Yao-Accept":  string(AcceptDesktopCUI),
			},
			expectedClientType: "web",
			expectedReferer:    RefererMCP,
			expectedAccept:     AcceptDesktopCUI,
		},
		{
			name: "Query parameters override headers",
			queryParams: map[string]string{
				"referer": RefererJSSDK,
				"accept":  string(AcceptStandard),
			},
			headers: map[string]string{
				"User-Agent":    "Mozilla/5.0",
				"X-Yao-Referer": RefererMCP,
				"X-Yao-Accept":  string(AcceptDesktopCUI),
			},
			expectedClientType: "web",
			expectedReferer:    RefererJSSDK,
			expectedAccept:     AcceptStandard,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Build query string
			req, _ := http.NewRequest("GET", "http://example.com/test", nil)
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			c.Request = req

			// Set route params
			for key, value := range tt.routeParams {
				c.Params = append(c.Params, gin.Param{Key: key, Value: value})
			}

			// Call NewGin
			ctx := NewGin(c)

			// Assertions
			assert.Equal(t, tt.expectedChatID, ctx.ChatID, "ChatID mismatch")
			assert.Equal(t, tt.expectedAssistant, ctx.AssistantID, "AssistantID mismatch")
			assert.Equal(t, tt.expectedLocale, ctx.Locale, "Locale mismatch")
			assert.Equal(t, tt.expectedTheme, ctx.Theme, "Theme mismatch")
			assert.Equal(t, tt.expectedClientType, ctx.Client.Type, "Client.Type mismatch")
			assert.Equal(t, tt.expectedReferer, ctx.Referer, "Referer mismatch")
			assert.Equal(t, tt.expectedAccept, ctx.Accept, "Accept mismatch")
			assert.NotNil(t, ctx.Space, "Space should not be nil")
			// Client.UserAgent and Client.IP are set from headers/request, may be empty in test context
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
			result := parseClientType(tt.userAgent)
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
