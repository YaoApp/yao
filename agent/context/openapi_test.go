package context_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// parseCompletionRequestData is a helper function for tests to parse completion request data
func parseCompletionRequestData(c *gin.Context) (*context.CompletionRequest, error) {
	var req context.CompletionRequest

	if c.Request.Body != nil {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				return nil, err
			}
			if len(req.Messages) > 0 {
				return &req, nil
			}
		}
	}
	return &req, nil
}

func TestGetMessages_FromBody(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Hello, world!",
		},
		{
			Role:    context.RoleAssistant,
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

	// Parse request first
	completionReq, _ := parseCompletionRequestData(c)

	result, err := context.GetMessages(c, completionReq)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(result))
	}

	if result[0].Role != context.RoleUser {
		t.Errorf("Expected first message role to be %s, got %s", context.RoleUser, result[0].Role)
	}
}

func TestGetMessages_FromQuery(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	messages := []context.Message{
		{
			Role:    context.RoleUser,
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

	result, err := context.GetMessages(c, nil)
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
		"messages": []context.Message{},
		"model":    "gpt-4",
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq, _ := parseCompletionRequestData(c)

	_, err := context.GetMessages(c, completionReq)
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

	chatID, err := context.GetChatID(c, cache, nil)
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

	chatID, err := context.GetChatID(c, cache, nil)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	if chatID != expectedChatID {
		t.Errorf("Expected chat ID %s, got %s", expectedChatID, chatID)
	}
}

func TestGetChatID_FromMetadata(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	expectedChatID := "metadata-chat-789"

	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Test"},
		},
		"metadata": map[string]interface{}{
			"chat_id": expectedChatID,
		},
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq, _ := parseCompletionRequestData(c)

	chatID, err := context.GetChatID(c, cache, completionReq)
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
	messages1 := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "First message",
		},
	}

	requestBody1 := map[string]interface{}{
		"model":    "gpt-4",
		"messages": messages1,
	}

	bodyBytes1, _ := json.Marshal(requestBody1)

	req := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq1, _ := parseCompletionRequestData(c)

	chatID1, err := context.GetChatID(c, cache, completionReq1)
	if err != nil {
		t.Fatalf("Failed to get chat ID: %v", err)
	}

	if chatID1 == "" {
		t.Error("Expected non-empty chat ID")
	}

	// Second request with two user messages (continuation)
	messages2 := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "First message",
		},
		{
			Role:    context.RoleUser,
			Content: "Second message",
		},
	}

	requestBody2 := map[string]interface{}{
		"model":    "gpt-4",
		"messages": messages2,
	}

	bodyBytes2, _ := json.Marshal(requestBody2)

	req2 := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = req2

	completionReq2, _ := parseCompletionRequestData(c2)

	chatID2, err := context.GetChatID(c2, cache, completionReq2)
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
	metadataChatID := "metadata-chat-id"

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "This should not be used",
		},
	}

	requestBody := map[string]interface{}{
		"model":    "gpt-4",
		"messages": messages,
		"metadata": map[string]interface{}{
			"chat_id": metadataChatID,
		},
	}

	bodyBytes, _ := json.Marshal(requestBody)

	// Test priority: query > header > metadata > messages
	req := httptest.NewRequest("POST", "/chat/completions?chat_id="+queryChatID, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Yao-Chat", headerChatID)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq, _ := parseCompletionRequestData(c)

	chatID, err := context.GetChatID(c, cache, completionReq)
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

	locale := context.GetLocale(c, nil)
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

	locale := context.GetLocale(c, nil)
	if locale != "en-us" {
		t.Errorf("Expected locale 'en-us', got '%s'", locale)
	}
}

func TestGetLocale_FromMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Metadata: map[string]interface{}{
			"locale": "ja-JP",
		},
	}

	locale := context.GetLocale(c, completionReq)
	if locale != "ja-jp" {
		t.Errorf("Expected locale 'ja-jp' from metadata, got '%s'", locale)
	}
}

func TestGetLocale_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?locale=fr-FR", nil)
	req.Header.Set("Accept-Language", "en-US")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Metadata: map[string]interface{}{
			"locale": "de-DE",
		},
	}

	locale := context.GetLocale(c, completionReq)
	if locale != "fr-fr" {
		t.Errorf("Expected query parameter to take priority, got '%s'", locale)
	}
}

func TestGetTheme_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?theme=dark", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	theme := context.GetTheme(c, nil)
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

	theme := context.GetTheme(c, nil)
	if theme != "light" {
		t.Errorf("Expected theme 'light', got '%s'", theme)
	}
}

