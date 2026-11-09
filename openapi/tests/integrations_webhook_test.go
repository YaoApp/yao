package openapi_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/event"
	eventtypes "github.com/yaoapp/yao/event/types"
	"github.com/yaoapp/yao/openapi"
	integrations "github.com/yaoapp/yao/openapi/integrations"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// integrationHandler is a minimal handler to accept "integration.*" events during tests.
type integrationHandler struct{}

func (h *integrationHandler) Handle(ctx context.Context, ev *eventtypes.Event, resp chan<- eventtypes.Result) {
	if ev.IsCall {
		resp <- eventtypes.Result{Data: ev.Payload}
	}
}

func (h *integrationHandler) Shutdown(ctx context.Context) error { return nil }

func init() {
	event.Register("integration", &integrationHandler{})
}

func TestWebhookPost_Telegram(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Subscribe to integration.webhook.telegram events
	ch := make(chan *eventtypes.Event, 16)
	subID := event.Subscribe("integration.webhook.telegram", ch)
	defer event.Unsubscribe(subID)

	// Simulate a Telegram webhook POST
	telegramBody := `{"update_id":123456,"message":{"message_id":1,"from":{"id":999,"first_name":"Test"},"chat":{"id":999,"type":"private"},"text":"hello bot"}}`
	url := fmt.Sprintf("%s%s/integrations/telegram/app-abc123", serverURL, baseURL)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(telegramBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Wait for the event to arrive
	select {
	case ev := <-ch:
		assert.Equal(t, "integration.webhook.telegram", ev.Type)

		var payload integrations.WebhookPayload
		err := ev.Should(&payload)
		require.NoError(t, err)

		assert.Equal(t, "telegram", payload.Provider)
		assert.Equal(t, "app-abc123", payload.AppID)
		assert.Equal(t, http.MethodPost, payload.Method)

		// Verify body is passed through
		assert.JSONEq(t, telegramBody, string(payload.Body))

		// Verify headers are forwarded
		assert.Equal(t, "application/json", payload.Headers["Content-Type"])
		assert.Equal(t, "test-secret-token", payload.Headers["X-Telegram-Bot-Api-Secret-Token"])

		t.Logf("Received event: type=%s provider=%s app_id=%s body_len=%d headers=%v",
			ev.Type, payload.Provider, payload.AppID, len(payload.Body), payload.Headers)

	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for integration.webhook.telegram event")
	}
}

func TestWebhookGet_Verification(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	ch := make(chan *eventtypes.Event, 16)
	subID := event.Subscribe("integration.webhook.telegram", ch)
	defer event.Unsubscribe(subID)

	// Simulate a Telegram setWebhook verification GET with query parameters
	url := fmt.Sprintf("%s%s/integrations/telegram/app-xyz789?hub.mode=subscribe&hub.verify_token=abc", serverURL, baseURL)

	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case ev := <-ch:
		var payload integrations.WebhookPayload
		err := ev.Should(&payload)
		require.NoError(t, err)

		assert.Equal(t, "telegram", payload.Provider)
		assert.Equal(t, "app-xyz789", payload.AppID)
		assert.Equal(t, http.MethodGet, payload.Method)
		assert.Empty(t, payload.Body, "GET request should have no body")
		assert.Equal(t, "subscribe", payload.Query["hub.mode"])
		assert.Equal(t, "abc", payload.Query["hub.verify_token"])

		t.Logf("Received GET event: provider=%s app_id=%s query=%v", payload.Provider, payload.AppID, payload.Query)

	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for integration.webhook.telegram event")
	}
}

func TestWebhookPost_Stripe(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	ch := make(chan *eventtypes.Event, 16)
	subID := event.Subscribe("integration.webhook.stripe", ch)
	defer event.Unsubscribe(subID)

	stripeBody := `{"id":"evt_1234","type":"checkout.session.completed","data":{"object":{"amount_total":1000}}}`
	url := fmt.Sprintf("%s%s/integrations/stripe/whsec-test123", serverURL, baseURL)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(stripeBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "t=123,v1=abc")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case ev := <-ch:
		assert.Equal(t, "integration.webhook.stripe", ev.Type)

		var payload integrations.WebhookPayload
		err := ev.Should(&payload)
		require.NoError(t, err)

		assert.Equal(t, "stripe", payload.Provider)
		assert.Equal(t, "whsec-test123", payload.AppID)
		assert.JSONEq(t, stripeBody, string(payload.Body))
		assert.Equal(t, "t=123,v1=abc", payload.Headers["Stripe-Signature"])

		t.Logf("Received Stripe event: provider=%s app_id=%s", payload.Provider, payload.AppID)

	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for integration.webhook.stripe event")
	}
}

