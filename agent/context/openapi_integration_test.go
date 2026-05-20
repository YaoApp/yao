//go:build integration

package context_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/store"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func parseCompletionRequestData(c *gin.Context) (*agentctx.CompletionRequest, error) {
	var req agentctx.CompletionRequest

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

// ==================== GetMessages ====================

func TestGetMessages_FromBody(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	messages := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "Hello, world!"},
		{Role: agentctx.RoleAssistant, Content: "Hi there!"},
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

	completionReq, _ := parseCompletionRequestData(c)

	result, err := agentctx.GetMessages(c, completionReq)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, agentctx.RoleUser, result[0].Role)
}

func TestGetMessages_FromQuery(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	messages := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "Test message"},
	}

	messagesJSON, _ := json.Marshal(messages)
	req := httptest.NewRequest("GET", "/chat/completions", nil)
	q := req.URL.Query()
	q.Add("messages", string(messagesJSON))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result, err := agentctx.GetMessages(c, nil)
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestGetMessages_EmptyMessages(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	requestBody := map[string]interface{}{
		"messages": []agentctx.Message{},
		"model":    "gpt-4",
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq, _ := parseCompletionRequestData(c)

	_, err := agentctx.GetMessages(c, completionReq)
	assert.Error(t, err)
}

// ==================== GetChatID ====================

func TestGetChatID_FromQuery(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	require.NoError(t, err)

	expectedChatID := "test-chat-123"
	req := httptest.NewRequest("GET", "/chat/completions?chat_id="+expectedChatID, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	chatID, err := agentctx.GetChatID(c, cache, nil)
	require.NoError(t, err)
	assert.Equal(t, expectedChatID, chatID)
}

func TestGetChatID_FromHeader(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	require.NoError(t, err)

	expectedChatID := "header-chat-456"
	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Chat", expectedChatID)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	chatID, err := agentctx.GetChatID(c, cache, nil)
	require.NoError(t, err)
	assert.Equal(t, expectedChatID, chatID)
}

func TestGetChatID_FromMetadata(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	require.NoError(t, err)

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

	chatID, err := agentctx.GetChatID(c, cache, completionReq)
	require.NoError(t, err)
	assert.Equal(t, expectedChatID, chatID)
}

func TestGetChatID_FromMessages(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	require.NoError(t, err)
	cache.Clear()

	messages1 := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "First message"},
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

	chatID1, err := agentctx.GetChatID(c, cache, completionReq1)
	require.NoError(t, err)
	assert.NotEmpty(t, chatID1)

	messages2 := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "First message"},
		{Role: agentctx.RoleUser, Content: "Second message"},
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

	chatID2, err := agentctx.GetChatID(c2, cache, completionReq2)
	require.NoError(t, err)
	assert.Equal(t, chatID1, chatID2, "Continuation should have same chat ID")
}

func TestGetChatID_Priority(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	require.NoError(t, err)

	queryChatID := "query-chat-id"
	headerChatID := "header-chat-id"
	metadataChatID := "metadata-chat-id"

	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "This should not be used"},
		},
		"metadata": map[string]interface{}{
			"chat_id": metadataChatID,
		},
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/chat/completions?chat_id="+queryChatID, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Yao-Chat", headerChatID)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq, _ := parseCompletionRequestData(c)

	chatID, err := agentctx.GetChatID(c, cache, completionReq)
	require.NoError(t, err)
	assert.Equal(t, queryChatID, chatID, "Query parameter should take priority")
}

// ==================== GetLocale ====================

func TestGetLocale_FromQuery(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?locale=zh-CN", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	locale := agentctx.GetLocale(c, nil)
	assert.Equal(t, "zh-cn", locale)
}

func TestGetLocale_FromHeader(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh;q=0.8")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	locale := agentctx.GetLocale(c, nil)
	assert.Equal(t, "en-us", locale)
}

func TestGetLocale_FromMetadata(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Metadata: map[string]interface{}{
			"locale": "ja-JP",
		},
	}

	locale := agentctx.GetLocale(c, completionReq)
	assert.Equal(t, "ja-jp", locale)
}

func TestGetLocale_Priority(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?locale=fr-FR", nil)
	req.Header.Set("Accept-Language", "en-US")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Metadata: map[string]interface{}{
			"locale": "de-DE",
		},
	}

	locale := agentctx.GetLocale(c, completionReq)
	assert.Equal(t, "fr-fr", locale, "Query parameter should take priority")
}

// ==================== GetTheme ====================

func TestGetTheme_FromQuery(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?theme=dark", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	theme := agentctx.GetTheme(c, nil)
	assert.Equal(t, "dark", theme)
}

func TestGetTheme_FromHeader(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Theme", "light")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	theme := agentctx.GetTheme(c, nil)
	assert.Equal(t, "light", theme)
}

func TestGetTheme_FromMetadata(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Metadata: map[string]interface{}{
			"theme": "auto",
		},
	}

	theme := agentctx.GetTheme(c, completionReq)
	assert.Equal(t, "auto", theme)
}

// ==================== GetReferer ====================

