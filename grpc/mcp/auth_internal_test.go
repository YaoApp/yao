package mcp

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"

	"github.com/yaoapp/yao/grpc/auth"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func TestAuthProviderFromCtx_ChatAndAssistant(t *testing.T) {
	ctx := context.Background()
	ctx = auth.WithAuthorizedInfo(ctx, &types.AuthorizedInfo{
		Subject:   "user-1",
		UserID:    "uid-1",
		ClientID:  "client-1",
		SessionID: "sess-1",
		TeamID:    "team-1",
		TenantID:  "tenant-1",
	})

	md := metadata.Pairs(
		"x-workspace-id", "ws-123",
		"x-sandbox-id", "sb-456",
		"x-chat-id", "chat-789",
		"x-assistant-id", "ast-abc",
		"x-locale", "en-US",
	)
	ctx = metadata.NewIncomingContext(ctx, md)

	ap := authProviderFromCtx(ctx)
	if ap == nil {
		t.Fatal("expected non-nil authProvider")
	}

	m := ap.GetAuthorizedMap()
	checks := map[string]string{
		"workspace_id": "ws-123",
		"sandbox_id":   "sb-456",
		"chat_id":      "chat-789",
		"assistant_id": "ast-abc",
		"locale":       "en-US",
		"user_id":      "uid-1",
	}
	for key, want := range checks {
		got, ok := m[key]
		if !ok {
			t.Errorf("missing key %q in auth map", key)
			continue
		}
		if got != want {
			t.Errorf("auth map[%q] = %q, want %q", key, got, want)
		}
	}
}

func TestAuthProviderFromCtx_NoChatID(t *testing.T) {
	ctx := context.Background()
	ctx = auth.WithAuthorizedInfo(ctx, &types.AuthorizedInfo{
		Subject: "user-2",
		UserID:  "uid-2",
	})

	md := metadata.Pairs("x-workspace-id", "ws-only")
	ctx = metadata.NewIncomingContext(ctx, md)

	ap := authProviderFromCtx(ctx)
	if ap == nil {
		t.Fatal("expected non-nil authProvider")
	}

	m := ap.GetAuthorizedMap()
	if _, ok := m["chat_id"]; ok {
		t.Error("chat_id should not be present when x-chat-id header is missing")
	}
	if _, ok := m["assistant_id"]; ok {
		t.Error("assistant_id should not be present when x-assistant-id header is missing")
	}
	if m["workspace_id"] != "ws-only" {
		t.Errorf("workspace_id = %q, want %q", m["workspace_id"], "ws-only")
	}
}
