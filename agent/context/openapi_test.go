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

	// Parse request first
	completionReq, _ := parseCompletionRequestData(c)

	result, err := GetMessages(c, completionReq)
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

	result, err := GetMessages(c, nil)
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

	completionReq, _ := parseCompletionRequestData(c)

	_, err := GetMessages(c, completionReq)
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

	chatID, err := GetChatID(c, cache, nil)
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

	chatID, err := GetChatID(c, cache, nil)
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
		"metadata": map[string]string{
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

	chatID, err := GetChatID(c, cache, completionReq)
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

	chatID1, err := GetChatID(c, cache, completionReq1)
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

	chatID2, err := GetChatID(c2, cache, completionReq2)
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

	messages := []Message{
		{
			Role:    RoleUser,
			Content: "This should not be used",
		},
	}

	requestBody := map[string]interface{}{
		"model":    "gpt-4",
		"messages": messages,
		"metadata": map[string]string{
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

	chatID, err := GetChatID(c, cache, completionReq)
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

	locale := GetLocale(c, nil)
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

	locale := GetLocale(c, nil)
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

	completionReq := &CompletionRequest{
		Metadata: map[string]string{
			"locale": "ja-JP",
		},
	}

	locale := GetLocale(c, completionReq)
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

	completionReq := &CompletionRequest{
		Metadata: map[string]string{
			"locale": "de-DE",
		},
	}

	locale := GetLocale(c, completionReq)
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

	theme := GetTheme(c, nil)
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

	theme := GetTheme(c, nil)
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

	completionReq := &CompletionRequest{
		Metadata: map[string]string{
			"theme": "auto",
		},
	}

	theme := GetTheme(c, completionReq)
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

	completionReq := &CompletionRequest{
		Metadata: map[string]string{
			"referer": "tool",
		},
	}

	referer := GetReferer(c, completionReq)
	if referer != RefererTool {
		t.Errorf("Expected referer 'tool' from metadata, got '%s'", referer)
	}
}

func TestGetAccept_FromMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &CompletionRequest{
		Metadata: map[string]string{
			"accept": "cui-native",
		},
	}

	accept := GetAccept(c, completionReq)
	if accept != AccepNativeCUI {
		t.Errorf("Expected accept 'cui-native' from metadata, got '%s'", accept)
	}
}

func TestGetAssistantID_FromModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &CompletionRequest{
		Model: "gpt-4-turbo-yao_myassistant",
	}

	assistantID, err := GetAssistantID(c, completionReq)
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

	completionReq := &CompletionRequest{
		Model: "gpt-4-yao_from_model",
	}

	assistantID, err := GetAssistantID(c, completionReq)
	if err != nil {
		t.Fatalf("Failed to get assistant ID: %v", err)
	}

	if assistantID != "from_query" {
		t.Errorf("Expected query parameter to take priority, got '%s'", assistantID)
	}
}

func TestGetRoute_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?yao_route=/dashboard/home", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	route := GetRoute(c, nil)
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

	route := GetRoute(c, nil)
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

	completionReq := &CompletionRequest{
		Route: "/admin/users",
	}

	route := GetRoute(c, completionReq)
	if route != "/admin/users" {
		t.Errorf("Expected route '/admin/users' from payload, got '%s'", route)
	}
}

func TestGetRoute_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?yao_route=/from/query", nil)
	req.Header.Set("X-Yao-Route", "/from/header")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &CompletionRequest{
		Route: "/from/payload",
	}

	route := GetRoute(c, completionReq)
	if route != "/from/query" {
		t.Errorf("Expected query parameter to take priority, got '%s'", route)
	}
}

func TestGetData_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	data := map[string]interface{}{
		"key1": "value1",
		"key2": float64(123),
	}
	dataJSON, _ := json.Marshal(data)

	req := httptest.NewRequest("GET", "/chat/completions?yao_data="+string(dataJSON), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetData(c, nil)
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

func TestGetData_FromHeader_Base64(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dataBase64 := "eyJ1c2VyX2lkIjo0NTYsImFjdGlvbiI6ImNyZWF0ZSJ9" // base64 of {"user_id":456,"action":"create"}

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Data", dataBase64)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetData(c, nil)
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

func TestGetData_FromPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	data := map[string]interface{}{
		"page":  float64(1),
		"limit": float64(10),
	}

	completionReq := &CompletionRequest{
		Data: data,
	}

	result := GetData(c, completionReq)
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

func TestGetData_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	queryData := map[string]interface{}{
		"source": "query",
	}
	queryDataJSON, _ := json.Marshal(queryData)

	headerDataBase64 := "eyJzb3VyY2UiOiJoZWFkZXIifQ==" // base64 of {"source":"header"}

	req := httptest.NewRequest("GET", "/chat/completions?yao_data="+string(queryDataJSON), nil)
	req.Header.Set("X-Yao-Data", headerDataBase64)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	payloadData := map[string]interface{}{
		"source": "payload",
	}

	completionReq := &CompletionRequest{
		Data: payloadData,
	}

	result := GetData(c, completionReq)
	if result == nil {
		t.Fatal("Expected data to be returned")
	}

	if result["source"] != "query" {
		t.Errorf("Expected query parameter to take priority, got '%v'", result["source"])
	}
}

func TestGetData_EmptyData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetData(c, nil)
	if result != nil {
		t.Errorf("Expected nil data, got '%v'", result)
	}
}
