//go:build integration

package task_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/board"
	"github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestConfig_SetAndGet(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Config Test Board", Icon: "material-settings", Color: "#10B981",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title: "Config Task", AssistantID: "asst-config-001", ColumnID: colID,
	})
	require.NoError(t, err)
	chatID := created.ChatID

	t.Run("GetConfig_default", func(t *testing.T) {
		cfg, err := task.GetConfig(ctx, auth, chatID)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.NotNil(t, cfg.Setting)
	})

	t.Run("SetConfig_basic_fields", func(t *testing.T) {
		runner := "docker"
		model := "gpt-4o"
		timeout := "30m"
		maxTurns := 10

		err := task.SetConfig(ctx, auth, chatID, &task.ConfigReq{
			Runner:   &runner,
			Model:    &model,
			Timeout:  &timeout,
			MaxTurns: &maxTurns,
			Skills:   []string{"code-review", "testing"},
		})
		require.NoError(t, err)

		cfg, err := task.GetConfig(ctx, auth, chatID)
		require.NoError(t, err)
		assert.Equal(t, "docker", cfg.Setting.Runner)
		assert.Equal(t, "gpt-4o", cfg.Setting.Model)
		assert.Equal(t, "30m", cfg.Setting.Timeout)
		assert.Equal(t, 10, cfg.Setting.MaxTurns)
		assert.Contains(t, cfg.Setting.Skills, "code-review")
		assert.Contains(t, cfg.Setting.Skills, "testing")
	})

	t.Run("SetConfig_with_schedule", func(t *testing.T) {
		err := task.SetConfig(ctx, auth, chatID, &task.ConfigReq{
			Schedule: &task.ScheduleConfig{
				Enabled:       true,
				Mode:          "interval",
				IntervalValue: 30,
				IntervalUnit:  "minutes",
				Timezone:      "Asia/Shanghai",
			},
		})
		require.NoError(t, err)

		cfg, err := task.GetConfig(ctx, auth, chatID)
		require.NoError(t, err)
		require.NotNil(t, cfg.Setting.Schedule)
		assert.True(t, cfg.Setting.Schedule.Enabled)
		assert.Equal(t, "interval", cfg.Setting.Schedule.Mode)
		assert.Equal(t, 30, cfg.Setting.Schedule.IntervalValue)
		assert.Equal(t, "minutes", cfg.Setting.Schedule.IntervalUnit)
		assert.Equal(t, "Asia/Shanghai", cfg.Setting.Schedule.Timezone)
	})

	t.Run("SetConfig_with_secrets", func(t *testing.T) {
		apiKey := "sk-test-123"
		dbURL := "postgres://localhost/test"
		err := task.SetConfig(ctx, auth, chatID, &task.ConfigReq{
			Secrets: map[string]*string{
				"API_KEY": &apiKey,
				"DB_URL":  &dbURL,
			},
		})
		require.NoError(t, err)

		cfg, err := task.GetConfig(ctx, auth, chatID)
		require.NoError(t, err)
		assert.Equal(t, "sk-test-123", cfg.Setting.Secrets["API_KEY"])
		assert.Equal(t, "postgres://localhost/test", cfg.Setting.Secrets["DB_URL"])
	})

	t.Run("SetConfig_with_services", func(t *testing.T) {
		err := task.SetConfig(ctx, auth, chatID, &task.ConfigReq{
			Services: []task.ServiceDecl{
				{Name: "web", Port: 8080, Protocol: "http", Public: true},
				{Name: "db", Port: 5432, Protocol: "tcp", Public: false},
			},
		})
		require.NoError(t, err)

		cfg, err := task.GetConfig(ctx, auth, chatID)
		require.NoError(t, err)
		require.Len(t, cfg.Setting.Services, 2)
		assert.Equal(t, "web", cfg.Setting.Services[0].Name)
		assert.Equal(t, 8080, cfg.Setting.Services[0].Port)
		assert.True(t, cfg.Setting.Services[0].Public)
	})

	t.Run("SetConfig_overwrite_previous", func(t *testing.T) {
		newModel := "claude-4"
		err := task.SetConfig(ctx, auth, chatID, &task.ConfigReq{
			Model: &newModel,
		})
		require.NoError(t, err)

		cfg, err := task.GetConfig(ctx, auth, chatID)
		require.NoError(t, err)
		assert.Equal(t, "claude-4", cfg.Setting.Model)
	})
}