func TestGetTheme_FromMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Metadata: map[string]interface{}{
			"theme": "auto",
		},
	}

	theme := context.GetTheme(c, completionReq)
	if theme != "auto" {
		t.Errorf("Expected theme 'auto' from metadata, got '%s'", theme)
	}
}

func TestGetReferer_FromMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Metadata: map[string]interface{}{
			"referer": "tool",
		},
	}

	referer := context.GetReferer(c, completionReq)
	if referer != context.RefererTool {
		t.Errorf("Expected referer 'tool' from metadata, got '%s'", referer)
	}
}

func TestGetAccept_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?accept=cui-web", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := context.GetAccept(c, nil)
	if accept != context.AcceptWebCUI {
		t.Errorf("Expected accept 'cui-web' from query, got '%s'", accept)
	}
}

func TestGetAccept_FromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Accept", "cui-desktop")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := context.GetAccept(c, nil)
	if accept != context.AcceptDesktopCUI {
		t.Errorf("Expected accept 'cui-desktop' from header, got '%s'", accept)
	}
}

func TestGetAccept_FromMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Metadata: map[string]interface{}{
			"accept": "cui-native",
		},
	}

	accept := context.GetAccept(c, completionReq)
	if accept != context.AccepNativeCUI {
		t.Errorf("Expected accept 'cui-native' from metadata, got '%s'", accept)
	}
}

func TestGetAccept_Default(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := context.GetAccept(c, nil)
	if accept != context.AcceptStandard {
		t.Errorf("Expected default accept 'standard', got '%s'", accept)
	}
}

func TestGetAccept_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?accept=cui-web", nil)
	req.Header.Set("X-Yao-Accept", "cui-desktop")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Metadata: map[string]interface{}{
			"accept": "cui-native",
		},
	}

	accept := context.GetAccept(c, completionReq)
	if accept != context.AcceptWebCUI {
		t.Errorf("Expected query parameter to take priority, got '%s'", accept)
	}
}

func TestGetAssistantID_FromModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Model: "gpt-4-turbo-yao_myassistant",
	}

	assistantID, err := context.GetAssistantID(c, completionReq)
	if err != nil {
		t.Fatalf("Failed to get assistant ID: %v", err)
	}

	if assistantID != "myassistant" {
		t.Errorf("Expected assistant ID 'myassistant', got '%s'", assistantID)
	}
}

func TestGetAssistantID_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?assistant_id=from_query", nil)
	req.Header.Set("X-Yao-Assistant", "from_header")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Model: "gpt-4-yao_from_model",
	}

	assistantID, err := context.GetAssistantID(c, completionReq)
	if err != nil {
		t.Fatalf("Failed to get assistant ID: %v", err)
	}

	if assistantID != "from_query" {
		t.Errorf("Expected query parameter to take priority, got '%s'", assistantID)
	}
}

func TestGetRoute_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?route=/dashboard/home", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	route := context.GetRoute(c, nil)
	if route != "/dashboard/home" {
		t.Errorf("Expected route '/dashboard/home', got '%s'", route)
	}
}

func TestGetRoute_FromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Route", "/settings/profile")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	route := context.GetRoute(c, nil)
	if route != "/settings/profile" {
		t.Errorf("Expected route '/settings/profile', got '%s'", route)
	}
}

func TestGetRoute_FromPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Route: "/admin/users",
	}

	route := context.GetRoute(c, completionReq)
	if route != "/admin/users" {
		t.Errorf("Expected route '/admin/users' from payload, got '%s'", route)
	}
}

func TestGetRoute_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?route=/from/query", nil)
	req.Header.Set("X-Yao-Route", "/from/header")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Route: "/from/payload",
	}

	route := context.GetRoute(c, completionReq)
	if route != "/from/query" {
		t.Errorf("Expected query parameter to take priority, got '%s'", route)
	}
}

func TestGetMetadata_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	data := map[string]interface{}{
		"key1": "value1",
		"key2": float64(123),
	}
	dataJSON, _ := json.Marshal(data)

	req := httptest.NewRequest("GET", "/chat/completions?metadata="+string(dataJSON), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := context.GetMetadata(c, nil)
	if result == nil {
		t.Fatal("Expected data to be returned")
	}

	if result["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got '%v'", result["key1"])
	}

	if result["key2"] != float64(123) {
		t.Errorf("Expected key2=123, got '%v'", result["key2"])
	}
}

