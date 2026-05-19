package events

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	eventtypes "github.com/yaoapp/yao/event/types"
)

func newTestHandler() *robotHandler {
	return &robotHandler{
		httpClient: http.DefaultClient,
	}
}

func TestRobotHandler_DeliveryWebhook(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		_ = decoder.Decode(&received)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	handler := newTestHandler()
	ev := &eventtypes.Event{
		Type:   Delivery,
		ID:     "test-ev-1",
		IsCall: true,
		Payload: DeliveryPayload{
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
	handler := newTestHandler()
	ev := &eventtypes.Event{
		Type:   Delivery,
		ID:     "test-ev-2",
		IsCall: true,
		Payload: DeliveryPayload{
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
	handler := newTestHandler()
	ev := &eventtypes.Event{
		Type:   Delivery,
		ID:     "test-ev-3",
		IsCall: true,
		Payload: DeliveryPayload{
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
	handler := newTestHandler()
	ev := &eventtypes.Event{
		Type:    Delivery,
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
	handler := newTestHandler()
	ev := &eventtypes.Event{
		Type: "robot.unknown",
		ID:   "test-ev-5",
	}

	resp := make(chan eventtypes.Result, 1)
	handler.Handle(context.Background(), ev, resp)
	// Fire-and-forget, no response expected
}

func TestRobotHandler_Shutdown(t *testing.T) {
	handler := newTestHandler()
	err := handler.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestVerifyHMACSignature(t *testing.T) {
	payload := []byte(`{"event":"robot.delivery"}`)
	secret := "test-secret"

	sig := ComputeHMACSignature(payload, secret)
	assert.True(t, VerifyHMACSignature(payload, secret, sig))
	assert.False(t, VerifyHMACSignature(payload, "wrong-secret", sig))
}
