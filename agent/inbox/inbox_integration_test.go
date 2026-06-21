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

func insertMail(t *testing.T, tc *testContext, overrides map[string]interface{}) string {
	t.Helper()
	mailID := uuid.New().String()
	now := time.Now()
	row := map[string]interface{}{
		"mail_id":          mailID,
		"type":             "input",
		"priority":         "high",
		"title":            "Test Mail",
		"body":             "Test body",
		"chat_id":          "chat-" + mailID[:8],
		"read":             false,
		"archived":         false,
		"starred":          false,
		"pinned":           false,
		"__yao_created_by": tc.auth.UserID,
		"__yao_team_id":    tc.auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	}
	for k, v := range overrides {
		row[k] = v
	}
	err := capsule.Global.Query().Table(mailTable()).Insert(row)
	require.NoError(t, err, "insert test mail")
	return mailID
}

func cleanupMails(t *testing.T, mailIDs ...string) {
	t.Helper()
	for _, id := range mailIDs {
		capsule.Global.Query().Table(mailTable()).Where("mail_id", "=", id).Delete()
	}
}

// ---------------------------------------------------------------------------
// Basic CRUD operations
// ---------------------------------------------------------------------------

func TestInboxOperations(t *testing.T) {
	tc := setupTest(t)

	mailID := insertMail(t, tc, map[string]interface{}{
		"title": "Test: Needs Input",
		"body":  "Please provide input",
	})
	t.Cleanup(func() { cleanupMails(t, mailID) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all"})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Mails, "expected at least 1 mail")

	counts, err := inbox.UnreadCount(tc.ctx, tc.auth)
	require.NoError(t, err)
	assert.Greater(t, counts.Total, 0, "expected non-zero unread total")
	assert.Greater(t, counts.Input, 0, "expected non-zero input count")

	require.NoError(t, inbox.Read(tc.ctx, tc.auth, mailID))

	countsAfter, err := inbox.UnreadCount(tc.ctx, tc.auth)
	require.NoError(t, err)
	assert.Less(t, countsAfter.Total, counts.Total, "unread count should decrease after marking as read")

	require.NoError(t, inbox.Star(tc.ctx, tc.auth, mailID))
	require.NoError(t, inbox.Pin(tc.ctx, tc.auth, mailID))
	require.NoError(t, inbox.Archive(tc.ctx, tc.auth, mailID))
}

// ---------------------------------------------------------------------------
// Unstar
// ---------------------------------------------------------------------------

func TestUnstar(t *testing.T) {
	tc := setupTest(t)

	mailID := insertMail(t, tc, nil)
	t.Cleanup(func() { cleanupMails(t, mailID) })

	require.NoError(t, inbox.Star(tc.ctx, tc.auth, mailID))

	starred, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "starred"})
	require.NoError(t, err)
	assert.True(t, containsMail(starred, mailID), "mail should appear in starred filter after Star()")

	require.NoError(t, inbox.Unstar(tc.ctx, tc.auth, mailID))

	starred2, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "starred"})
	require.NoError(t, err)
	assert.False(t, containsMail(starred2, mailID), "mail should not appear in starred filter after Unstar()")
}