func TestGetMetadata_FromHeader_Base64(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dataBase64 := "eyJ1c2VyX2lkIjo0NTYsImFjdGlvbiI6ImNyZWF0ZSJ9" // base64 of {"user_id":456,"action":"create"}

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Metadata", dataBase64)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := context.GetMetadata(c, nil)
	if result == nil {
		t.Fatal("Expected data to be returned")
	}

	if result["action"] != "create" {
		t.Errorf("Expected action='create', got '%v'", result["action"])
	}

	if result["user_id"] != float64(456) {
		t.Errorf("Expected user_id=456, got '%v'", result["user_id"])
	}
}

func TestGetMetadata_FromPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	data := map[string]interface{}{
		"page":  float64(1),
		"limit": float64(10),
	}

	completionReq := &context.CompletionRequest{
		Metadata: data,
	}

	result := context.GetMetadata(c, completionReq)
	if result == nil {
		t.Fatal("Expected data to be returned")
	}

	if result["page"] != float64(1) {
		t.Errorf("Expected page=1, got '%v'", result["page"])
	}

	if result["limit"] != float64(10) {
		t.Errorf("Expected limit=10, got '%v'", result["limit"])
	}
}

func TestGetMetadata_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	queryData := map[string]interface{}{
		"source": "query",
	}
	queryDataJSON, _ := json.Marshal(queryData)

	headerDataBase64 := "eyJzb3VyY2UiOiJoZWFkZXIifQ==" // base64 of {"source":"header"}

	req := httptest.NewRequest("GET", "/chat/completions?metadata="+string(queryDataJSON), nil)
	req.Header.Set("X-Yao-Metadata", headerDataBase64)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	payloadData := map[string]interface{}{
		"source": "payload",
	}

	completionReq := &context.CompletionRequest{
		Metadata: payloadData,
	}

	result := context.GetMetadata(c, completionReq)
	if result == nil {
		t.Fatal("Expected data to be returned")
	}

	if result["source"] != "query" {
		t.Errorf("Expected query parameter to take priority, got '%v'", result["source"])
	}
}

func TestGetMetadata_EmptyData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := context.GetMetadata(c, nil)
	if result != nil {
		t.Errorf("Expected nil data, got '%v'", result)
	}
}

func TestGetCompletionRequest_WriterInitialized(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Test message",
		},
	}

	requestBody := map[string]interface{}{
		"model":    "gpt-4-yao_test",
		"messages": messages,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq, ctx, opts, err := context.GetCompletionRequest(c, cache)
	if err != nil {
		t.Fatalf("Failed to get completion request: %v", err)
	}
	defer ctx.Release()

	// Check that Writer is initialized
	if ctx.Writer == nil {
		t.Error("Expected ctx.Writer to be initialized, got nil")
	}

	// Check that Writer is the same as gin context writer
	if ctx.Writer != c.Writer {
		t.Error("Expected ctx.Writer to be the same as gin context writer")
	}

	// Check that Options is initialized
	if opts == nil {
		t.Error("Expected opts to be initialized, got nil")
	}

	// Check other fields
	if completionReq.Model != "gpt-4-yao_test" {
		t.Errorf("Expected model 'gpt-4-yao_test', got '%s'", completionReq.Model)
	}

	if ctx.AssistantID != "test" {
		t.Errorf("Expected assistant ID 'test', got '%s'", ctx.AssistantID)
	}

	// Check that ChatID was generated (fallback)
	if ctx.ChatID == "" {
		t.Error("Expected ChatID to be generated, got empty string")
	}
}

func TestGetCompletionRequest_ChatIDFallback(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	// Request without explicit chat_id should generate one
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Test message",
		},
	}

	requestBody := map[string]interface{}{
		"model":    "gpt-4-yao_assistant1",
		"messages": messages,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	_, ctx, opts, err := context.GetCompletionRequest(c, cache)
	if err != nil {
		t.Fatalf("Failed to get completion request: %v", err)
	}
	defer ctx.Release()

	// Check that Options is initialized
	if opts == nil {
		t.Error("Expected opts to be initialized, got nil")
	}

	// ChatID should be generated (not empty)
	if ctx.ChatID == "" {
		t.Error("Expected ChatID to be generated via fallback, got empty string")
	}

	// ChatID should be a valid NanoID format (16 characters)
	if len(ctx.ChatID) < 8 {
		t.Errorf("Expected ChatID to be at least 8 characters, got %d", len(ctx.ChatID))
	}
}

func TestGetSkip_FromBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Skip: &context.Skip{
			History: true,
			Trace:   false,
		},
	}

	skip := context.GetSkip(c, completionReq)
	if skip == nil {
		t.Fatal("Expected skip to be returned")
	}

	if !skip.History {
		t.Error("Expected skip.History to be true")
	}

	if skip.Trace {
		t.Error("Expected skip.Trace to be false")
	}
}