func TestGetReferer_FromMetadata(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Metadata: map[string]interface{}{
			"referer": "tool",
		},
	}

	referer := agentctx.GetReferer(c, completionReq)
	assert.Equal(t, agentctx.RefererTool, referer)
}

// ==================== GetAccept ====================

func TestGetAccept_FromQuery(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?accept=cui-web", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := agentctx.GetAccept(c, nil)
	assert.Equal(t, agentctx.Accept(agentctx.AcceptWebCUI), accept)
}

func TestGetAccept_FromHeader(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Accept", "cui-desktop")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := agentctx.GetAccept(c, nil)
	assert.Equal(t, agentctx.Accept(agentctx.AcceptDesktopCUI), accept)
}

func TestGetAccept_FromMetadata(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Metadata: map[string]interface{}{
			"accept": "cui-native",
		},
	}

	accept := agentctx.GetAccept(c, completionReq)
	assert.Equal(t, agentctx.Accept(agentctx.AccepNativeCUI), accept)
}

func TestGetAccept_Default(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	accept := agentctx.GetAccept(c, nil)
	assert.Equal(t, agentctx.Accept(agentctx.AcceptStandard), accept)
}

func TestGetAccept_Priority(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?accept=cui-web", nil)
	req.Header.Set("X-Yao-Accept", "cui-desktop")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Metadata: map[string]interface{}{
			"accept": "cui-native",
		},
	}

	accept := agentctx.GetAccept(c, completionReq)
	assert.Equal(t, agentctx.Accept(agentctx.AcceptWebCUI), accept, "Query parameter should take priority")
}

// ==================== GetAssistantID ====================

func TestGetAssistantID_FromModel(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Model: "gpt-4-turbo-yao_myassistant",
	}

	assistantID, err := agentctx.GetAssistantID(c, completionReq)
	require.NoError(t, err)
	assert.Equal(t, "myassistant", assistantID)
}

func TestGetAssistantID_Priority(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?assistant_id=from_query", nil)
	req.Header.Set("X-Yao-Assistant", "from_header")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Model: "gpt-4-yao_from_model",
	}

	assistantID, err := agentctx.GetAssistantID(c, completionReq)
	require.NoError(t, err)
	assert.Equal(t, "from_query", assistantID, "Query parameter should take priority")
}

// ==================== GetRoute ====================

func TestGetRoute_FromQuery(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?route=/dashboard/home", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	route := agentctx.GetRoute(c, nil)
	assert.Equal(t, "/dashboard/home", route)
}

func TestGetRoute_FromHeader(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Route", "/settings/profile")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	route := agentctx.GetRoute(c, nil)
	assert.Equal(t, "/settings/profile", route)
}

func TestGetRoute_FromPayload(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Route: "/admin/users",
	}

	route := agentctx.GetRoute(c, completionReq)
	assert.Equal(t, "/admin/users", route)
}

func TestGetRoute_Priority(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?route=/from/query", nil)
	req.Header.Set("X-Yao-Route", "/from/header")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Route: "/from/payload",
	}

	route := agentctx.GetRoute(c, completionReq)
	assert.Equal(t, "/from/query", route, "Query parameter should take priority")
}

// ==================== GetMetadata ====================

func TestGetMetadata_FromQuery(t *testing.T) {
	testprepare.PrepareSandbox(t)

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

	result := agentctx.GetMetadata(c, nil)
	require.NotNil(t, result)
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, float64(123), result["key2"])
}

func TestGetMetadata_FromHeader_Base64(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	dataBase64 := "eyJ1c2VyX2lkIjo0NTYsImFjdGlvbiI6ImNyZWF0ZSJ9" // {"user_id":456,"action":"create"}

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Metadata", dataBase64)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := agentctx.GetMetadata(c, nil)
	require.NotNil(t, result)
	assert.Equal(t, "create", result["action"])
	assert.Equal(t, float64(456), result["user_id"])
}

func TestGetMetadata_FromPayload(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	data := map[string]interface{}{
		"page":  float64(1),
		"limit": float64(10),
	}

	completionReq := &agentctx.CompletionRequest{
		Metadata: data,
	}

	result := agentctx.GetMetadata(c, completionReq)
	require.NotNil(t, result)
	assert.Equal(t, float64(1), result["page"])
	assert.Equal(t, float64(10), result["limit"])
}

func TestGetMetadata_Priority(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	queryData := map[string]interface{}{
		"source": "query",
	}
	queryDataJSON, _ := json.Marshal(queryData)

	headerDataBase64 := "eyJzb3VyY2UiOiJoZWFkZXIifQ==" // {"source":"header"}

	req := httptest.NewRequest("GET", "/chat/completions?metadata="+string(queryDataJSON), nil)
	req.Header.Set("X-Yao-Metadata", headerDataBase64)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	payloadData := map[string]interface{}{
		"source": "payload",
	}

	completionReq := &agentctx.CompletionRequest{
		Metadata: payloadData,
	}

	result := agentctx.GetMetadata(c, completionReq)
	require.NotNil(t, result)
	assert.Equal(t, "query", result["source"], "Query parameter should take priority")
}