func TestUnstar_NotFound(t *testing.T) {
	tc := setupTest(t)
	err := inbox.Unstar(tc.ctx, tc.auth, "nonexistent-mail-id")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Unpin
// ---------------------------------------------------------------------------

func TestUnpin(t *testing.T) {
	tc := setupTest(t)

	mailID := insertMail(t, tc, nil)
	t.Cleanup(func() { cleanupMails(t, mailID) })

	require.NoError(t, inbox.Pin(tc.ctx, tc.auth, mailID))

	row, err := capsule.Global.Query().Table(mailTable()).
		Where("mail_id", "=", mailID).First()
	require.NoError(t, err)
	assert.NotNil(t, row)

	require.NoError(t, inbox.Unpin(tc.ctx, tc.auth, mailID))

	row2, err := capsule.Global.Query().Table(mailTable()).
		Select("pinned").
		Where("mail_id", "=", mailID).First()
	require.NoError(t, err)
	require.NotNil(t, row2)
}

func TestUnpin_NotFound(t *testing.T) {
	tc := setupTest(t)
	err := inbox.Unpin(tc.ctx, tc.auth, "nonexistent-mail-id")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// ReadAll
// ---------------------------------------------------------------------------

func TestReadAll_AllTypes(t *testing.T) {
	tc := setupTest(t)

	id1 := insertMail(t, tc, map[string]interface{}{"type": "input", "read": false})
	id2 := insertMail(t, tc, map[string]interface{}{"type": "completed", "read": false})
	id3 := insertMail(t, tc, map[string]interface{}{"type": "failed", "read": false})
	t.Cleanup(func() { cleanupMails(t, id1, id2, id3) })

	countsBefore, err := inbox.UnreadCount(tc.ctx, tc.auth)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, countsBefore.Total, 3)

	affected, err := inbox.ReadAll(tc.ctx, tc.auth, "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, affected, int64(3), "ReadAll should mark at least 3 unread mails as read")

	countsAfter, err := inbox.UnreadCount(tc.ctx, tc.auth)
	require.NoError(t, err)
	assert.Equal(t, 0, countsAfter.Total, "all unread should be zero after ReadAll")
}

func TestReadAll_FilterByType(t *testing.T) {
	tc := setupTest(t)

	id1 := insertMail(t, tc, map[string]interface{}{"type": "input", "read": false})
	id2 := insertMail(t, tc, map[string]interface{}{"type": "completed", "read": false})
	id3 := insertMail(t, tc, map[string]interface{}{"type": "failed", "read": false})
	t.Cleanup(func() { cleanupMails(t, id1, id2, id3) })

	affected, err := inbox.ReadAll(tc.ctx, tc.auth, "input")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, affected, int64(1), "ReadAll('input') should mark at least 1 mail as read")

	counts, err := inbox.UnreadCount(tc.ctx, tc.auth)
	require.NoError(t, err)
	assert.Equal(t, 0, counts.Input, "input unread should be zero after ReadAll('input')")
	assert.Greater(t, counts.Total, 0, "other types should still be unread")
}

func TestReadAll_AlreadyRead(t *testing.T) {
	tc := setupTest(t)

	id1 := insertMail(t, tc, map[string]interface{}{"read": true})
	t.Cleanup(func() { cleanupMails(t, id1) })

	affected, err := inbox.ReadAll(tc.ctx, tc.auth, "")
	require.NoError(t, err)
	assert.Equal(t, int64(0), affected, "ReadAll on already-read mails should affect 0 rows")
}

func TestReadAll_SkipsArchived(t *testing.T) {
	tc := setupTest(t)

	id1 := insertMail(t, tc, map[string]interface{}{"read": false, "archived": true})
	t.Cleanup(func() { cleanupMails(t, id1) })

	affected, err := inbox.ReadAll(tc.ctx, tc.auth, "")
	require.NoError(t, err)
	assert.Equal(t, int64(0), affected, "ReadAll should skip archived mails")
}

// ---------------------------------------------------------------------------
// List filters
// ---------------------------------------------------------------------------

func TestListFilter_Unread(t *testing.T) {
	tc := setupTest(t)

	unreadID := insertMail(t, tc, map[string]interface{}{"read": false, "title": "unread-filter-test"})
	readID := insertMail(t, tc, map[string]interface{}{"read": true, "title": "read-filter-test"})
	t.Cleanup(func() { cleanupMails(t, unreadID, readID) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "unread"})
	require.NoError(t, err)
	assert.True(t, containsMail(result, unreadID), "unread mail should appear in unread filter")
	assert.False(t, containsMail(result, readID), "read mail should not appear in unread filter")
}

func TestListFilter_Starred(t *testing.T) {
	tc := setupTest(t)

	starredID := insertMail(t, tc, map[string]interface{}{"starred": true, "title": "starred-filter-test"})
	normalID := insertMail(t, tc, map[string]interface{}{"starred": false, "title": "normal-filter-test"})
	t.Cleanup(func() { cleanupMails(t, starredID, normalID) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "starred"})
	require.NoError(t, err)
	assert.True(t, containsMail(result, starredID), "starred mail should appear")
	assert.False(t, containsMail(result, normalID), "non-starred mail should not appear")
}

func TestListFilter_Input(t *testing.T) {
	tc := setupTest(t)

	inputID := insertMail(t, tc, map[string]interface{}{"type": "input"})
	completedID := insertMail(t, tc, map[string]interface{}{"type": "completed"})
	t.Cleanup(func() { cleanupMails(t, inputID, completedID) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "input"})
	require.NoError(t, err)
	assert.True(t, containsMail(result, inputID))
	assert.False(t, containsMail(result, completedID))
}

func TestListFilter_Completed(t *testing.T) {
	tc := setupTest(t)

	completedID := insertMail(t, tc, map[string]interface{}{"type": "completed"})
	inputID := insertMail(t, tc, map[string]interface{}{"type": "input"})
	t.Cleanup(func() { cleanupMails(t, completedID, inputID) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "completed"})
	require.NoError(t, err)
	assert.True(t, containsMail(result, completedID))
	assert.False(t, containsMail(result, inputID))
}

func TestListFilter_Failed(t *testing.T) {
	tc := setupTest(t)

	failedID := insertMail(t, tc, map[string]interface{}{"type": "failed"})
	inputID := insertMail(t, tc, map[string]interface{}{"type": "input"})
	t.Cleanup(func() { cleanupMails(t, failedID, inputID) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "failed"})
	require.NoError(t, err)
	assert.True(t, containsMail(result, failedID))
	assert.False(t, containsMail(result, inputID))
}

func TestListFilter_Archived(t *testing.T) {
	tc := setupTest(t)

	archivedID := insertMail(t, tc, map[string]interface{}{"archived": true})
	normalID := insertMail(t, tc, map[string]interface{}{"archived": false})
	t.Cleanup(func() { cleanupMails(t, archivedID, normalID) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "archived"})
	require.NoError(t, err)
	assert.True(t, containsMail(result, archivedID), "archived mail should appear in archived filter")
	assert.False(t, containsMail(result, normalID), "non-archived mail should not appear in archived filter")
}

func TestListFilter_Keyword(t *testing.T) {
	tc := setupTest(t)

	matchTitle := insertMail(t, tc, map[string]interface{}{"title": "UniqueAlphaKeyword search target", "body": "normal body"})
	matchBody := insertMail(t, tc, map[string]interface{}{"title": "normal title", "body": "UniqueAlphaKeyword in body"})
	noMatch := insertMail(t, tc, map[string]interface{}{"title": "nothing here", "body": "nothing here either"})
	t.Cleanup(func() { cleanupMails(t, matchTitle, matchBody, noMatch) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all", Keyword: "UniqueAlphaKeyword"})
	require.NoError(t, err)
	assert.True(t, containsMail(result, matchTitle), "mail with keyword in title should appear")
	assert.True(t, containsMail(result, matchBody), "mail with keyword in body should appear")
	assert.False(t, containsMail(result, noMatch), "mail without keyword should not appear")
}

func TestListFilter_KeywordCombinedWithType(t *testing.T) {
	tc := setupTest(t)

	match := insertMail(t, tc, map[string]interface{}{"type": "failed", "title": "BetaUniqueKW failure report"})
	wrongType := insertMail(t, tc, map[string]interface{}{"type": "input", "title": "BetaUniqueKW input item"})
	t.Cleanup(func() { cleanupMails(t, match, wrongType) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "failed", Keyword: "BetaUniqueKW"})
	require.NoError(t, err)
	assert.True(t, containsMail(result, match))
	assert.False(t, containsMail(result, wrongType))
}

func TestListFilter_ArchivedExcludedFromAll(t *testing.T) {
	tc := setupTest(t)

	archivedID := insertMail(t, tc, map[string]interface{}{"archived": true, "title": "archived-excl-test"})
	normalID := insertMail(t, tc, map[string]interface{}{"archived": false, "title": "normal-excl-test"})
	t.Cleanup(func() { cleanupMails(t, archivedID, normalID) })

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all"})
	require.NoError(t, err)
	assert.False(t, containsMail(result, archivedID), "archived mail should NOT appear in 'all' filter")
	assert.True(t, containsMail(result, normalID), "non-archived mail should appear in 'all' filter")
}

// ---------------------------------------------------------------------------
// List pagination
// ---------------------------------------------------------------------------

func TestList_Pagination(t *testing.T) {
	tc := setupTest(t)

	var ids []string
	for i := 0; i < 5; i++ {
		id := insertMail(t, tc, map[string]interface{}{"title": "pagination-test"})
		ids = append(ids, id)
	}
	t.Cleanup(func() { cleanupMails(t, ids...) })

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

	if len(page1.Mails) > 0 && len(page2.Mails) > 0 {
		assert.NotEqual(t, page1.Mails[0].MailID, page2.Mails[0].MailID, "pages should return different mails")
	}
}

func TestList_DefaultPageSize(t *testing.T) {
	tc := setupTest(t)

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all"})
	require.NoError(t, err)
	assert.Equal(t, 20, result.Size, "default page size should be 20")
	assert.Equal(t, 1, result.Page, "default page should be 1")
}

// ---------------------------------------------------------------------------
// Read edge cases
// ---------------------------------------------------------------------------

func TestRead_NotFound(t *testing.T) {
	tc := setupTest(t)
	err := inbox.Read(tc.ctx, tc.auth, "nonexistent-mail-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRead_Idempotent(t *testing.T) {
	tc := setupTest(t)

	mailID := insertMail(t, tc, map[string]interface{}{"read": false})
	t.Cleanup(func() { cleanupMails(t, mailID) })

	require.NoError(t, inbox.Read(tc.ctx, tc.auth, mailID))
	require.NoError(t, inbox.Read(tc.ctx, tc.auth, mailID))
}

// ---------------------------------------------------------------------------
// OnStatusChange trigger
// ---------------------------------------------------------------------------

func TestOnStatusChange_Waiting(t *testing.T) {
	tc := setupTest(t)

	b, columnID := createBoardWithColumn(t, tc)
	chatID := insertChat(t, tc, "Waiting task chat")

	task := &inbox.AgentTask{
		ChatID:      chatID,
		ColumnID:    columnID,
		AssistantID: "assistant-001",
		CreatedBy:   tc.auth.UserID,
		TeamID:      tc.auth.TeamID,
	}

	mailID, err := inbox.OnStatusChange(tc.ctx, task, "waiting")
	require.NoError(t, err)
	assert.NotEmpty(t, mailID, "waiting status should create a mail")

	t.Cleanup(func() {
		cleanupMails(t, mailID)
		cleanupBoard(t, b.BoardID)
		cleanupChat(t, chatID)
	})

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "input"})
	require.NoError(t, err)
	assert.True(t, containsMail(result, mailID), "waiting should create 'input' type mail")

	found := findMail(result, mailID)
	require.NotNil(t, found)
	assert.Equal(t, "input", found.Type)
	assert.Equal(t, "high", found.Priority)
	assert.Equal(t, chatID, found.ChatID)
	assert.Equal(t, "kanban", found.SourceType)
	assert.Equal(t, b.BoardID, found.SourceID)
	assert.Equal(t, b.Name, found.SourceName)
	assert.False(t, found.Read)
}

func TestOnStatusChange_Completed(t *testing.T) {
	tc := setupTest(t)

	b, columnID := createBoardWithColumn(t, tc)
	chatID := insertChat(t, tc, "Completed task chat")

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
		cleanupMails(t, mailID)
		cleanupBoard(t, b.BoardID)
		cleanupChat(t, chatID)
	})

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "completed"})
	require.NoError(t, err)
	assert.True(t, containsMail(result, mailID))

	found := findMail(result, mailID)
	require.NotNil(t, found)
	assert.Equal(t, "completed", found.Type)
	assert.Equal(t, "low", found.Priority)
}

