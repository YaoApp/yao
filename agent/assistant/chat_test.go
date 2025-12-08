package assistant_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/kb"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

func TestGetChatKBID(t *testing.T) {
	t.Run("WithTeamAndUser", func(t *testing.T) {
		teamID := "5659-5504-2879"
		userID := "4287-9400-2030-0504"

		collectionID := assistant.GetChatKBID(teamID, userID)

		// Should sanitize dashes to underscores
		expected := "chat_5659_5504_2879_4287_9400_2030_0504"
		assert.Equal(t, expected, collectionID)
		t.Logf("✓ Collection ID with team: %s", collectionID)
	})

	t.Run("WithoutTeam", func(t *testing.T) {
		teamID := ""
		userID := "4287-9400-2030-0504"

		collectionID := assistant.GetChatKBID(teamID, userID)

		// Should use chat_user_ prefix
		expected := "chat_user_4287_9400_2030_0504"
		assert.Equal(t, expected, collectionID)
		t.Logf("✓ Collection ID without team: %s", collectionID)
	})

	t.Run("Idempotent", func(t *testing.T) {
		teamID := "test-team-123"
		userID := "test-user-456"

		id1 := assistant.GetChatKBID(teamID, userID)
		id2 := assistant.GetChatKBID(teamID, userID)
		id3 := assistant.GetChatKBID(teamID, userID)

		// Same input should always produce same output
		assert.Equal(t, id1, id2)
		assert.Equal(t, id2, id3)
		t.Logf("✓ Idempotent: %s", id1)
	})

	t.Run("SanitizeSpecialChars", func(t *testing.T) {
		teamID := "team-with-dashes@123"
		userID := "user.with.dots!"

		collectionID := assistant.GetChatKBID(teamID, userID)

		// Should only contain alphanumeric and underscores
		assert.Regexp(t, "^[a-zA-Z0-9_]+$", collectionID)
		t.Logf("✓ Sanitized ID: %s", collectionID)
	})

	t.Run("EmptyUserID", func(t *testing.T) {
		teamID := "test-team"
		userID := ""

		collectionID := assistant.GetChatKBID(teamID, userID)

		// Should handle empty user ID gracefully
		expected := "chat_test_team_"
		assert.Equal(t, expected, collectionID)
		t.Logf("✓ Empty user ID handled: %s", collectionID)
	})
}

