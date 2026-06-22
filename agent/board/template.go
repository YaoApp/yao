package board

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"gopkg.in/yaml.v3"
)

//go:embed templates/*.yaml
var templateFS embed.FS

var cachedTemplates []*Template

// Templates returns all available board templates.
// If locale is provided, template names are resolved to the given locale.
func Templates(ctx context.Context, locale ...string) ([]*Template, error) {
	if cachedTemplates == nil {
		entries, err := templateFS.ReadDir("templates")
		if err != nil {
			return nil, fmt.Errorf("board.Templates: %w", err)
		}

		templates := make([]*Template, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := filepath.Ext(entry.Name())
			if ext != ".yaml" && ext != ".yml" {
				continue
			}

			data, err := templateFS.ReadFile("templates/" + entry.Name())
			if err != nil {
				continue
			}

			var tmpl Template
			if err := yaml.Unmarshal(data, &tmpl); err != nil {
				continue
			}
			templates = append(templates, &tmpl)
		}
		cachedTemplates = templates
	}

	// Return locale-resolved copies if locale is specified
	loc := ""
	if len(locale) > 0 {
		loc = locale[0]
	}
	if loc == "" {
		return cachedTemplates, nil
	}

	resolved := make([]*Template, 0, len(cachedTemplates))
	for _, t := range cachedTemplates {
		rt := &Template{
			ID:      t.ID,
			Name:    t.ResolvedName(loc),
			Icon:    t.Icon,
			Color:   t.Color,
			Columns: make([]TemplateColumn, len(t.Columns)),
			Locales: t.Locales,
		}
		for i, col := range t.Columns {
			rt.Columns[i] = TemplateColumn{
				Name:  t.ResolvedColumnName(i, loc),
				Icon:  col.Icon,
				Color: col.Color,
			}
		}
		resolved = append(resolved, rt)
	}
	return resolved, nil
}

// FromTemplate creates a board from a template with locale-aware names
func FromTemplate(ctx context.Context, auth *process.AuthorizedInfo, req *FromTemplateReq) (*Board, error) {
	templates, err := Templates(ctx)
	if err != nil {
		return nil, err
	}

	var tmpl *Template
	for _, t := range templates {
		if t.ID == req.TemplateID {
			tmpl = t
			break
		}
	}
	if tmpl == nil {
		return nil, fmt.Errorf("board.FromTemplate: template %s not found", req.TemplateID)
	}

	// Resolve name: explicit > locale > fallback
	name := req.Name
	if name == "" {
		name = tmpl.ResolvedName(req.Locale)
	}

	// Create board
	board, err := Create(ctx, auth, &CreateReq{
		Name:  name,
		Icon:  tmpl.Icon,
		Color: tmpl.Color,
	})
	if err != nil {
		return nil, err
	}

	// The default column was already created by Create, remove it if template has columns
	if len(tmpl.Columns) > 0 && len(board.Columns) > 0 {
		for _, col := range board.Columns {
			capsule.Global.Query().Table(tableBoardColumn()).
				Where("column_id", "=", col.ColumnID).
				MustDelete()
		}

		// Create template columns with locale-resolved names
		for i, tc := range tmpl.Columns {
			colName := tmpl.ResolvedColumnName(i, req.Locale)
			_, err := CreateColumn(ctx, auth, board.BoardID, &ColumnReq{
				Name:  colName,
				Icon:  tc.Icon,
				Color: tc.Color,
			})
			if err != nil {
				return nil, fmt.Errorf("board.FromTemplate create column: %w", err)
			}
		}
	}

	return Get(ctx, auth, board.BoardID)
}
