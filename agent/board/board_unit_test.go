//go:build unit

package board_test

import (
	"context"
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/board"
)

// --- Templates loading ---

func TestTemplatesLoadStructure(t *testing.T) {
	templates, err := board.Templates(context.Background())
	if err != nil {
		t.Fatalf("Templates() failed: %v", err)
	}
	if len(templates) < 2 {
		t.Fatalf("expected at least 2 templates, got %d", len(templates))
	}
	for _, tmpl := range templates {
		if tmpl.ID == "" {
			t.Error("template ID is empty")
		}
		if tmpl.Name == "" {
			t.Error("template name is empty")
		}
		if len(tmpl.Columns) == 0 {
			t.Errorf("template %s has no columns", tmpl.ID)
		}
		for i, col := range tmpl.Columns {
			if col.Name == "" {
				t.Errorf("template %s column[%d] has empty name", tmpl.ID, i)
			}
			if col.Icon == "" {
				t.Errorf("template %s column[%d] has empty icon", tmpl.ID, i)
			}
			if col.Color == "" {
				t.Errorf("template %s column[%d] has empty color", tmpl.ID, i)
			}
		}
	}
}

func TestTemplatesHaveLocales(t *testing.T) {
	templates, err := board.Templates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range templates {
		if tmpl.Locales == nil || len(tmpl.Locales) == 0 {
			t.Errorf("template %s should have at least one locale", tmpl.ID)
		}
	}
}

func TestTemplatesDefaultIsEnglish(t *testing.T) {
	templates, err := board.Templates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range templates {
		if tmpl.ID == "kanban-basic" {
			if tmpl.Name != "Basic Kanban" {
				t.Errorf("default name = %q, want English", tmpl.Name)
			}
			if tmpl.Columns[0].Name != "To Do" {
				t.Errorf("default col[0] = %q, want English", tmpl.Columns[0].Name)
			}
		}
	}
}

func TestTemplatesWithZhCNLocale(t *testing.T) {
	templates, err := board.Templates(context.Background(), "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range templates {
		if tmpl.ID == "dev-workflow" {
			if tmpl.Name != "开发工作流" {
				t.Errorf("zh-CN name = %q, want 开发工作流", tmpl.Name)
			}
			if tmpl.Columns[0].Name != "待处理" {
				t.Errorf("zh-CN col[0] = %q, want 待处理", tmpl.Columns[0].Name)
			}
			if tmpl.Columns[2].Name != "已完成" {
				t.Errorf("zh-CN col[2] = %q, want 已完成", tmpl.Columns[2].Name)
			}
			return
		}
	}
	t.Fatal("dev-workflow not found")
}

func TestTemplatesWithUnknownLocale(t *testing.T) {
	templates, err := board.Templates(context.Background(), "fr")
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range templates {
		if tmpl.ID == "dev-workflow" {
			if tmpl.Name != "Dev Workflow" {
				t.Errorf("unknown locale should fallback to English, got %q", tmpl.Name)
			}
			return
		}
	}
}

// --- ResolvedName ---

func TestResolvedName(t *testing.T) {
	tmpl := &board.Template{
		Name: "Default",
		Locales: map[string]board.TemplateLocale{
			"zh-CN": {Name: "中文"},
			"ja":    {Name: "日本語"},
		},
	}
	tests := []struct {
		locale, want string
	}{
		{"", "Default"},
		{"zh-CN", "中文"},
		{"ja", "日本語"},
		{"fr", "Default"},
	}
	for _, tt := range tests {
		if got := tmpl.ResolvedName(tt.locale); got != tt.want {
			t.Errorf("ResolvedName(%q) = %q, want %q", tt.locale, got, tt.want)
		}
	}
}

func TestResolvedNameEmptyLocaleValue(t *testing.T) {
	tmpl := &board.Template{
		Name:    "Fallback",
		Locales: map[string]board.TemplateLocale{"zh-CN": {Name: ""}},
	}
	if got := tmpl.ResolvedName("zh-CN"); got != "Fallback" {
		t.Errorf("empty locale name should fallback, got %q", got)
	}
}

func TestResolvedNameNilLocales(t *testing.T) {
	tmpl := &board.Template{Name: "Only"}
	if got := tmpl.ResolvedName("en"); got != "Only" {
		t.Errorf("nil locales should fallback, got %q", got)
	}
}

// --- ResolvedColumnName ---

func TestResolvedColumnName(t *testing.T) {
	tmpl := &board.Template{
		Columns: []board.TemplateColumn{
			{Name: "To Do"}, {Name: "In Progress"}, {Name: "Done"},
		},
		Locales: map[string]board.TemplateLocale{
			"zh-CN": {Columns: []board.TemplateColumnName{
				{Name: "待办"}, {Name: "进行中"}, {Name: "完成"},
			}},
			"partial": {Columns: []board.TemplateColumnName{
				{Name: "First Only"},
			}},
		},
	}

	if got := tmpl.ResolvedColumnName(0, "zh-CN"); got != "待办" {
		t.Errorf("col[0] zh-CN = %q", got)
	}
	if got := tmpl.ResolvedColumnName(2, "zh-CN"); got != "完成" {
		t.Errorf("col[2] zh-CN = %q", got)
	}
	if got := tmpl.ResolvedColumnName(0, ""); got != "To Do" {
		t.Errorf("col[0] no-locale = %q", got)
	}
	if got := tmpl.ResolvedColumnName(0, "partial"); got != "First Only" {
		t.Errorf("col[0] partial = %q", got)
	}
	if got := tmpl.ResolvedColumnName(1, "partial"); got != "In Progress" {
		t.Errorf("col[1] partial should fallback, got %q", got)
	}
	if got := tmpl.ResolvedColumnName(99, ""); got != "" {
		t.Errorf("out of bounds should be empty, got %q", got)
	}
}

