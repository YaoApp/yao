package context

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestGetMessages_FromBody(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	messages := []Message{
		{
			Role:    RoleUser,
			Content: "Hello, world!",
		},
		{
			Role:    RoleAssistant,
			Content: "Hi there!",
		},
	}

	requestBody := map[string]interface{}{
		"messages": messages,
		"model":    "gpt-4",
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result, err := GetMessages(c)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(result))
	}

	if result[0].Role != RoleUser {
		t.Errorf("Expected first message role to be %s, got %s", RoleUser, result[0].Role)
	}
}

func TestGetMessages_FromQuery(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	messages := []Message{
		{
			Role:    RoleUser,
			Content: "Test message",
		},
	}

	messagesJSON, _ := json.Marshal(messages)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	q := req.URL.Query()
	q.Add("messages", string(messagesJSON))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result, err := GetMessages(c)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}
}

func TestGetMessages_EmptyMessages(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	requestBody := map[string]interface{}{
		"messages": []Message{},
		"model":    "gpt-4",
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	_, err := GetMessages(c)
	if err == nil {
		t.Error("Expected error for empty messages")
	}
}

func TestGetChatID_FromQuery(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	expectedChatID := "test-chat-123"

	req := httptest.NewRequest("GET", "/chat/completions?chat_id="+expectedChatID, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	chatID, err := GetChatID(c, cache)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	if chatID != expectedChatID {
		t.Errorf("Expected chat ID %s, got %s", expectedChatID, chatID)
	}
}

func TestGetChatID_FromHeader(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	expectedChatID := "header-chat-456"

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Chat", expectedChatID)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	chatID, err := GetChatID(c, cache)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	if chatID != expectedChatID {
		t.Errorf("Expected chat ID %s, got %s", expectedChatID, chatID)
	}
}

func TestGetChatID_FromMessages(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}
	cache.Clear()

	// First request with one user message
	messages1 := []Message{
		{
			Role:    RoleUser,
			Content: "First message",
		},
	}

	requestBody1 := map[string]interface{}{
		"messages": messages1,
	}

	bodyBytes1, _ := json.Marshal(requestBody1)

	req := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	chatID1, err := GetChatID(c, cache)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	if chatID1 == "" {
		t.Error("Expected non-empty chat ID")
	}

	// Second request with two user messages (continuation)
	messages2 := []Message{
		{
			Role:    RoleUser,
			Content: "First message",
		},
		{
			Role:    RoleUser,
			Content: "Second message",
		},
	}

	requestBody2 := map[string]interface{}{
		"messages": messages2,
	}

	bodyBytes2, _ := json.Marshal(requestBody2)

	req2 := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = req2

	chatID2, err := GetChatID(c2, cache)
	if err != nil {
		t.Fatalf("Failed to get chat ID second time: %v", err)
	}

	// Should get same chat ID (continuation of conversation)
	if chatID1 != chatID2 {
		t.Errorf("Expected same chat ID for continuation, got %s and %s", chatID1, chatID2)
	}
}

func TestGetChatID_Priority(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	queryChatID := "query-chat-id"
	headerChatID := "header-chat-id"

	messages := []Message{
		{
			Role:    RoleUser,
			Content: "This should not be used",
		},
	}

	requestBody := map[string]interface{}{
		"messages": messages,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	// Test priority: query > header > messages
	req := httptest.NewRequest("POST", "/chat/completions?chat_id="+queryChatID, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Yao-Chat", headerChatID)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	chatID, err := GetChatID(c, cache)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	if chatID != queryChatID {
		t.Errorf("Expected query parameter to take priority, got %s instead of %s", chatID, queryChatID)
	}
}

func TestGetLocale_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?locale=zh-CN", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	locale := GetLocale(c)
	if locale != "zh-cn" {
		t.Errorf("Expected locale 'zh-cn', got '%s'", locale)
	}
}

func TestGetLocale_FromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh;q=0.8")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	locale := GetLocale(c)
	if locale != "en-us" {
		t.Errorf("Expected locale 'en-us', got '%s'", locale)
	}
}

