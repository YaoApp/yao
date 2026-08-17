//go:build unit

package dsh_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	dsh "github.com/yaoapp/yao/agent/sandbox/v2/dsh"
)

// --- buildCancelRPC ---

func TestBuildCancelRPC_Format(t *testing.T) {
	rpc := dsh.ExportBuildCancelRPC("chat-123")
	s := string(rpc)
	if !strings.Contains(s, `"method":"session/cancel"`) {
		t.Errorf("missing method: %s", s)
	}
	if !strings.Contains(s, `"sessionId":"chat-123"`) {
		t.Errorf("missing sessionId: %s", s)
	}
	if !strings.HasSuffix(s, "\n") {
		t.Error("cancel RPC must end with newline")
	}
}

func TestBuildCancelRPC_JSONEscape(t *testing.T) {
	rpc := dsh.ExportBuildCancelRPC(`chat"with"quotes`)
	s := string(rpc)
	var msg map[string]interface{}
	if err := json.Unmarshal(rpc[:len(rpc)-1], &msg); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, s)
	}
	params := msg["params"].(map[string]interface{})
	if params["sessionId"] != `chat"with"quotes` {
		t.Errorf("sessionId not properly escaped: %v", params["sessionId"])
	}
}

func TestBuildCancelRPC_EmptyID(t *testing.T) {
	rpc := dsh.ExportBuildCancelRPC("")
	var msg map[string]interface{}
	if err := json.Unmarshal(rpc[:len(rpc)-1], &msg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	params := msg["params"].(map[string]interface{})
	if params["sessionId"] != "" {
		t.Errorf("expected empty sessionId, got %v", params["sessionId"])
	}
}

func TestBuildCancelRPC_SpecialChars(t *testing.T) {
	cases := []string{
		"chat\nwith\nnewlines",
		"chat\\backslash",
		"chat\twith\ttabs",
		"chat/with/slashes",
		`chat{"json":"injection"}`,
	}
	for _, id := range cases {
		t.Run(fmt.Sprintf("id=%q", id), func(t *testing.T) {
			rpc := dsh.ExportBuildCancelRPC(id)
			var msg map[string]interface{}
			if err := json.Unmarshal(rpc[:len(rpc)-1], &msg); err != nil {
				t.Fatalf("invalid JSON for id=%q: %v\nraw: %s", id, err, string(rpc))
			}
			params := msg["params"].(map[string]interface{})
			if params["sessionId"] != id {
				t.Errorf("sessionId mismatch: want %q, got %q", id, params["sessionId"])
			}
		})
	}
}

func TestBuildCancelRPC_IDField(t *testing.T) {
	rpc := dsh.ExportBuildCancelRPC("x")
	var msg map[string]interface{}
	json.Unmarshal(rpc[:len(rpc)-1], &msg)
	id := msg["id"].(float64)
	if id != 9998 {
		t.Errorf("cancel RPC id = %v, want 9998", id)
	}
}

// --- buildShutdownRPC ---

func TestBuildShutdownRPC_Format(t *testing.T) {
	rpc := dsh.ExportBuildShutdownRPC()
	s := string(rpc)
	if !strings.Contains(s, `"method":"shutdown"`) {
		t.Errorf("missing method: %s", s)
	}
	if !strings.Contains(s, `"id":9999`) {
		t.Errorf("missing id: %s", s)
	}
	if !strings.HasSuffix(s, "\n") {
		t.Error("shutdown RPC must end with newline")
	}
	var msg map[string]interface{}
	if err := json.Unmarshal(rpc[:len(rpc)-1], &msg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestBuildShutdownRPC_NoParams(t *testing.T) {
	rpc := dsh.ExportBuildShutdownRPC()
	var msg map[string]interface{}
	json.Unmarshal(rpc[:len(rpc)-1], &msg)
	if _, has := msg["params"]; has {
		t.Error("shutdown RPC should not have params")
	}
}

func TestBuildCancelAndShutdown_DistinctIDs(t *testing.T) {
	cancel := dsh.ExportBuildCancelRPC("test")
	shutdown := dsh.ExportBuildShutdownRPC()

	var cm, sm map[string]interface{}
	json.Unmarshal(cancel[:len(cancel)-1], &cm)
	json.Unmarshal(shutdown[:len(shutdown)-1], &sm)

	cancelID := cm["id"].(float64)
	shutdownID := sm["id"].(float64)
	if cancelID == shutdownID {
		t.Errorf("cancel and shutdown should have distinct IDs, both are %v", cancelID)
	}
}

// --- isContextErr ---

func TestIsContextErr_Canceled(t *testing.T) {
	if !dsh.ExportIsContextErr(context.Canceled) {
		t.Error("context.Canceled should be recognized")
	}
}

func TestIsContextErr_DeadlineExceeded(t *testing.T) {
	if !dsh.ExportIsContextErr(context.DeadlineExceeded) {
		t.Error("context.DeadlineExceeded should be recognized")
	}
}

func TestIsContextErr_Nil(t *testing.T) {
	if dsh.ExportIsContextErr(nil) {
		t.Error("nil should not be context error")
	}
}

func TestIsContextErr_OtherError(t *testing.T) {
	if dsh.ExportIsContextErr(context.TODO().Err()) {
		t.Error("non-cancelled context error should not match")
	}
}

func TestIsContextErr_WrappedCanceled(t *testing.T) {
	err := fmt.Errorf("wrapper: %w", context.Canceled)
	if !dsh.ExportIsContextErr(err) {
		t.Error("wrapped context.Canceled should be recognized")
	}
}