// --- getColumnIDs ---

func TestGetColumnIDs(t *testing.T) {
	cols := []*board.Column{
		{ColumnID: "col-1"}, {ColumnID: "col-2"}, {ColumnID: "col-3"},
	}
	ids := board.ExportGetColumnIDs(cols)
	if len(ids) != 3 {
		t.Fatalf("len = %d, want 3", len(ids))
	}
	if ids[0] != "col-1" || ids[2] != "col-3" {
		t.Errorf("ids = %v", ids)
	}
}

func TestGetColumnIDsEmpty(t *testing.T) {
	if len(board.ExportGetColumnIDs(nil)) != 0 {
		t.Error("nil should return empty")
	}
	if len(board.ExportGetColumnIDs([]*board.Column{})) != 0 {
		t.Error("empty should return empty")
	}
}

// --- rowToBoard ---

func TestRowToBoard(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	row := map[string]interface{}{
		"board_id":   "board-001",
		"name":       "Test Board",
		"icon":       "material-star",
		"color":      "#FF0000",
		"position":   float64(2),
		"created_at": now,
		"updated_at": now.Format(time.RFC3339),
	}
	b := board.ExportRowToBoard(row)
	if b.BoardID != "board-001" {
		t.Errorf("BoardID = %q", b.BoardID)
	}
	if b.Name != "Test Board" {
		t.Errorf("Name = %q", b.Name)
	}
	if b.Position != 2 {
		t.Errorf("Position = %d", b.Position)
	}
	if b.CreatedAt.IsZero() {
		t.Error("CreatedAt zero")
	}
	if b.UpdatedAt.IsZero() {
		t.Error("UpdatedAt zero")
	}
}

// --- rowToColumn ---

func TestRowToColumn(t *testing.T) {
	row := map[string]interface{}{
		"column_id":  "col-x",
		"board_id":   "board-y",
		"name":       "In Progress",
		"icon":       "material-sync",
		"color":      "#3B82F6",
		"position":   float64(2),
		"collapsed":  true,
		"created_at": time.Now(),
	}
	c := board.ExportRowToColumn(row)
	if c.ColumnID != "col-x" {
		t.Errorf("ColumnID = %q", c.ColumnID)
	}
	if c.Name != "In Progress" {
		t.Errorf("Name = %q", c.Name)
	}
	if c.Position != 2 {
		t.Errorf("Position = %d", c.Position)
	}
	if !c.Collapsed {
		t.Error("Collapsed should be true")
	}
}

// --- helper functions ---

func TestHelperGetString(t *testing.T) {
	row := map[string]interface{}{"k": "v", "nil": nil, "num": 42}
	if board.ExportGetString(row, "k") != "v" {
		t.Error("valid string")
	}
	if board.ExportGetString(row, "nil") != "" {
		t.Error("nil")
	}
	if board.ExportGetString(row, "num") != "" {
		t.Error("non-string")
	}
}

func TestHelperGetInt(t *testing.T) {
	row := map[string]interface{}{"f": float64(7), "i64": int64(8), "i": 9}
	if board.ExportGetInt(row, "f") != 7 {
		t.Error("float64")
	}
	if board.ExportGetInt(row, "i64") != 8 {
		t.Error("int64")
	}
	if board.ExportGetInt(row, "i") != 9 {
		t.Error("int")
	}
	if board.ExportGetInt(row, "missing") != 0 {
		t.Error("missing")
	}
}

func TestHelperGetBool(t *testing.T) {
	row := map[string]interface{}{"t": true, "f": false, "f1": float64(1), "i0": int64(0)}
	if !board.ExportGetBool(row, "t") {
		t.Error("true")
	}
	if board.ExportGetBool(row, "f") {
		t.Error("false")
	}
	if !board.ExportGetBool(row, "f1") {
		t.Error("float64(1)")
	}
	if board.ExportGetBool(row, "i0") {
		t.Error("int64(0)")
	}
}

func TestHelperGetTime(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	row := map[string]interface{}{
		"t": now, "s": now.Format(time.RFC3339),
		"d": now.Format("2006-01-02 15:04:05"), "bad": "x",
	}
	if board.ExportGetTime(row, "t") == nil {
		t.Error("time.Time")
	}
	if board.ExportGetTime(row, "s") == nil {
		t.Error("RFC3339")
	}
	if board.ExportGetTime(row, "d") == nil {
		t.Error("datetime")
	}
	if board.ExportGetTime(row, "bad") != nil {
		t.Error("invalid")
	}
}