func TestOnStatusChange_Failed(t *testing.T) {
	tc := setupTest(t)

	b, columnID := createBoardWithColumn(t, tc)
	chatID := insertChat(t, tc, "Failed task chat")

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
		cleanupMails(t, mailID)
		cleanupBoard(t, b.BoardID)
		cleanupChat(t, chatID)
	})

	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "failed"})
	require.NoError(t, err)
	assert.True(t, containsMail(result, mailID))

	found := findMail(result, mailID)
	require.NotNil(t, found)
	assert.Equal(t, "failed", found.Type)
	assert.Equal(t, "medium", found.Priority)
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
	assert.Empty(t, mailID, "running status should NOT create a mail")
}

func TestOnStatusChange_UnknownStatus_NoMail(t *testing.T) {
	tc := setupTest(t)

	task := &inbox.AgentTask{
		ChatID:    "chat-unknown-test",
		CreatedBy: tc.auth.UserID,
		TeamID:    tc.auth.TeamID,
	}

	mailID, err := inbox.OnStatusChange(tc.ctx, task, "paused")
	require.NoError(t, err)
	assert.Empty(t, mailID, "unknown status should NOT create a mail")
}

func TestOnStatusChange_DeletedTask_NoMail(t *testing.T) {
	tc := setupTest(t)

	deletedAt := time.Now()
	task := &inbox.AgentTask{
		ChatID:    "chat-deleted-test",
		ColumnID:  "",
		CreatedBy: tc.auth.UserID,
		TeamID:    tc.auth.TeamID,
		DeletedAt: &deletedAt,
	}

	mailID, err := inbox.OnStatusChange(tc.ctx, task, "waiting")
	require.NoError(t, err)
	assert.Empty(t, mailID, "deleted task should NOT create a mail")
}

