//go:build integration

package inbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/inbox"
	"github.com/yaoapp/yao/config"
	yaotest "github.com/yaoapp/yao/test"
)

func TestInboxOperations(t *testing.T) {
	yaotest.Prepare(t, config.Conf)
	defer yaotest.Clean()

	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: "test-user-001",
		TeamID: "test-team-001",
	}

	// Insert test mail directly
	mailID := uuid.New().String()
	now := time.Now()
	err := capsule.Global.Query().Table("agent_mail").Insert(map[string]interface{}{
		"mail_id":          mailID,
		"type":             "input",
		"priority":         "high",
		"title":            "Test: Needs Input",
		"body":             "Please provide input",
		"chat_id":          "chat-test-001",
		"read":             false,
		"archived":         false,
		"starred":          false,
		"pinned":           false,
		"__yao_created_by": auth.UserID,
		"__yao_team_id":    auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	})
	if err != nil {
		t.Fatalf("insert test mail failed: %v", err)
	}

	// Test List
	result, err := inbox.List(ctx, auth, &inbox.ListQuery{Filter: "all"})
	if err != nil {
		t.Fatalf("inbox.List failed: %v", err)
	}
	if len(result.Mails) == 0 {
		t.Fatal("expected at least 1 mail")
	}

	// Test UnreadCount
	counts, err := inbox.UnreadCount(ctx, auth)
	if err != nil {
		t.Fatalf("inbox.UnreadCount failed: %v", err)
	}
	if counts.Total == 0 {
		t.Error("expected non-zero unread count")
	}
	if counts.Input == 0 {
		t.Error("expected non-zero input count")
	}

	// Test Read
	err = inbox.Read(ctx, auth, mailID)
	if err != nil {
		t.Fatalf("inbox.Read failed: %v", err)
	}

	// Verify read
	countsAfter, err := inbox.UnreadCount(ctx, auth)
	if err != nil {
		t.Fatalf("inbox.UnreadCount after read failed: %v", err)
	}
	if countsAfter.Total >= counts.Total {
		t.Error("unread count should decrease after marking as read")
	}

	// Test Star
	err = inbox.Star(ctx, auth, mailID)
	if err != nil {
		t.Fatalf("inbox.Star failed: %v", err)
	}

	// Test Pin
	err = inbox.Pin(ctx, auth, mailID)
	if err != nil {
		t.Fatalf("inbox.Pin failed: %v", err)
	}

	// Test Archive
	err = inbox.Archive(ctx, auth, mailID)
	if err != nil {
		t.Fatalf("inbox.Archive failed: %v", err)
	}
}
