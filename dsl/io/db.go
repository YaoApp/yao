package io

import (
	"fmt"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/dsl/types"
)

// DB is the db io
type DB struct {
	Type types.Type
}

// NewDB create a new db io
func NewDB(typ types.Type) types.IO {
	return &DB{Type: typ}
}

// fmtRow format the row data for DSL info
func fmtRow(row map[string]interface{}) map[string]interface{} {
	// Handle source field first
	if source, ok := row["source"]; ok {
		if str, ok := source.(string); ok {
			row["source"] = str
		}
	}

	// Map fields
	if id, ok := row["dsl_id"]; ok {
		row["id"] = id
		delete(row, "dsl_id")
	}
	if readonly, ok := row["readonly"]; ok {
		row["readonly"] = toBool(readonly)
		delete(row, "readonly")
	}
	if builtin, ok := row["built_in"]; ok {
		row["built_in"] = toBool(builtin)
	}

	// Convert time values
	if mtime, ok := row["mtime"]; ok && mtime != nil {
		if timeStr := toTime(mtime); timeStr != "" {
			row["mtime"] = timeStr
		}
	}
	if ctime, ok := row["ctime"]; ok && ctime != nil {
		if timeStr := toTime(ctime); timeStr != "" {
			row["ctime"] = timeStr
		}
	}

	return row
}

// Inspect get the info from the db
func (db *DB) Inspect(id string) (*types.Info, bool, error) {

	// Get from database
	m := model.Select("__yao.dsl")

	// Get the info
	var info types.Info
	rows, err := m.Get(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "dsl_id", Value: id},
			{Column: "type", Value: db.Type},
		},
		Select: []interface{}{
			"dsl_id",
			"type",
			"label",
			"path",
			"sort",
			"tags",
			"description",
			"store",
			"mtime",
			"ctime",
			"readonly",
			"built_in",
		},
		Limit:  1,
		Orders: []model.QueryOrder{{Column: "sort", Option: "asc"}, {Column: "mtime", Option: "desc"}},
	})
	if err != nil {
		return nil, false, err
	}

	if len(rows) == 0 {
		return nil, false, nil
	}

	// Format row data
	row := fmtRow(rows[0])

	raw, err := jsoniter.Marshal(row)
	if err != nil {
		return nil, false, err
	}

	err = jsoniter.Unmarshal(raw, &info)
	if err != nil {
		return nil, false, err
	}

	// Force set Store to DB since this record is from database
	info.Store = types.StoreTypeDB

	return &info, true, nil
}

// Source get the source from the db
func (db *DB) Source(id string) (string, bool, error) {

	// Get from database
	m := model.Select("__yao.dsl")

	// Get the source
	rows, err := m.Get(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "dsl_id", Value: id},
			{Column: "type", Value: db.Type},
		},
		Select: []interface{}{"source"},
		Limit:  1,
	})
	if err != nil {
		return "", false, err
	}

	if len(rows) == 0 {
		return "", false, nil
	}

	if rows[0]["source"] == nil {
		return "", true, nil
	}

	source, ok := rows[0]["source"].(string)
	if !ok {
		return "", true, fmt.Errorf("%s %s source is not a string", db.Type, id)
	}

	return source, true, nil
}

// List get the list from the db
func (db *DB) List(options *types.ListOptions) ([]*types.Info, error) {

	// Get from database
	m := model.Select("__yao.dsl")

	var orders []model.QueryOrder = []model.QueryOrder{{Column: "mtime", Option: "desc"}}
	if options.Sort == "sort" {
		orders = []model.QueryOrder{{Column: "sort", Option: "asc"}}
	}

	var wheres []model.QueryWhere = []model.QueryWhere{{Column: "type", Value: db.Type}}

	// Filter by tags
	if len(options.Tags) > 0 {
		var orwheres []model.QueryWhere = []model.QueryWhere{}
		for _, tag := range options.Tags {
			match := "%" + strings.TrimSpace(tag) + "%"
			orwheres = append(orwheres, model.QueryWhere{Column: "tags", Value: match, OP: "like", Method: "orwhere"})
		}
		wheres = append(wheres, model.QueryWhere{Wheres: orwheres})
	}

	// Select fields
	fields := []interface{}{
		"dsl_id",
		"type",
		"label",
		"path",
		"sort",
		"tags",
		"description",
		"store",
		"mtime",
		"ctime",
		"readonly",
		"built_in",
	}
	if options.Source {
		fields = append(fields, "source")
	}

	// Get the list
	rows, err := m.Get(model.QueryParam{
		Wheres: wheres,
		Select: fields,
		Orders: orders,
	})
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	// Format rows data
	for i := range rows {
		rows[i] = fmtRow(rows[i])
	}

	var infos []*types.Info
	raw, err := jsoniter.Marshal(rows)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(raw, &infos)
	if err != nil {
		return nil, err
	}

	// Force set Store to DB since these records are from database
	for _, info := range infos {
		info.Store = types.StoreTypeDB
	}

	return infos, nil
}