func TestOnStatusChange_EmptyColumnID(t *testing.T) {
	tc := setupTest(t)

	chatID := insertChat(t, tc, "No column task")

	task := &inbox.AgentTask{
		ChatID:    chatID,
		ColumnID:  "",
		CreatedBy: tc.auth.UserID,
		TeamID:    tc.auth.TeamID,
	}

	mailID, err := inbox.OnStatusChange(tc.ctx, task, "completed")
	require.NoError(t, err)
	assert.NotEmpty(t, mailID, "should still create mail even with empty column_id")

	t.Cleanup(func() {
		cleanupMails(t, mailID)
		cleanupChat(t, chatID)
	})

	found := findMailByID(t, tc, mailID)
	require.NotNil(t, found)
	assert.Empty(t, found.SourceID, "source_id should be empty when column_id is empty")
	assert.Empty(t, found.SourceName, "source_name should be empty when column_id is empty")
}

func TestOnStatusChange_TitleFromChat(t *testing.T) {
	tc := setupTest(t)

	chatTitle := "My Important Chat"
	chatID := insertChat(t, tc, chatTitle)
	b, columnID := createBoardWithColumn(t, tc)

	task := &inbox.AgentTask{
		ChatID:    chatID,
		ColumnID:  columnID,
		CreatedBy: tc.auth.UserID,
		TeamID:    tc.auth.TeamID,
	}

	mailID, err := inbox.OnStatusChange(tc.ctx, task, "waiting")
	require.NoError(t, err)
	assert.NotEmpty(t, mailID)

	t.Cleanup(func() {
		cleanupMails(t, mailID)
		cleanupBoard(t, b.BoardID)
		cleanupChat(t, chatID)
	})

	found := findMailByID(t, tc, mailID)
	require.NotNil(t, found)
	assert.Contains(t, found.Title, chatTitle, "mail title should include the chat title")
}

