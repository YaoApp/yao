//go:build unit

package events_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	events "github.com/yaoapp/yao/agent/robot/events"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	eventtypes "github.com/yaoapp/yao/event/types"
)

func TestRobotHandler_DeliveryWebhook(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		_ = decoder.Decode(&received)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	handler := events.NewTestHandler()
	ev := &eventtypes.Event{
		Type:   events.Delivery,
		ID:     "test-ev-1",
		IsCall: true,
		Payload: events.DeliveryPayload{
			ExecutionID: "exec-1",
			MemberID:    "member-1",
			TeamID:      "team-1",
			Content: &robottypes.DeliveryContent{
				Summary: "test summary",
				Body:    "test body",
			},
			Preferences: &robottypes.DeliveryPreferences{
				Webhook: &robottypes.WebhookPreference{
					Enabled: true,
					Targets: []robottypes.WebhookTarget{
						{URL: server.URL},
					},
				},
			},
		},
	}

	resp := make(chan eventtypes.Result, 1)
	handler.Handle(context.Background(), ev, resp)

	result := <-resp
	require.NotNil(t, result.Data)
	assert.NoError(t, result.Err)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "exec-1", data["execution_id"])

	require.NotNil(t, received)
	assert.Equal(t, "robot.delivery", received["event"])
}

func TestRobotHandler_DeliveryNoContent(t *testing.T) {
	handler := events.NewTestHandler()
	ev := &eventtypes.Event{
		Type:   events.Delivery,
		ID:     "test-ev-2",
		IsCall: true,
		Payload: events.DeliveryPayload{
			ExecutionID: "exec-2",
			MemberID:    "member-2",
			TeamID:      "team-2",
		},
	}

	resp := make(chan eventtypes.Result, 1)
	handler.Handle(context.Background(), ev, resp)

	result := <-resp
	assert.Equal(t, "no content", result.Data)
}

func TestRobotHandler_DeliveryNoPreferences(t *testing.T) {
	handler := events.NewTestHandler()
	ev := &eventtypes.Event{
		Type:   events.Delivery,
		ID:     "test-ev-3",
		IsCall: true,
		Payload: events.DeliveryPayload{
			ExecutionID: "exec-3",
			MemberID:    "member-3",
			TeamID:      "team-3",
			Content: &robottypes.DeliveryContent{
				Summary: "test",
				Body:    "body",
			},
		},
	}

	resp := make(chan eventtypes.Result, 1)
	handler.Handle(context.Background(), ev, resp)

	result := <-resp
	assert.Equal(t, "no preferences, skipped", result.Data)
}

func TestRobotHandler_InvalidPayload(t *testing.T) {
	handler := events.NewTestHandler()
	ev := &eventtypes.Event{
		Type:    events.Delivery,
		ID:      "test-ev-4",
		IsCall:  true,
		Payload: "invalid",
	}

	resp := make(chan eventtypes.Result, 1)
	handler.Handle(context.Background(), ev, resp)

	result := <-resp
	assert.Error(t, result.Err)
}

func TestRobotHandler_UnhandledEvent(t *testing.T) {
	handler := events.NewTestHandler()
	ev := &eventtypes.Event{
		Type: "robot.unknown",
		ID:   "test-ev-5",
	}

	resp := make(chan eventtypes.Result, 1)
	handler.Handle(context.Background(), ev, resp)
}

func TestRobotHandler_Shutdown(t *testing.T) {
	handler := events.NewTestHandler()
	err := handler.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestComputeHMACSignature(t *testing.T) {
	payload := []byte(`{"event":"robot.delivery"}`)
	secret := "test-secret"

	sig := events.ComputeHMACSignature(payload, secret)
	assert.NotEmpty(t, sig)
	assert.Len(t, sig, 64) // SHA-256 hex is 64 chars
}

func TestVerifyHMACSignature(t *testing.T) {
	payload := []byte(`{"event":"robot.delivery"}`)
	secret := "test-secret"

	sig := events.ComputeHMACSignature(payload, secret)
	assert.True(t, events.VerifyHMACSignature(payload, secret, sig))
	assert.False(t, events.VerifyHMACSignature(payload, "wrong-secret", sig))
}

func TestVerifyHMACSignature_EmptyPayload(t *testing.T) {
	payload := []byte("")
	secret := "test-secret"

	sig := events.ComputeHMACSignature(payload, secret)
	assert.True(t, events.VerifyHMACSignature(payload, secret, sig))
}

func TestVerifyHMACSignature_TamperedPayload(t *testing.T) {
	payload := []byte(`{"event":"robot.delivery"}`)
	secret := "test-secret"

	sig := events.ComputeHMACSignature(payload, secret)
	tampered := []byte(`{"event":"robot.delivery","extra":"injected"}`)
	assert.False(t, events.VerifyHMACSignature(tampered, secret, sig))
}

func TestRobotHandler_WebhookWithSignature(t *testing.T) {
	var receivedSig string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-Yao-Signature")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	handler := events.NewTestHandler()
	ev := &eventtypes.Event{
		Type:   events.Delivery,
		ID:     "test-ev-sig",
		IsCall: true,
		Payload: events.DeliveryPayload{
			ExecutionID: "exec-sig",
			MemberID:    "member-sig",
			TeamID:      "team-sig",
			Content: &robottypes.DeliveryContent{
				Summary: "signed delivery",
				Body:    "body",
			},
			Preferences: &robottypes.DeliveryPreferences{
				Webhook: &robottypes.WebhookPreference{
					Enabled: true,
					Targets: []robottypes.WebhookTarget{
						{URL: server.URL, Secret: "my-webhook-secret"},
					},
				},
			},
		},
	}

	resp := make(chan eventtypes.Result, 1)
	handler.Handle(context.Background(), ev, resp)

	result := <-resp
	require.NotNil(t, result.Data)
	assert.NoError(t, result.Err)
	assert.NotEmpty(t, receivedSig, "webhook should receive HMAC signature header")
	assert.Len(t, receivedSig, 64)
}
