//go:build integration

package task_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/board"
	"github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func tableTask() string { return task.ExportTableTask() }
func tableChat() string { return task.ExportTableChat() }

func TestArchive_SetsArchiveStatus(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Archive Test Board", Icon: "material-test", Color: "#3B82F6",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Archive Test Task",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	err = task.Archive(ctx, auth, created.ChatID)
	require.NoError(t, err)

	got, err := task.Get(ctx, auth, created.ChatID)
	require.NoError(t, err)
	assert.Equal(t, "archived", got.ArchiveStatus)
	assert.Equal(t, "pending", got.RunStatus, "run_status should be preserved")
}

func TestArchive_Idempotent(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Archive Idempotent Board", Icon: "material-test", Color: "#3B82F6",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Idempotent Task",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	err = task.Archive(ctx, auth, created.ChatID)
	require.NoError(t, err)

	err = task.Archive(ctx, auth, created.ChatID)
	require.NoError(t, err)
}

func TestArchive_PreservesRunStatus(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Preserve RunStatus Board", Icon: "material-test", Color: "#3B82F6",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Preserve RunStatus Task",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	// Set run_status to 'running' before archive
	_, err = capsule.Global.Query().Table(tableTask()).
		Where("chat_id", "=", created.ChatID).
		Update(map[string]interface{}{"run_status": "running"})
	require.NoError(t, err)

	err = task.Archive(ctx, auth, created.ChatID)
	require.NoError(t, err)

	got, err := task.Get(ctx, auth, created.ChatID)
	require.NoError(t, err)
	assert.Equal(t, "archived", got.ArchiveStatus)
	assert.Equal(t, "running", got.RunStatus, "run_status must not change on archive")
}

func TestArchive_PreservesColumnID(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Preserve Column Board", Icon: "material-test", Color: "#3B82F6",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Preserve Column Task",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	err = task.Archive(ctx, auth, created.ChatID)
	require.NoError(t, err)

	got, err := task.Get(ctx, auth, created.ChatID)
	require.NoError(t, err)
	assert.Equal(t, "archived", got.ArchiveStatus)
	assert.NotNil(t, got.ColumnID)
	assert.Equal(t, colID, *got.ColumnID, "column_id must not change on archive")
}

func TestArchive_ChatStatusUnchanged(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Chat Status Board", Icon: "material-test", Color: "#3B82F6",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Chat Status Task",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	err = task.Archive(ctx, auth, created.ChatID)
	require.NoError(t, err)

	row, err := capsule.Global.Query().Table(tableChat()).
		Select("status").
		Where("chat_id", "=", created.ChatID).
		First()
	require.NoError(t, err)
	require.NotNil(t, row)
	status, _ := row["status"].(string)
	assert.Equal(t, "active", status, "chat status must remain 'active' after archive")
}

func TestUnarchive_ClearsArchiveStatus(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Unarchive Test Board", Icon: "material-test", Color: "#3B82F6",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Unarchive Test Task",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	err = task.Archive(ctx, auth, created.ChatID)
	require.NoError(t, err)

	err = task.Unarchive(ctx, auth, created.ChatID, colID)
	require.NoError(t, err)

	got, err := task.Get(ctx, auth, created.ChatID)
	require.NoError(t, err)
	assert.Equal(t, "", got.ArchiveStatus, "archive_status should be cleared")
	assert.Equal(t, "pending", got.RunStatus, "run_status should be preserved")
	assert.NotNil(t, got.ColumnID)
	assert.Equal(t, colID, *got.ColumnID)
}

func TestUnarchive_ChangesColumn(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Unarchive Column Board", Icon: "material-test", Color: "#3B82F6",
	})
	require.NoError(t, err)
	require.True(t, len(b.Columns) >= 1, "board should have at least 1 column")
	colID1 := b.Columns[0].ColumnID

	col2, err := board.CreateColumn(ctx, auth, b.BoardID, &board.ColumnReq{
		Name: "Second Column",
	})
	require.NoError(t, err)
	colID2 := col2.ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Unarchive Column Task",
		AssistantID: "asst-test-001",
		ColumnID:    colID1,
	})
	require.NoError(t, err)

	err = task.Archive(ctx, auth, created.ChatID)
	require.NoError(t, err)

	err = task.Unarchive(ctx, auth, created.ChatID, colID2)
	require.NoError(t, err)

	got, err := task.Get(ctx, auth, created.ChatID)
	require.NoError(t, err)
	assert.Equal(t, "", got.ArchiveStatus)
	assert.NotNil(t, got.ColumnID)
	assert.Equal(t, colID2, *got.ColumnID, "column_id should be updated to new column")
}

func TestUnarchive_NonArchivedTaskFails(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Non-archived Board", Icon: "material-test", Color: "#3B82F6",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Non-archived Task",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	err = task.Unarchive(ctx, auth, created.ChatID, colID)
	assert.Error(t, err, "should fail for non-archived task")
	assert.Contains(t, err.Error(), "not archived")
}

func TestList_ExcludesArchived(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "List Filter Board", Icon: "material-test", Color: "#3B82F6",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "To Be Archived",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	err = task.Archive(ctx, auth, created.ChatID)
	require.NoError(t, err)

	result, err := task.List(ctx, auth, &task.ListQuery{PageSize: 100})
	require.NoError(t, err)
	for _, item := range result.Tasks {
		assert.NotEqual(t, created.ChatID, item.ChatID, "archived task should not appear in default list")
	}
}

func TestList_IncludesArchivedWhenFiltered(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "List Include Board", Icon: "material-test", Color: "#3B82F6",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Archived For Filter",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	err = task.Archive(ctx, auth, created.ChatID)
	require.NoError(t, err)

	result, err := task.List(ctx, auth, &task.ListQuery{PageSize: 100, ArchiveStatus: "archived"})
	require.NoError(t, err)

	found := false
	for _, item := range result.Tasks {
		if item.ChatID == created.ChatID {
			found = true
			break
		}
	}
	assert.True(t, found, "archived task should appear when ArchiveStatus filter is used")
}