// ---------------------------------------------------------------------------
// Multi-user isolation
// ---------------------------------------------------------------------------

func TestList_UserIsolation(t *testing.T) {
	tc := setupTest(t)

	otherUserID := uuid.New().String()
	otherAuth := &process.AuthorizedInfo{
		UserID: otherUserID,
		TeamID: tc.auth.TeamID,
	}

	myMailID := insertMail(t, tc, map[string]interface{}{"title": "my-isolation-mail"})
	otherMailID := uuid.New().String()
	now := time.Now()
	err := capsule.Global.Query().Table(mailTable()).Insert(map[string]interface{}{
		"mail_id":          otherMailID,
		"type":             "input",
		"priority":         "high",
		"title":            "other-user-mail",
		"body":             "belongs to other user",
		"chat_id":          "chat-other",
		"read":             false,
		"archived":         false,
		"starred":          false,
		"pinned":           false,
		"__yao_created_by": otherUserID,
		"__yao_team_id":    tc.auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	})
	require.NoError(t, err)

	t.Cleanup(func() { cleanupMails(t, myMailID, otherMailID) })

	myResult, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all"})
	require.NoError(t, err)
	assert.True(t, containsMail(myResult, myMailID))
	assert.False(t, containsMail(myResult, otherMailID), "should not see other user's mail")

	otherResult, err := inbox.List(tc.ctx, otherAuth, &inbox.ListQuery{Filter: "all"})
	require.NoError(t, err)
	assert.True(t, containsMail(otherResult, otherMailID))
	assert.False(t, containsMail(otherResult, myMailID), "other user should not see my mail")
}