func TestMove_CrossColumn(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Move Test Board", Icon: "material-swap", Color: "#F59E0B",
	})
	require.NoError(t, err)
	col1ID := b.Columns[0].ColumnID

	col2, err := board.CreateColumn(ctx, auth, b.BoardID, &board.ColumnReq{
		Name: "In Progress", Icon: "material-play",
	})
	require.NoError(t, err)
	col2ID := col2.ColumnID

	col3, err := board.CreateColumn(ctx, auth, b.BoardID, &board.ColumnReq{
		Name: "Done", Icon: "material-check",
	})
	require.NoError(t, err)
	col3ID := col3.ColumnID

	task1, err := task.Create(ctx, auth, &task.CreateReq{
		Title: "Move Task 1", AssistantID: "asst-move-001", ColumnID: col1ID,
	})
	require.NoError(t, err)

	task2, err := task.Create(ctx, auth, &task.CreateReq{
		Title: "Move Task 2", AssistantID: "asst-move-001", ColumnID: col1ID,
	})
	require.NoError(t, err)

	t.Run("move_to_second_column", func(t *testing.T) {
		err := task.Move(ctx, auth, task1.ChatID, &task.MoveReq{
			ColumnID: col2ID,
			Position: 0,
		})
		require.NoError(t, err)

		got, err := task.Get(ctx, auth, task1.ChatID)
		require.NoError(t, err)
		require.NotNil(t, got.ColumnID)
		assert.Equal(t, col2ID, *got.ColumnID)
		assert.Equal(t, 0, got.Position)
	})

	t.Run("move_to_third_column", func(t *testing.T) {
		err := task.Move(ctx, auth, task2.ChatID, &task.MoveReq{
			ColumnID: col3ID,
			Position: 0,
		})
		require.NoError(t, err)

		got, err := task.Get(ctx, auth, task2.ChatID)
		require.NoError(t, err)
		require.NotNil(t, got.ColumnID)
		assert.Equal(t, col3ID, *got.ColumnID)
	})

	t.Run("move_same_position_noop", func(t *testing.T) {
		got, err := task.Get(ctx, auth, task1.ChatID)
		require.NoError(t, err)
		colID := ""
		if got.ColumnID != nil {
			colID = *got.ColumnID
		}
		err = task.Move(ctx, auth, task1.ChatID, &task.MoveReq{
			ColumnID: colID,
			Position: got.Position,
		})
		assert.NoError(t, err)
	})
}

func TestMove_SameColumnReposition(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Reposition Board", Icon: "material-sort", Color: "#8B5CF6",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	t1, err := task.Create(ctx, auth, &task.CreateReq{
		Title: "Pos Task A", AssistantID: "asst-pos-001", ColumnID: colID,
	})
	require.NoError(t, err)

	t2, err := task.Create(ctx, auth, &task.CreateReq{
		Title: "Pos Task B", AssistantID: "asst-pos-001", ColumnID: colID,
	})
	require.NoError(t, err)

	t3, err := task.Create(ctx, auth, &task.CreateReq{
		Title: "Pos Task C", AssistantID: "asst-pos-001", ColumnID: colID,
	})
	require.NoError(t, err)

	t.Run("move_last_to_first", func(t *testing.T) {
		got3, err := task.Get(ctx, auth, t3.ChatID)
		require.NoError(t, err)
		origPos := got3.Position

		err = task.Move(ctx, auth, t3.ChatID, &task.MoveReq{
			ColumnID: colID,
			Position: 0,
		})
		require.NoError(t, err)

		got3After, err := task.Get(ctx, auth, t3.ChatID)
		require.NoError(t, err)
		assert.Equal(t, 0, got3After.Position)
		assert.NotEqual(t, origPos, got3After.Position)
	})

	t.Run("move_first_to_last", func(t *testing.T) {
		err := task.Move(ctx, auth, t1.ChatID, &task.MoveReq{
			ColumnID: colID,
			Position: 2,
		})
		require.NoError(t, err)

		got1, err := task.Get(ctx, auth, t1.ChatID)
		require.NoError(t, err)
		assert.Equal(t, 2, got1.Position)
	})

	t.Run("no_op_same_position", func(t *testing.T) {
		got2, err := task.Get(ctx, auth, t2.ChatID)
		require.NoError(t, err)
		currentPos := got2.Position

		err = task.Move(ctx, auth, t2.ChatID, &task.MoveReq{
			ColumnID: colID,
			Position: currentPos,
		})
		require.NoError(t, err)

		got2After, err := task.Get(ctx, auth, t2.ChatID)
		require.NoError(t, err)
		assert.Equal(t, currentPos, got2After.Position)
	})
}