func TestGetLocale_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?locale=fr-FR", nil)
	req.Header.Set("Accept-Language", "en-US")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	locale := GetLocale(c)
	if locale != "fr-fr" {
		t.Errorf("Expected query parameter to take priority, got '%s'", locale)
	}
}

func TestGetLocale_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	locale := GetLocale(c)
	if locale != "" {
		t.Errorf("Expected empty locale, got '%s'", locale)
	}
}

func TestGetTheme_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?theme=dark", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	theme := GetTheme(c)
	if theme != "dark" {
		t.Errorf("Expected theme 'dark', got '%s'", theme)
	}
}

func TestGetTheme_FromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Theme", "light")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	theme := GetTheme(c)
	if theme != "light" {
		t.Errorf("Expected theme 'light', got '%s'", theme)
	}
}

func TestGetTheme_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?theme=auto", nil)
	req.Header.Set("X-Yao-Theme", "dark")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	theme := GetTheme(c)
	if theme != "auto" {
		t.Errorf("Expected query parameter to take priority, got '%s'", theme)
	}
}

func TestGetTheme_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	theme := GetTheme(c)
	if theme != "" {
		t.Errorf("Expected empty theme, got '%s'", theme)
	}
}

func TestGetReferer_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?referer=jssdk", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	referer := GetReferer(c)
	if referer != "jssdk" {
		t.Errorf("Expected referer 'jssdk', got '%s'", referer)
	}
}

func TestGetReferer_FromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Referer", "agent")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	referer := GetReferer(c)
	if referer != "agent" {
		t.Errorf("Expected referer 'agent', got '%s'", referer)
	}
}

func TestGetReferer_Default(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	referer := GetReferer(c)
	if referer != RefererAPI {
		t.Errorf("Expected default referer '%s', got '%s'", RefererAPI, referer)
	}
}

func TestGetReferer_InvalidValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?referer=invalid", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	referer := GetReferer(c)
	if referer != RefererAPI {
		t.Errorf("Expected default referer for invalid value, got '%s'", referer)
	}
}

func TestGetReferer_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?referer=process", nil)
	req.Header.Set("X-Yao-Referer", "tool")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	referer := GetReferer(c)
	if referer != "process" {
		t.Errorf("Expected query parameter to take priority, got '%s'", referer)
	}
}

func TestGetAccept_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?accept=cui-web", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := GetAccept(c)
	if accept != AcceptWebCUI {
		t.Errorf("Expected accept '%s', got '%s'", AcceptWebCUI, accept)
	}
}

func TestGetAccept_FromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Accept", "cui-native")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := GetAccept(c)
	if accept != AccepNativeCUI {
		t.Errorf("Expected accept '%s', got '%s'", AccepNativeCUI, accept)
	}
}

func TestGetAccept_FromUserAgent_Web(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := GetAccept(c)
	if accept != AcceptWebCUI {
		t.Errorf("Expected accept '%s' for web user agent, got '%s'", AcceptWebCUI, accept)
	}
}

func TestGetAccept_FromUserAgent_Android(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("User-Agent", "Android App")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := GetAccept(c)
	if accept != AccepNativeCUI {
		t.Errorf("Expected accept '%s' for Android, got '%s'", AccepNativeCUI, accept)
	}
}

func TestGetAccept_FromUserAgent_Desktop(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("User-Agent", "Windows NT 10.0")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := GetAccept(c)
	if accept != AcceptDesktopCUI {
		t.Errorf("Expected accept '%s' for Windows, got '%s'", AcceptDesktopCUI, accept)
	}
}

func TestGetAccept_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?accept=standard", nil)
	req.Header.Set("X-Yao-Accept", "cui-web")
	req.Header.Set("User-Agent", "Android App")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := GetAccept(c)
	if accept != AcceptStandard {
		t.Errorf("Expected query parameter to take priority, got '%s'", accept)
	}
}

func TestGetAccept_InvalidValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?accept=invalid", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := GetAccept(c)
	if accept != AcceptStandard {
		t.Errorf("Expected default accept for invalid value, got '%s'", accept)
	}
}