// ---------------------------------------------------------------------------
// UnreadCount grouping
// ---------------------------------------------------------------------------

func TestUnreadCount_GroupsByType(t *testing.T) {
	tc := setupTest(t)

	// Mark all existing as read first
	inbox.ReadAll(tc.ctx, tc.auth, "")

	id1 := insertMail(t, tc, map[string]interface{}{"type": "input", "read": false})
	id2 := insertMail(t, tc, map[string]interface{}{"type": "input", "read": false})
	id3 := insertMail(t, tc, map[string]interface{}{"type": "completed", "read": false})
	id4 := insertMail(t, tc, map[string]interface{}{"type": "failed", "read": false})
	t.Cleanup(func() { cleanupMails(t, id1, id2, id3, id4) })

	counts, err := inbox.UnreadCount(tc.ctx, tc.auth)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, counts.Input, 2)
	assert.GreaterOrEqual(t, counts.Completed, 1)
	assert.GreaterOrEqual(t, counts.Failed, 1)
	assert.Equal(t, counts.Input+counts.Completed+counts.Failed, counts.Total)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func containsMail(result *inbox.ListResult, mailID string) bool {
	return findMail(result, mailID) != nil
}

func findMail(result *inbox.ListResult, mailID string) *inbox.AgentMail {
	for _, m := range result.Mails {
		if m.MailID == mailID {
			return m
		}
	}
	return nil
}

func findMailByID(t *testing.T, tc *testContext, mailID string) *inbox.AgentMail {
	t.Helper()
	result, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "all", Size: 100})
	require.NoError(t, err)
	if m := findMail(result, mailID); m != nil {
		return m
	}
	archivedResult, err := inbox.List(tc.ctx, tc.auth, &inbox.ListQuery{Filter: "archived", Size: 100})
	require.NoError(t, err)
	return findMail(archivedResult, mailID)
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
	require.NotEmpty(t, b.Columns, "board.Create should produce at least one default column")
	return b, b.Columns[0].ColumnID
}

func insertChat(t *testing.T, tc *testContext, title string) string {
	t.Helper()
	chatID := uuid.New().String()
	now := time.Now()
	err := capsule.Global.Query().Table(chatTable()).Insert(map[string]interface{}{
		"chat_id":          chatID,
		"title":            title,
		"__yao_created_by": tc.auth.UserID,
		"__yao_team_id":    tc.auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	})
	require.NoError(t, err, "insert test chat")
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