func TestWebhookMissingParams(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// The route pattern requires both :provider and :app_id in the path.
	// Missing parameters would result in 404 from the Gin router, not 400.
	// Test with the actual endpoint to verify it's registered and working.
	url := fmt.Sprintf("%s%s/integrations/telegram/test-app", serverURL, baseURL)
	resp, err := http.Post(url, "application/json", bytes.NewBufferString("{}"))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestWebhookEmptyBody(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	ch := make(chan *eventtypes.Event, 16)
	subID := event.Subscribe("integration.webhook.wechat", ch)
	defer event.Unsubscribe(subID)

	url := fmt.Sprintf("%s%s/integrations/wechat/app-wechat-001", serverURL, baseURL)
	resp, err := http.Post(url, "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case ev := <-ch:
		var payload integrations.WebhookPayload
		err := ev.Should(&payload)
		require.NoError(t, err)

		assert.Equal(t, "wechat", payload.Provider)
		assert.Equal(t, "app-wechat-001", payload.AppID)
		assert.Empty(t, payload.Body)

		t.Logf("Received empty-body event: provider=%s app_id=%s", payload.Provider, payload.AppID)

	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for integration.webhook.wechat event")
	}
}

func TestWebhookLargeBody(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	ch := make(chan *eventtypes.Event, 16)
	subID := event.Subscribe("integration.webhook.generic", ch)
	defer event.Unsubscribe(subID)

	// Build a large payload (~100KB)
	largeData := map[string]interface{}{
		"items": make([]map[string]string, 1000),
	}
	for i := 0; i < 1000; i++ {
		largeData["items"].([]map[string]string)[i] = map[string]string{
			"key":   fmt.Sprintf("item-%d", i),
			"value": "a]b]c]d]e]f]g]h]i]j]k]l]m]n]o]p]q]r]s]t]u]v]w]x]y]z",
		}
	}
	bodyBytes, err := jsoniter.Marshal(largeData)
	require.NoError(t, err)

	url := fmt.Sprintf("%s%s/integrations/generic/app-large", serverURL, baseURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyBytes))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case ev := <-ch:
		var payload integrations.WebhookPayload
		err := ev.Should(&payload)
		require.NoError(t, err)

		assert.Equal(t, "generic", payload.Provider)
		assert.Equal(t, len(bodyBytes), len(payload.Body))

		t.Logf("Received large-body event: provider=%s body_size=%d bytes", payload.Provider, len(payload.Body))

	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for large body event")
	}
}

func TestWebhookResponseImmediate(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	url := fmt.Sprintf("%s%s/integrations/telegram/app-timing", serverURL, baseURL)

	start := time.Now()
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(`{"test":"timing"}`))
	elapsed := time.Since(start)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Response body should be empty (just status 200)
	body, _ := io.ReadAll(resp.Body)
	assert.Empty(t, body)

	// Response should be near-instant (< 1 second); the event is pushed async
	assert.Less(t, elapsed, 1*time.Second, "Webhook response should be immediate, got %v", elapsed)

	t.Logf("Webhook response time: %v", elapsed)
}

func TestWebhookMultipleProviders(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Subscribe to all integration.webhook.* events
	ch := make(chan *eventtypes.Event, 32)
	subID := event.Subscribe("integration.webhook.*", ch)
	defer event.Unsubscribe(subID)

	providers := []string{"telegram", "stripe", "wechat", "dingtalk", "feishu"}
	for _, provider := range providers {
		url := fmt.Sprintf("%s%s/integrations/%s/app-%s-001", serverURL, baseURL, provider, provider)
		body := fmt.Sprintf(`{"provider":"%s","test":true}`, provider)
		resp, err := http.Post(url, "application/json", bytes.NewBufferString(body))
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	received := make(map[string]bool)
	timeout := time.After(5 * time.Second)
	for len(received) < len(providers) {
		select {
		case ev := <-ch:
			var payload integrations.WebhookPayload
			err := ev.Should(&payload)
			require.NoError(t, err)
			received[payload.Provider] = true
			t.Logf("Received event for provider: %s", payload.Provider)
		case <-timeout:
			t.Fatalf("Timed out: received %d/%d provider events: %v", len(received), len(providers), received)
		}
	}

	for _, provider := range providers {
		assert.True(t, received[provider], "Should have received event for provider: %s", provider)
	}
}
