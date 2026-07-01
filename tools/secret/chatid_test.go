package secret

import (
	"context"
	"testing"

	"github.com/yaoapp/gou/process"
	"google.golang.org/grpc/metadata"
)

func TestExtractChatID_Present(t *testing.T) {
	md := metadata.Pairs("x-chat-id", "chat-123")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	proc := &process.Process{Context: ctx}

	chatID := extractChatID(proc)
	if chatID != "chat-123" {
		t.Errorf("extractChatID = %q, want %q", chatID, "chat-123")
	}
}

func TestExtractChatID_Missing(t *testing.T) {
	md := metadata.Pairs("x-assistant-id", "ast-1")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	proc := &process.Process{Context: ctx}

	chatID := extractChatID(proc)
	if chatID != "" {
		t.Errorf("extractChatID = %q, want empty", chatID)
	}
}

func TestExtractChatID_NilContext(t *testing.T) {
	proc := &process.Process{Context: nil}
	chatID := extractChatID(proc)
	if chatID != "" {
		t.Errorf("extractChatID with nil ctx = %q, want empty", chatID)
	}
}

func TestExtractSecretsMap_Valid(t *testing.T) {
	data := map[string]interface{}{
		"secrets": map[string]interface{}{
			"API_KEY": map[string]interface{}{"value": "encrypted-val"},
		},
	}
	result := extractSecretsMap(data)
	if result == nil {
		t.Fatal("expected non-nil")
	}
	if _, ok := result["API_KEY"]; !ok {
		t.Error("expected API_KEY in result")
	}
}

func TestExtractSecretsMap_Nil(t *testing.T) {
	result := extractSecretsMap(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

func TestExtractSecretsMap_NoSecretsKey(t *testing.T) {
	data := map[string]interface{}{"other": "value"}
	result := extractSecretsMap(data)
	if result != nil {
		t.Errorf("expected nil when no secrets key, got %v", result)
	}
}

func TestTaskNamespace(t *testing.T) {
	ns := taskNamespace("chat-xyz")
	if ns != "task-config.task.chat-xyz" {
		t.Errorf("taskNamespace = %q, want %q", ns, "task-config.task.chat-xyz")
	}
}
