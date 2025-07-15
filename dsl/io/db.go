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

// Inspect get the info from the db
func (db *DB) Inspect(id string) (*types.Info, bool, error) {

	// Get from database
	m := model.Select("__yao.dsl")

	// Get the info
	var info types.Info
	rows, err := m.Get(model.QueryParam{
		Wheres: []model.QueryWhere{{Column: "dsl_id", Value: id}},
		Select: []interface{}{
			"dsl_id",
			"type",
			"label",
			"path",
			"sort",
			"tags",
			"description",
			"status",
			"store",
			"mtime",
			"ctime",
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

	raw, err := jsoniter.Marshal(rows[0])
	if err != nil {
		return nil, false, err
	}

	err = jsoniter.Unmarshal(raw, &info)
	if err != nil {
		return nil, false, err
	}

	return &info, true, nil
}

// Source get the source from the db
func (db *DB) Source(id string) (string, bool, error) {

	// Get from database
	m := model.Select("__yao.dsl")

	// Get the source
	rows, err := m.Get(model.QueryParam{
		Wheres: []model.QueryWhere{{Column: "dsl_id", Value: id}},
		Select: []interface{}{"source"},
		Limit:  1,
	})
	if err != nil {
		return "", false, err
	}

	if len(rows) == 0 {
		return "", false, nil
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
	fields := []interface{}{"dsl_id", "label", "path", "sort", "tags", "description", "status", "store", "mtime", "ctime"}
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

	var infos []*types.Info
	raw, err := jsoniter.Marshal(rows)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(raw, &infos)
	if err != nil {
		return nil, err
	}

	return infos, nil
}

// Create create the dsl
func (db *DB) Create(options *types.CreateOptions) error {

	if options.Source == "" {
		return fmt.Errorf("%s %s source is required", db.Type, options.ID)
	}

	// Get info from source
	var info types.Info
	err := jsoniter.Unmarshal([]byte(options.Source), &info)
	if err != nil {
		return err
	}

	// Get the info
	m := model.Select("__yao.dsl")
	data := map[string]interface{}{
		"source":      options.Source,
		"dsl_id":      options.ID,
		"type":        db.Type,
		"label":       info.Label,
		"path":        info.Path,
		"sort":        info.Sort,
		"tags":        info.Tags,
		"description": info.Description,
		"status":      info.Status,
		"store":       info.Store,
		"mtime":       time.Now().Unix(),
		"ctime":       time.Now().Unix(),
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
		Wheres: []model.QueryWhere{{Column: "dsl_id", Value: options.ID}},
		Select: []interface{}{"id", "dsl_id"},
		Limit:  1,
	})
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return fmt.Errorf("%s %s not found", db.Type, options.ID)
	}

	row := rows[0]

	// update source
	var data map[string]interface{} = map[string]interface{}{}
	if options.Source != "" {
		data["source"] = options.Source
	} else {
		// Update info
		if options.Info.Label != "" {
			data["label"] = options.Info.Label
		}

		if options.Info.Path != "" {
			data["path"] = options.Info.Path
		}

		if options.Info.Sort != 0 {
			data["sort"] = options.Info.Sort
		}

		if options.Info.Tags != nil {
			data["tags"] = options.Info.Tags
		}

		if options.Info.Description != "" {
			data["description"] = options.Info.Description
		}

	}

	// Update the data
	err = m.Update(row["id"], data)
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
		Wheres: []model.QueryWhere{{Column: "dsl_id", Value: id}},
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
		Wheres: []model.QueryWhere{{Column: "dsl_id", Value: id}},
		Select: []interface{}{"id", "dsl_id"},
		Limit:  1,
	})
	if err != nil {
		return false, err
	}

	return len(rows) > 0, nil
}