func TestList_Filters(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Filter Board", Icon: "material-filter", Color: "#EF4444",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	_, err = task.Create(ctx, auth, &task.CreateReq{
		Title: "Alpha Task", AssistantID: "asst-filter-alpha", ColumnID: colID,
	})
	require.NoError(t, err)

	_, err = task.Create(ctx, auth, &task.CreateReq{
		Title: "Beta Task", AssistantID: "asst-filter-beta", ColumnID: colID,
	})
	require.NoError(t, err)

	_, err = task.Create(ctx, auth, &task.CreateReq{
		Title: "Gamma Task", AssistantID: "asst-filter-alpha", ColumnID: colID,
	})
	require.NoError(t, err)

	t.Run("filter_by_assistant_id", func(t *testing.T) {
		result, err := task.List(ctx, auth, &task.ListQuery{
			AssistantID: "asst-filter-alpha",
			PageSize:    50,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, int(result.Total), 2)
		for _, tsk := range result.Tasks {
			assert.Equal(t, "asst-filter-alpha", tsk.AssistantID)
		}
	})

	t.Run("filter_by_run_status", func(t *testing.T) {
		result, err := task.List(ctx, auth, &task.ListQuery{
			RunStatus: "pending",
			PageSize:  100,
		})
		require.NoError(t, err)
		for _, tsk := range result.Tasks {
			assert.Equal(t, "pending", tsk.RunStatus)
		}
	})

	t.Run("filter_by_board_id", func(t *testing.T) {
		result, err := task.List(ctx, auth, &task.ListQuery{
			BoardID:  b.BoardID,
			PageSize: 50,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, int(result.Total), 3)
	})

	t.Run("pagination", func(t *testing.T) {
		page1, err := task.List(ctx, auth, &task.ListQuery{
			BoardID:  b.BoardID,
			Page:     1,
			PageSize: 2,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(page1.Tasks), 2)
		assert.Equal(t, 1, page1.Page)
		assert.Equal(t, 2, page1.PageSize)

		if page1.Total > 2 {
			page2, err := task.List(ctx, auth, &task.ListQuery{
				BoardID:  b.BoardID,
				Page:     2,
				PageSize: 2,
			})
			require.NoError(t, err)
			assert.Equal(t, 2, page2.Page)
			assert.NotEmpty(t, page2.Tasks)
		}
	})

	t.Run("empty_result", func(t *testing.T) {
		result, err := task.List(ctx, auth, &task.ListQuery{
			AssistantID: "asst-nonexistent-id",
			PageSize:    10,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Empty(t, result.Tasks)
	})

	t.Run("default_page_size", func(t *testing.T) {
		result, err := task.List(ctx, auth, &task.ListQuery{
			BoardID: b.BoardID,
		})
		require.NoError(t, err)
		assert.Equal(t, 50, result.PageSize)
	})
}

func TestSetPriority(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Priority Board", Icon: "material-flag", Color: "#06B6D4",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title: "Priority Task", AssistantID: "asst-priority-001", ColumnID: colID,
	})
	require.NoError(t, err)
	chatID := created.ChatID

	t.Run("set_high_priority", func(t *testing.T) {
		err := task.SetPriority(ctx, auth, chatID, 900)
		require.NoError(t, err)
	})

	t.Run("set_low_priority", func(t *testing.T) {
		err := task.SetPriority(ctx, auth, chatID, 100)
		require.NoError(t, err)
	})

	t.Run("set_default_priority", func(t *testing.T) {
		err := task.SetPriority(ctx, auth, chatID, 500)
		require.NoError(t, err)
	})

	t.Run("set_max_priority", func(t *testing.T) {
		err := task.SetPriority(ctx, auth, chatID, 9999)
		require.NoError(t, err)
	})
}
