package board

import "time"

// ListQuery parameters for listing boards
type ListQuery struct{}

// ListResult board list response
type ListResult struct {
	Boards []*Board `json:"boards"`
}

// Board represents a kanban board
type Board struct {
	BoardID   string    `json:"board_id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon,omitempty"`
	Color     string    `json:"color,omitempty"`
	Position  int       `json:"position"`
	TaskCount int       `json:"task_count"`
	Columns   []*Column `json:"columns,omitempty"`
	Metadata  any       `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Column represents a board column
type Column struct {
	ColumnID  string    `json:"column_id"`
	BoardID   string    `json:"board_id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon,omitempty"`
	Color     string    `json:"color,omitempty"`
	Position  int       `json:"position"`
	Collapsed bool      `json:"collapsed"`
	Metadata  any       `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateReq for creating a new board
type CreateReq struct {
	Name  string `json:"name"`
	Icon  string `json:"icon,omitempty"`
	Color string `json:"color,omitempty"`
}

// UpdateReq for updating a board
type UpdateReq struct {
	Name     *string `json:"name,omitempty"`
	Icon     *string `json:"icon,omitempty"`
	Color    *string `json:"color,omitempty"`
	Position *int    `json:"position,omitempty"`
	Metadata any     `json:"metadata,omitempty"`
}

// ColumnReq for creating or updating a column
type ColumnReq struct {
	Name      string `json:"name"`
	Icon      string `json:"icon,omitempty"`
	Color     string `json:"color,omitempty"`
	Collapsed *bool  `json:"collapsed,omitempty"`
	Metadata  any    `json:"metadata,omitempty"`
}

// Template represents a board template with i18n support
type Template struct {
	ID      string                    `json:"id" yaml:"id"`
	Name    string                    `json:"name" yaml:"name"`
	Icon    string                    `json:"icon" yaml:"icon"`
	Color   string                    `json:"color" yaml:"color"`
	Columns []TemplateColumn          `json:"columns" yaml:"columns"`
	Locales map[string]TemplateLocale `json:"locales,omitempty" yaml:"locales,omitempty"`
}

// TemplateLocale holds translated text for a specific locale
type TemplateLocale struct {
	Name    string               `json:"name" yaml:"name"`
	Columns []TemplateColumnName `json:"columns,omitempty" yaml:"columns,omitempty"`
}

// TemplateColumnName holds a column's localized name
type TemplateColumnName struct {
	Name string `json:"name" yaml:"name"`
}

// TemplateColumn a column in a template
type TemplateColumn struct {
	Name  string `json:"name" yaml:"name"`
	Icon  string `json:"icon" yaml:"icon"`
	Color string `json:"color" yaml:"color"`
}

// FromTemplateReq for creating a board from a template
type FromTemplateReq struct {
	TemplateID string `json:"template_id"`
	Name       string `json:"name,omitempty"`
	Locale     string `json:"locale,omitempty"`
}

// ResolvedName returns the template name for the given locale (fallback to default)
func (t *Template) ResolvedName(locale string) string {
	if locale != "" && t.Locales != nil {
		if l, ok := t.Locales[locale]; ok && l.Name != "" {
			return l.Name
		}
	}
	return t.Name
}

// ResolvedColumnName returns the column name at index for the given locale
func (t *Template) ResolvedColumnName(index int, locale string) string {
	if locale != "" && t.Locales != nil {
		if l, ok := t.Locales[locale]; ok && index < len(l.Columns) && l.Columns[index].Name != "" {
			return l.Columns[index].Name
		}
	}
	if index < len(t.Columns) {
		return t.Columns[index].Name
	}
	return ""
}