func TestGetMetadata_EmptyData(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := agentctx.GetMetadata(c, nil)
	assert.Nil(t, result)
}

// ==================== GetSkip ====================

func TestGetSkip_FromBody(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Skip: &agentctx.Skip{
			History: true,
			Trace:   false,
		},
	}

	skip := agentctx.GetSkip(c, completionReq)
	require.NotNil(t, skip)
	assert.True(t, skip.History)
	assert.False(t, skip.Trace)
}

func TestGetSkip_FromQueryParams(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?skip_history=true&skip_trace=false", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	skip := agentctx.GetSkip(c, nil)
	require.NotNil(t, skip)
	assert.True(t, skip.History)
	assert.False(t, skip.Trace)
}

func TestGetSkip_FromQueryParams_ShortForm(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?skip_history=1&skip_trace=1", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	skip := agentctx.GetSkip(c, nil)
	require.NotNil(t, skip)
	assert.True(t, skip.History)
	assert.True(t, skip.Trace)
}

func TestGetSkip_Priority(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/chat/completions?skip_history=false&skip_trace=false", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Skip: &agentctx.Skip{
			History: true,
			Trace:   true,
		},
	}

	skip := agentctx.GetSkip(c, completionReq)
	require.NotNil(t, skip)
	assert.True(t, skip.History, "Body should take priority")
	assert.True(t, skip.Trace, "Body should take priority")
}

func TestGetSkip_Nil(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	skip := agentctx.GetSkip(c, nil)
	assert.Nil(t, skip)
}

func TestGetSkip_OnlyHistorySet(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?skip_history=true", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	skip := agentctx.GetSkip(c, nil)
	require.NotNil(t, skip)
	assert.True(t, skip.History)
	assert.False(t, skip.Trace)
}

func TestGetSkip_FromBodyViaParseRequest(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	requestBody := map[string]interface{}{
		"model": "workers.system.title-yao_test",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Generate a title for this chat"},
		},
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

	completionReq, err := parseCompletionRequestData(c)
	require.NoError(t, err)

	require.NotNil(t, completionReq.Skip)
	assert.True(t, completionReq.Skip.History)
	assert.False(t, completionReq.Skip.Trace)

	skip := agentctx.GetSkip(c, completionReq)
	require.NotNil(t, skip)
	assert.True(t, skip.History)
	assert.False(t, skip.Trace)
}

// ==================== GetMode ====================

func TestGetMode_FromQuery(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?mode=task", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	mode := agentctx.GetMode(c, nil)
	assert.Equal(t, "task", mode)
}

func TestGetMode_FromHeader(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	req.Header.Set("X-Yao-Mode", "chat")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	mode := agentctx.GetMode(c, nil)
	assert.Equal(t, "chat", mode)
}

func TestGetMode_FromMetadata(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Metadata: map[string]interface{}{
			"mode": "task",
		},
	}

	mode := agentctx.GetMode(c, completionReq)
	assert.Equal(t, "task", mode)
}

func TestGetMode_Priority(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions?mode=query_mode", nil)
	req.Header.Set("X-Yao-Mode", "header_mode")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq := &agentctx.CompletionRequest{
		Metadata: map[string]interface{}{
			"mode": "metadata_mode",
		},
	}

	mode := agentctx.GetMode(c, completionReq)
	assert.Equal(t, "query_mode", mode, "Query parameter should take priority")
}

func TestGetMode_Empty(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/chat/completions", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	mode := agentctx.GetMode(c, nil)
	assert.Empty(t, mode)
}

// ==================== GetCompletionRequest additional tests ====================

func TestGetCompletionRequest_WriterInitialized(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	require.NoError(t, err)

	requestBody := map[string]interface{}{
		"model": "gpt-4-yao_test",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Test message"},
		},
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	completionReq, ctx, opts, err := agentctx.GetCompletionRequest(c, cache)
	require.NoError(t, err)
	defer ctx.Release()

	assert.NotNil(t, ctx.Writer)
	assert.Equal(t, c.Writer, ctx.Writer)
	require.NotNil(t, opts)
	assert.Equal(t, "gpt-4-yao_test", completionReq.Model)
	assert.Equal(t, "test", ctx.AssistantID)
	assert.NotEmpty(t, ctx.ChatID)
}

func TestGetCompletionRequest_ChatIDFallback(t *testing.T) {
	testprepare.PrepareSandbox(t)

	gin.SetMode(gin.TestMode)

	cache, err := store.Get("__yao.agent.cache")
	require.NoError(t, err)

	requestBody := map[string]interface{}{
		"model": "gpt-4-yao_assistant1",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Test message"},
		},
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/chat/completions", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	_, ctx, opts, err := agentctx.GetCompletionRequest(c, cache)
	require.NoError(t, err)
	defer ctx.Release()

	require.NotNil(t, opts)
	assert.NotEmpty(t, ctx.ChatID)
	assert.GreaterOrEqual(t, len(ctx.ChatID), 8)
}