func TestGetSkip_FromQueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?skip_history=true&skip_trace=false", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	skip := context.GetSkip(c, nil)
	if skip == nil {
		t.Fatal("Expected skip to be returned")
	}

	if !skip.History {
		t.Error("Expected skip.History to be true from query param")
	}

	if skip.Trace {
		t.Error("Expected skip.Trace to be false")
	}
}

func TestGetSkip_FromQueryParams_ShortForm(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?skip_history=1&skip_trace=1", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	skip := context.GetSkip(c, nil)
	if skip == nil {
		t.Fatal("Expected skip to be returned")
	}

	if !skip.History {
		t.Error("Expected skip.History to be true from query param (1)")
	}

	if !skip.Trace {
		t.Error("Expected skip.Trace to be true from query param (1)")
	}
}

func TestGetSkip_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Body should take priority over query
	req := httptest.NewRequest("POST", "/chat/completions?skip_history=false&skip_trace=false", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Skip: &context.Skip{
			History: true,
			Trace:   true,
		},
	}

	skip := context.GetSkip(c, completionReq)
	if skip == nil {
		t.Fatal("Expected skip to be returned")
	}

	// Body should take priority
	if !skip.History {
		t.Error("Expected body parameter to take priority, skip.History should be true")
	}

	if !skip.Trace {
		t.Error("Expected body parameter to take priority, skip.Trace should be true")
	}
}

func TestGetSkip_Nil(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	skip := context.GetSkip(c, nil)
	if skip != nil {
		t.Errorf("Expected skip to be nil, got %v", skip)
	}
}

func TestGetSkip_OnlyHistorySet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?skip_history=true", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	skip := context.GetSkip(c, nil)
	if skip == nil {
		t.Fatal("Expected skip to be returned")
	}

	if !skip.History {
		t.Error("Expected skip.History to be true")
	}

	if skip.Trace {
		t.Error("Expected skip.Trace to be false (default)")
	}
}

func TestGetSkip_FromBodyViaParseRequest(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	// Test parsing Skip from full request body
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Generate a title for this chat",
		},
	}

	requestBody := map[string]interface{}{
		"model":    "workers.system.title-yao_test",
		"messages": messages,
		"skip": map[string]interface{}{
			"history": true,
			"trace":   false,
		},
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Parse the request
	completionReq, err := parseCompletionRequestData(c)
	if err != nil {
		t.Fatalf("Failed to parse completion request: %v", err)
	}

	// Verify Skip was parsed correctly
	if completionReq.Skip == nil {
		t.Fatal("Expected Skip to be parsed from body, got nil")
	}

	if !completionReq.Skip.History {
		t.Error("Expected Skip.History to be true from body")
	}

	if completionReq.Skip.Trace {
		t.Error("Expected Skip.Trace to be false from body")
	}

	// Now test GetSkip function with the parsed request
	skip := context.GetSkip(c, completionReq)
	if skip == nil {
		t.Fatal("Expected GetSkip to return skip configuration")
	}

	if !skip.History {
		t.Error("Expected GetSkip to return History=true")
	}

	if skip.Trace {
		t.Error("Expected GetSkip to return Trace=false")
	}
}

func TestGetMode_FromQuery(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?mode=task", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	mode := context.GetMode(c, nil)
	if mode != "task" {
		t.Errorf("Expected mode 'task' from query, got '%s'", mode)
	}
}

func TestGetMode_FromHeader(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Mode", "chat")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	mode := context.GetMode(c, nil)
	if mode != "chat" {
		t.Errorf("Expected mode 'chat' from header, got '%s'", mode)
	}
}

func TestGetMode_FromMetadata(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Metadata: map[string]interface{}{
			"mode": "task",
		},
	}

	mode := context.GetMode(c, completionReq)
	if mode != "task" {
		t.Errorf("Expected mode 'task' from metadata, got '%s'", mode)
	}
}

func TestGetMode_Priority(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	// Query has highest priority
	req := httptest.NewRequest("GET", "/chat/completions?mode=query_mode", nil)
	req.Header.Set("X-Yao-Mode", "header_mode")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &context.CompletionRequest{
		Metadata: map[string]interface{}{
			"mode": "metadata_mode",
		},
	}

	mode := context.GetMode(c, completionReq)
	if mode != "query_mode" {
		t.Errorf("Expected mode 'query_mode' (query has priority), got '%s'", mode)
	}
}

func TestGetMode_Empty(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	mode := context.GetMode(c, nil)
	if mode != "" {
		t.Errorf("Expected empty mode, got '%s'", mode)
	}
}