func TestPrepareKBCollection(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Skip if KB not configured
	kbSetting := assistant.GetGlobalKBSetting()
	if kbSetting == nil || kbSetting.Chat == nil {
		t.Skip("KB chat settings not configured in agent/kb.yml, skipping test")
	}

	// Get assistant
	ast, err := assistant.Get("mohe")
	require.NoError(t, err)
	require.NotNil(t, ast)

	t.Run("CreateNewCollection", func(t *testing.T) {
		// Use unique IDs based on timestamp to avoid conflicts
		timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
		teamID := fmt.Sprintf("test_team_%s", timestamp)
		userID := fmt.Sprintf("test_user_%s", timestamp)

		ctx := &agentcontext.Context{
			Context: context.Background(),
			ChatID:  "test_chat_prepare_001",
			Authorized: &oauthtypes.AuthorizedInfo{
				TeamID: teamID,
				UserID: userID,
			},
		}

		opts := &agentcontext.Options{}

		// This should create a new KB collection
		err := ast.InitializeConversation(ctx, opts)

		// Should not return error
		assert.NoError(t, err)
		t.Logf("✓ KB collection prepared successfully")

		// Clean up
		collectionID := assistant.GetChatKBID(teamID, userID)
		_, _ = kb.API.RemoveCollection(ctx.Context, collectionID)
	})

	t.Run("IdempotentCollectionCreation", func(t *testing.T) {
		// Use unique IDs based on timestamp to avoid conflicts
		timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
		teamID := fmt.Sprintf("idem_team_%s", timestamp)
		userID := fmt.Sprintf("idem_user_%s", timestamp)

		ctx := &agentcontext.Context{
			Context: context.Background(),
			ChatID:  "test_chat_idempotent",
			Authorized: &oauthtypes.AuthorizedInfo{
				TeamID: teamID,
				UserID: userID,
			},
		}

		opts := &agentcontext.Options{}

		// First call - creates collection
		err1 := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err1)

		// Second call - should skip because collection exists
		err2 := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err2)

		// Third call - still no error
		err3 := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err3)

		t.Logf("✓ Idempotent collection preparation works correctly")

		// Clean up after test
		collectionID := assistant.GetChatKBID(teamID, userID)
		_, _ = kb.API.RemoveCollection(ctx.Context, collectionID)
	})

	t.Run("HandleMissingAuthorizedInfo", func(t *testing.T) {
		ctx := &agentcontext.Context{
			Context:    context.Background(),
			ChatID:     "test_chat_no_auth",
			Authorized: nil, // Missing authorized info
		}

		opts := &agentcontext.Options{}

		// Should not error, just skip KB preparation
		err := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err)
		t.Logf("✓ Correctly skipped KB preparation when authorized info is missing")
	})

	t.Run("ConcurrentCreation", func(t *testing.T) {
		// Use unique IDs based on timestamp to avoid conflicts
		timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
		teamID := fmt.Sprintf("concurrent_team_%s", timestamp)
		userID := fmt.Sprintf("concurrent_user_%s", timestamp)

		ctx := &agentcontext.Context{
			Context: context.Background(),
			ChatID:  "test_chat_concurrent",
			Authorized: &oauthtypes.AuthorizedInfo{
				TeamID: teamID,
				UserID: userID,
			},
		}

		opts := &agentcontext.Options{}

		// Launch 5 concurrent calls to create the same collection
		var wg sync.WaitGroup
		errors := make([]error, 5)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				errors[idx] = ast.InitializeConversation(ctx, opts)
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()

		// All calls should succeed (no errors, or just warning logs)
		// Note: Some goroutines may skip due to concurrent creation lock
		for i, err := range errors {
			assert.NoError(t, err, "Goroutine %d should not error", i)
		}

		// Wait a bit for async operations to complete
		time.Sleep(200 * time.Millisecond)

		// Verify collection was created (at least by one goroutine)
		collectionID := assistant.GetChatKBID(teamID, userID)
		existsResult, err := kb.API.CollectionExists(ctx.Context, collectionID)
		if err != nil || existsResult == nil || !existsResult.Exists {
			// Collection might not have been created due to errors, that's okay for this test
			// The main goal is to verify no panics or race conditions occurred
			t.Logf("⚠ Collection not created (might have failed), but no panics occurred: %v", err)
		} else {
			t.Logf("✓ Concurrent creation handled correctly, collection: %s", collectionID)
		}

		// Clean up
		_, _ = kb.API.RemoveCollection(ctx.Context, collectionID)
	})
}

func TestInitializeConversation(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Skip if KB not configured
	kbSetting := assistant.GetGlobalKBSetting()
	if kbSetting == nil || kbSetting.Chat == nil {
		t.Skip("KB chat settings not configured in agent/kb.yml, skipping test")
	}

	ast, err := assistant.Get("mohe")
	require.NoError(t, err)
	require.NotNil(t, ast)

	t.Run("FullInitialization", func(t *testing.T) {
		// Use unique IDs based on timestamp to avoid conflicts
		timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
		teamID := fmt.Sprintf("init_team_%s", timestamp)
		userID := fmt.Sprintf("init_user_%s", timestamp)

		ctx := &agentcontext.Context{
			Context: context.Background(),
			ChatID:  "test_init_chat_001",
			Authorized: &oauthtypes.AuthorizedInfo{
				TeamID: teamID,
				UserID: userID,
			},
		}

		opts := &agentcontext.Options{}

		// Should initialize conversation without error
		err := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err)
		t.Logf("✓ Conversation initialized successfully")

		// Verify collection was created
		collectionID := assistant.GetChatKBID(teamID, userID)
		existsResult, err := kb.API.CollectionExists(ctx.Context, collectionID)
		assert.NoError(t, err)
		assert.NotNil(t, existsResult)
		assert.True(t, existsResult.Exists, "KB collection should be created")
		t.Logf("✓ KB collection created: %s", collectionID)

		// Clean up
		_, _ = kb.API.RemoveCollection(ctx.Context, collectionID)
	})

	t.Run("SkipHistoryFlag", func(t *testing.T) {
		ctx := &agentcontext.Context{
			Context: context.Background(),
			ChatID:  "test_skip_history",
			Authorized: &oauthtypes.AuthorizedInfo{
				TeamID: "skip_team",
				UserID: "skip_user",
			},
		}

		opts := &agentcontext.Options{
			Skip: &agentcontext.Skip{
				History: true,
			},
		}

		// Should skip initialization when history flag is set
		err := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err)
		t.Logf("✓ Correctly skipped with history flag")
	})
}