// Create create the dsl
func (db *DB) Create(options *types.CreateOptions) error {

	if options.Source == "" {
		return fmt.Errorf("%s %s source is required", db.Type, options.ID)
	}

	// Parse the source to extract metadata
	var sourceData map[string]interface{}
	err := jsoniter.Unmarshal([]byte(options.Source), &sourceData)
	if err != nil {
		return err
	}

	// Extract common fields from source
	var label, description string
	var tags []string
	var sort int

	if v, ok := sourceData["label"]; ok {
		if s, ok := v.(string); ok {
			label = s
		}
	}

	if v, ok := sourceData["description"]; ok {
		if s, ok := v.(string); ok {
			description = s
		}
	}

	if v, ok := sourceData["tags"]; ok {
		if tagsList, ok := v.([]interface{}); ok {
			for _, tag := range tagsList {
				if s, ok := tag.(string); ok {
					tags = append(tags, s)
				}
			}
		}
	}

	if v, ok := sourceData["sort"]; ok {
		if s, ok := v.(float64); ok {
			sort = int(s)
		}
	}

	// Set default store type if not specified
	store := options.Store
	if store == "" {
		store = types.StoreTypeFile
	}

	// Get the info
	m := model.Select("__yao.dsl")
	data := map[string]interface{}{
		"source":      options.Source,
		"dsl_id":      options.ID,
		"type":        db.Type,
		"label":       label,
		"path":        types.ToPath(db.Type, options.ID),
		"sort":        sort,
		"tags":        tags,
		"description": description,
		"store":       store,
		"mtime":       time.Now(),
		"ctime":       time.Now(),
		"readonly":    0,
		"built_in":    0,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	}

	_, err = m.Create(data)
	if err != nil {
		return err
	}

	return nil
}

// Update update the dsl
func (db *DB) Update(options *types.UpdateOptions) error {
	if options.Source == "" && options.Info == nil {
		return fmt.Errorf("%s %s one of source or info is required", db.Type, options.ID)
	}

	m := model.Select("__yao.dsl")

	// Check if the dsl exists
	rows, err := m.Get(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "dsl_id", Value: options.ID},
			{Column: "type", Value: db.Type},
		},
		Select: []interface{}{"id"},
		Limit:  1,
	})
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return fmt.Errorf("%s %s not found", db.Type, options.ID)
	}

	// update source
	var data map[string]interface{} = map[string]interface{}{
		"source": options.Source,
	}
	if options.Source != "" {
		// Parse source to extract metadata
		var sourceData map[string]interface{}
		err = jsoniter.Unmarshal([]byte(options.Source), &sourceData)
		if err != nil {
			return err
		}

		// Extract common fields from source
		if v, ok := sourceData["label"]; ok {
			if s, ok := v.(string); ok {
				data["label"] = s
			}
		}

		if v, ok := sourceData["description"]; ok {
			if s, ok := v.(string); ok {
				data["description"] = s
			}
		}

		if v, ok := sourceData["tags"]; ok {
			if tagsList, ok := v.([]interface{}); ok {
				var tags []string
				for _, tag := range tagsList {
					if s, ok := tag.(string); ok {
						tags = append(tags, s)
					}
				}
				data["tags"] = tags
			}
		}

		if v, ok := sourceData["sort"]; ok {
			if s, ok := v.(float64); ok {
				data["sort"] = int(s)
			}
		}
	} else {
		// Update info
		if options.Info.Label != "" {
			data["label"] = options.Info.Label
		}
		if options.Info.Description != "" {
			data["description"] = options.Info.Description
		}
		if len(options.Info.Tags) > 0 {
			data["tags"] = options.Info.Tags
		}
		if options.Info.Sort != 0 {
			data["sort"] = options.Info.Sort
		}
		if options.Info.Status != "" {
			data["status"] = options.Info.Status
		}
		if options.Info.Store != "" {
			data["store"] = options.Info.Store
		}
		if options.Info.Readonly {
			data["readonly"] = 1
		}
		if options.Info.Builtin {
			data["built_in"] = 1
		}
	}

	data["updated_at"] = time.Now()
	data["mtime"] = time.Now()

	err = m.Update(rows[0]["id"], data)
	if err != nil {
		return err
	}

	return nil
}

// Delete delete the dsl
func (db *DB) Delete(id string) error {

	// Get from database
	m := model.Select("__yao.dsl")

	// Check if the dsl exists
	rows, err := m.Get(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "dsl_id", Value: id},
			{Column: "type", Value: db.Type},
		},
		Select: []interface{}{"id", "dsl_id"},
		Limit:  1,
	})
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return fmt.Errorf("%s %s not found", db.Type, id)
	}

	// Delete the dsl
	row := rows[0]
	return m.Delete(row["id"])
}

// Exists check if the dsl exists
func (db *DB) Exists(id string) (bool, error) {

	// Get from database
	m := model.Select("__yao.dsl")

	// Check if the dsl exists
	rows, err := m.Get(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "dsl_id", Value: id},
			{Column: "type", Value: db.Type},
		},
		Select: []interface{}{"id", "dsl_id"},
		Limit:  1,
	})
	if err != nil {
		return false, err
	}

	return len(rows) > 0, nil
}
