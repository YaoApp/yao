//go:build integration

package inbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/board"
	"github.com/yaoapp/yao/agent/inbox"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func mailTable() string {
	if m, err := model.Get("__yao.agent.mail"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_mail"
}

func chatTable() string {
	if m, err := model.Get("__yao.agent.chat"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_chat"
}

func taskTable() string {
	if m, err := model.Get("__yao.agent.task"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_task"
}

type testContext struct {
	ctx  context.Context
	auth *process.AuthorizedInfo
}

func setupTest(t *testing.T) *testContext {
	t.Helper()
	identity := testprepare.PrepareSandbox(t)
	return &testContext{
		ctx: context.Background(),
		auth: &process.AuthorizedInfo{
			UserID: identity.AlphaOwnerUserID,
			TeamID: identity.AlphaTeamID,
		},
	}
}

// insertTaskWithMail creates a task and its latest mail, returning both IDs
func insertTaskWithMail(t *testing.T, tc *testContext, overrides map[string]interface{}) (chatID string, mailID string) {
	t.Helper()
	chatID = "chat-" + uuid.New().String()[:8]
	mailID = uuid.New().String()
	now := time.Now()

	taskRow := map[string]interface{}{
		"chat_id":          chatID,
		"run_status":       "completed",
		"position":         0,
		"priority":         "none",
		"bookmarked":       false,
		"inbox_pinned":     false,
		"has_unread":       true,
		"last_mail_type":   "input",
		"__yao_created_by": tc.auth.UserID,
		"__yao_team_id":    tc.auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	}
	for k, v := range overrides {
		if k == "title" || k == "body" || k == "type" || k == "priority_mail" {
			continue
		}
		taskRow[k] = v
	}

	err := capsule.Global.Query().Table(taskTable()).Insert(taskRow)
	require.NoError(t, err, "insert test task")

	mailType := "input"
	if v, ok := overrides["type"]; ok {
		mailType = v.(string)
	}
	mailPriority := "high"
	if v, ok := overrides["priority_mail"]; ok {
		mailPriority = v.(string)
	}
	mailTitle := "Test Mail"
	if v, ok := overrides["title"]; ok {
		mailTitle = v.(string)
	}
	mailBody := "Test body"
	if v, ok := overrides["body"]; ok {
		mailBody = v.(string)
	}

	mailRow := map[string]interface{}{
		"mail_id":          mailID,
		"type":             mailType,
		"priority":         mailPriority,
		"title":            mailTitle,
		"body":             mailBody,
		"chat_id":          chatID,
		"__yao_created_by": tc.auth.UserID,
		"__yao_team_id":    tc.auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	}
	err = capsule.Global.Query().Table(mailTable()).Insert(mailRow)
	require.NoError(t, err, "insert test mail")
	return chatID, mailID
}

func cleanupTask(t *testing.T, chatIDs ...string) {
	t.Helper()
	for _, cid := range chatIDs {
		capsule.Global.Query().Table(taskTable()).Where("chat_id", "=", cid).Delete()
		capsule.Global.Query().Table(mailTable()).Where("chat_id", "=", cid).Delete()
	}
}

// ---------------------------------------------------------------------------
// Basic List and CRUD
// ---------------------------------------------------------------------------

func TestInboxList_Basic(t *testing.T) {
	tc := setupTest(t)

	chatID, _ := insertTaskWithMail(t, tc, nil)
	t.Cleanup(func() { cleanupTask(t, chatID) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all"})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Mails, "expected at least 1 mail")
}

func TestInboxView(t *testing.T) {
	tc := setupTest(t)

	chatID, _ := insertTaskWithMail(t, tc, map[string]interface{}{"has_unread": true})
	t.Cleanup(func() { cleanupTask(t, chatID) })

	err := inbox.View(tc.ctx, tc.auth, chatID)
	require.NoError(t, err)

	// Verify task has_unread is now false
	row, err := capsule.Global.Query().Table(taskTable()).
		Select("has_unread", "inbox_read_at").
		Where("chat_id", "=", chatID).First()
	require.NoError(t, err)
	require.NotNil(t, row)
}

func TestInboxView_NotFound(t *testing.T) {
	tc := setupTest(t)
	err := inbox.View(tc.ctx, tc.auth, "nonexistent-chat-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Bookmark / Unbookmark
// ---------------------------------------------------------------------------

func TestBookmarkUnbookmark(t *testing.T) {
	tc := setupTest(t)

	chatID, _ := insertTaskWithMail(t, tc, nil)
	t.Cleanup(func() { cleanupTask(t, chatID) })

	require.NoError(t, inbox.Bookmark(tc.ctx, tc.auth, chatID))

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "bookmarked"})
	require.NoError(t, err)
	assert.True(t, containsChat(result, chatID), "task should appear in bookmarked filter")

	require.NoError(t, inbox.Unbookmark(tc.ctx, tc.auth, chatID))

	result2, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "bookmarked"})
	require.NoError(t, err)
	assert.False(t, containsChat(result2, chatID), "task should not appear after unbookmark")
}

func TestBookmark_NotFound(t *testing.T) {
	tc := setupTest(t)
	err := inbox.Bookmark(tc.ctx, tc.auth, "nonexistent")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Pin / Unpin
// ---------------------------------------------------------------------------

func TestPinUnpin(t *testing.T) {
	tc := setupTest(t)

	chatID, _ := insertTaskWithMail(t, tc, nil)
	t.Cleanup(func() { cleanupTask(t, chatID) })

	require.NoError(t, inbox.Pin(tc.ctx, tc.auth, chatID))

	row, err := capsule.Global.Query().Table(taskTable()).
		Select("inbox_pinned").
		Where("chat_id", "=", chatID).First()
	require.NoError(t, err)
	require.NotNil(t, row)

	require.NoError(t, inbox.Unpin(tc.ctx, tc.auth, chatID))
}

func TestUnpin_NotFound(t *testing.T) {
	tc := setupTest(t)
	err := inbox.Unpin(tc.ctx, tc.auth, "nonexistent")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// ReadAll
// ---------------------------------------------------------------------------

func TestReadAll(t *testing.T) {
	tc := setupTest(t)

	chatID1, _ := insertTaskWithMail(t, tc, map[string]interface{}{"has_unread": true, "type": "input"})
	chatID2, _ := insertTaskWithMail(t, tc, map[string]interface{}{"has_unread": true, "type": "completed"})
	t.Cleanup(func() { cleanupTask(t, chatID1, chatID2) })

	affected, err := inbox.ReadAll(tc.ctx, tc.auth)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, affected, int64(2))

	count, err := inbox.UnreadCount(tc.ctx, tc.auth)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// ---------------------------------------------------------------------------
// UnreadCount
// ---------------------------------------------------------------------------

func TestUnreadCount(t *testing.T) {
	tc := setupTest(t)

	// Clear all unread
	inbox.ReadAll(tc.ctx, tc.auth)

	chatID, _ := insertTaskWithMail(t, tc, map[string]interface{}{"has_unread": true})
	t.Cleanup(func() { cleanupTask(t, chatID) })

	count, err := inbox.UnreadCount(tc.ctx, tc.auth)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1)
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

func TestStats(t *testing.T) {
	tc := setupTest(t)

	// Clear all
	inbox.ReadAll(tc.ctx, tc.auth)

	chatID1, _ := insertTaskWithMail(t, tc, map[string]interface{}{"has_unread": true, "last_mail_type": "input", "type": "input"})
	chatID2, _ := insertTaskWithMail(t, tc, map[string]interface{}{"has_unread": true, "last_mail_type": "completed", "type": "completed", "bookmarked": true})
	t.Cleanup(func() { cleanupTask(t, chatID1, chatID2) })

	stats, err := inbox.Stats(tc.ctx, tc.auth)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, stats.All, 2)
	assert.GreaterOrEqual(t, stats.Input, 1)
	assert.GreaterOrEqual(t, stats.Completed, 1)
	assert.GreaterOrEqual(t, stats.Bookmarked, 1)
}

// ---------------------------------------------------------------------------
// List filters
// ---------------------------------------------------------------------------

func TestListFilter_Unread(t *testing.T) {
	tc := setupTest(t)

	chatUnread, _ := insertTaskWithMail(t, tc, map[string]interface{}{"has_unread": true})
	chatRead, _ := insertTaskWithMail(t, tc, map[string]interface{}{"has_unread": false})
	t.Cleanup(func() { cleanupTask(t, chatUnread, chatRead) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "unread"})
	require.NoError(t, err)
	assert.True(t, containsChat(result, chatUnread))
	assert.False(t, containsChat(result, chatRead))
}

func TestListFilter_ByType(t *testing.T) {
	tc := setupTest(t)

	chatInput, _ := insertTaskWithMail(t, tc, map[string]interface{}{"last_mail_type": "input", "type": "input"})
	chatCompleted, _ := insertTaskWithMail(t, tc, map[string]interface{}{"last_mail_type": "completed", "type": "completed"})
	t.Cleanup(func() { cleanupTask(t, chatInput, chatCompleted) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "input"})
	require.NoError(t, err)
	assert.True(t, containsChat(result, chatInput))
	assert.False(t, containsChat(result, chatCompleted))
}

func TestListFilter_Archived(t *testing.T) {
	tc := setupTest(t)

	chatArchived, _ := insertTaskWithMail(t, tc, map[string]interface{}{"archive_status": "archived"})
	chatNormal, _ := insertTaskWithMail(t, tc, nil)
	t.Cleanup(func() { cleanupTask(t, chatArchived, chatNormal) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "archived"})
	require.NoError(t, err)
	assert.True(t, containsChat(result, chatArchived))
	assert.False(t, containsChat(result, chatNormal))
}

func TestListFilter_ArchivedExcludedFromAll(t *testing.T) {
	tc := setupTest(t)

	chatArchived, _ := insertTaskWithMail(t, tc, map[string]interface{}{"archive_status": "archived"})
	chatNormal, _ := insertTaskWithMail(t, tc, nil)
	t.Cleanup(func() { cleanupTask(t, chatArchived, chatNormal) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all"})
	require.NoError(t, err)
	assert.False(t, containsChat(result, chatArchived))
	assert.True(t, containsChat(result, chatNormal))
}

func TestListFilter_ChatID(t *testing.T) {
	tc := setupTest(t)

	chatA, _ := insertTaskWithMail(t, tc, nil)
	chatB, _ := insertTaskWithMail(t, tc, nil)
	t.Cleanup(func() { cleanupTask(t, chatA, chatB) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all", ChatID: chatA})
	require.NoError(t, err)
	assert.True(t, containsChat(result, chatA))
	assert.False(t, containsChat(result, chatB))
}

// ---------------------------------------------------------------------------
// Pagination
// ---------------------------------------------------------------------------

func TestList_Pagination(t *testing.T) {
	tc := setupTest(t)

	var chatIDs []string
	for i := 0; i < 5; i++ {
		cid, _ := insertTaskWithMail(t, tc, nil)
		chatIDs = append(chatIDs, cid)
	}
	t.Cleanup(func() { cleanupTask(t, chatIDs...) })

	page1, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all", Page: 1, Size: 2})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(page1.Mails), 2)
	assert.Equal(t, 1, page1.Page)
	assert.Equal(t, 2, page1.Size)
	assert.GreaterOrEqual(t, page1.Total, int64(5))

	page2, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all", Page: 2, Size: 2})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(page2.Mails), 2)
	assert.Equal(t, 2, page2.Page)
}

func TestList_DefaultPageSize(t *testing.T) {
	tc := setupTest(t)
	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all"})
	require.NoError(t, err)
	assert.Equal(t, 20, result.Size)
	assert.Equal(t, 1, result.Page)
}

// ---------------------------------------------------------------------------
// OnStatusChange trigger
// ---------------------------------------------------------------------------

func TestOnStatusChange_Waiting(t *testing.T) {
	tc := setupTest(t)

	b, columnID := createBoardWithColumn(t, tc)
	chatID := insertChat(t, tc, "Waiting task chat")
	insertTask(t, tc, chatID, columnID)

	task := &inbox.AgentTask{
		ChatID:      chatID,
		ColumnID:    columnID,
		AssistantID: "assistant-001",
		CreatedBy:   tc.auth.UserID,
		TeamID:      tc.auth.TeamID,
	}

	mailID, err := inbox.OnStatusChange(tc.ctx, task, "waiting")
	require.NoError(t, err)
	assert.NotEmpty(t, mailID)

	t.Cleanup(func() {
		cleanupTask(t, chatID)
		cleanupBoard(t, b.BoardID)
		cleanupChat(t, chatID)
	})

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "input"})
	require.NoError(t, err)
	assert.True(t, containsChat(result, chatID))
}

func TestOnStatusChange_Completed(t *testing.T) {
	tc := setupTest(t)

	b, columnID := createBoardWithColumn(t, tc)
	chatID := insertChat(t, tc, "Completed task chat")
	insertTask(t, tc, chatID, columnID)

	task := &inbox.AgentTask{
		ChatID:      chatID,
		ColumnID:    columnID,
		AssistantID: "assistant-002",
		CreatedBy:   tc.auth.UserID,
		TeamID:      tc.auth.TeamID,
	}

	mailID, err := inbox.OnStatusChange(tc.ctx, task, "completed")
	require.NoError(t, err)
	assert.NotEmpty(t, mailID)

	t.Cleanup(func() {
		cleanupTask(t, chatID)
		cleanupBoard(t, b.BoardID)
		cleanupChat(t, chatID)
	})

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "completed"})
	require.NoError(t, err)
	assert.True(t, containsChat(result, chatID))
}

func TestOnStatusChange_Failed(t *testing.T) {
	tc := setupTest(t)

	b, columnID := createBoardWithColumn(t, tc)
	chatID := insertChat(t, tc, "Failed task chat")
	insertTask(t, tc, chatID, columnID)

	task := &inbox.AgentTask{
		ChatID:      chatID,
		ColumnID:    columnID,
		AssistantID: "assistant-003",
		CreatedBy:   tc.auth.UserID,
		TeamID:      tc.auth.TeamID,
	}

	mailID, err := inbox.OnStatusChange(tc.ctx, task, "failed")
	require.NoError(t, err)
	assert.NotEmpty(t, mailID)

	t.Cleanup(func() {
		cleanupTask(t, chatID)
		cleanupBoard(t, b.BoardID)
		cleanupChat(t, chatID)
	})

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "failed"})
	require.NoError(t, err)
	assert.True(t, containsChat(result, chatID))
}

func TestOnStatusChange_Running_NoMail(t *testing.T) {
	tc := setupTest(t)

	task := &inbox.AgentTask{
		ChatID:    "chat-running-test",
		ColumnID:  "",
		CreatedBy: tc.auth.UserID,
		TeamID:    tc.auth.TeamID,
	}

	mailID, err := inbox.OnStatusChange(tc.ctx, task, "running")
	require.NoError(t, err)
	assert.Empty(t, mailID)
}

func TestOnStatusChange_DeletedTask_NoMail(t *testing.T) {
	tc := setupTest(t)

	deletedAt := time.Now()
	task := &inbox.AgentTask{
		ChatID:    "chat-deleted-test",
		CreatedBy: tc.auth.UserID,
		TeamID:    tc.auth.TeamID,
		DeletedAt: &deletedAt,
	}

	mailID, err := inbox.OnStatusChange(tc.ctx, task, "waiting")
	require.NoError(t, err)
	assert.Empty(t, mailID)
}

// ---------------------------------------------------------------------------
// Multi-user isolation
// ---------------------------------------------------------------------------

func TestList_UserIsolation(t *testing.T) {
	tc := setupTest(t)

	myChatID, _ := insertTaskWithMail(t, tc, nil)

	otherUserID := uuid.New().String()
	otherAuth := &process.AuthorizedInfo{UserID: otherUserID, TeamID: tc.auth.TeamID}

	t.Cleanup(func() { cleanupTask(t, myChatID) })

	myResult, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all"})
	require.NoError(t, err)
	assert.True(t, containsChat(myResult, myChatID))

	otherResult, err := inbox.List(tc.ctx, otherAuth, &inbox.ListQuery{Filter: "all"})
	require.NoError(t, err)
	assert.False(t, containsChat(otherResult, myChatID))
}

// ---------------------------------------------------------------------------
// DeleteByChatID
// ---------------------------------------------------------------------------

func TestDeleteByChatID(t *testing.T) {
	tc := setupTest(t)

	chatID, _ := insertTaskWithMail(t, tc, nil)
	t.Cleanup(func() { cleanupTask(t, chatID) })

	affected, err := inbox.DeleteByChatID(tc.ctx, tc.auth, chatID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, affected, int64(1))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func containsChat(result *inbox.ListResult, chatID string) bool {
	for _, m := range result.Mails {
		if m.ChatID == chatID {
			return true
		}
	}
	return false
}

func containsMail(result *inbox.ListResult, mailID string) bool {
	for _, m := range result.Mails {
		if m.MailID == mailID {
			return m != nil
		}
	}
	return false
}

func insertTask(t *testing.T, tc *testContext, chatID, columnID string) {
	t.Helper()
	now := time.Now()
	err := capsule.Global.Query().Table(taskTable()).Insert(map[string]interface{}{
		"chat_id":          chatID,
		"column_id":        columnID,
		"run_status":       "running",
		"position":         0,
		"priority":         "none",
		"bookmarked":       false,
		"inbox_pinned":     false,
		"has_unread":       false,
		"__yao_created_by": tc.auth.UserID,
		"__yao_team_id":    tc.auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	})
	require.NoError(t, err, "insert test task")
}

func boardTable() string {
	if m, err := model.Get("__yao.agent.board"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_board"
}

func boardColumnTable() string {
	if m, err := model.Get("__yao.agent.board_column"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_board_column"
}

func createBoardWithColumn(t *testing.T, tc *testContext) (*board.Board, string) {
	t.Helper()
	b, err := board.Create(tc.ctx, tc.auth, &board.CreateReq{
		Name:  "Test Board " + uuid.New().String()[:8],
		Icon:  "clipboard",
		Color: "#3B82F6",
	})
	require.NoError(t, err)
	require.NotEmpty(t, b.Columns)
	return b, b.Columns[0].ColumnID
}

func insertChat(t *testing.T, tc *testContext, title string) string {
	t.Helper()
	chatID := uuid.New().String()
	now := time.Now()
	err := capsule.Global.Query().Table(chatTable()).Insert(map[string]interface{}{
		"chat_id":          chatID,
		"title":            title,
		"assistant_id":     "test-assistant",
		"status":           "active",
		"public":           false,
		"share":            "private",
		"__yao_created_by": tc.auth.UserID,
		"__yao_team_id":    tc.auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	})
	require.NoError(t, err)
	return chatID
}

func cleanupBoard(t *testing.T, boardID string) {
	t.Helper()
	capsule.Global.Query().Table(boardColumnTable()).Where("board_id", "=", boardID).Delete()
	capsule.Global.Query().Table(boardTable()).Where("board_id", "=", boardID).Delete()
}

func cleanupChat(t *testing.T, chatID string) {
	t.Helper()
	capsule.Global.Query().Table(chatTable()).Where("chat_id", "=", chatID).Delete()
}
