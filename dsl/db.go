package dsl

import (
	"fmt"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/dsl/types"
)

// getInfoFromDB get the info from the db
func (dsl *DSL) dbInspect(id string) (*types.Info, bool, error) {

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

// getSourceFromDB get the source from the db
func (dsl *DSL) dbSource(id string) (string, bool, error) {

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
		return "", true, fmt.Errorf("%s %s source is not a string", dsl.Type, id)
	}

	return source, true, nil
}

// getListFromDB get the list from the db
func (dsl *DSL) dbList(options *types.ListOptions) ([]*types.Info, error) {

	// Get from database
	m := model.Select("__yao.dsl")

	var orders []model.QueryOrder = []model.QueryOrder{{Column: "mtime", Option: "desc"}}
	if options.Sort == "sort" {
		orders = []model.QueryOrder{{Column: "sort", Option: "asc"}}
	}

	var wheres []model.QueryWhere = []model.QueryWhere{{Column: "type", Value: dsl.Type}}

	// Filter by tags
	if len(options.Tags) > 0 {
		var orwheres []model.QueryWhere = []model.QueryWhere{}
		for _, tag := range options.Tags {
			match := "%" + strings.TrimSpace(tag) + "%"
			orwheres = append(orwheres, model.QueryWhere{Column: "tags", Value: match, OP: "like", Method: "orwhere"})
		}
		wheres = append(wheres, model.QueryWhere{Wheres: orwheres})
	}

	// Get the list
	rows, err := m.Get(model.QueryParam{
		Wheres: []model.QueryWhere{{Column: "type", Value: dsl.Type}},
		Select: []interface{}{"dsl_id", "label", "path", "sort", "tags", "description", "status", "store", "mtime", "ctime"},
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

func (dsl *DSL) dbCreate(options *types.CreateOptions) error {

	if options.Source == "" {
		return fmt.Errorf("%s %s source is required", dsl.Type, options.ID)
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
		"type":        dsl.Type,
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

func (dsl *DSL) dbUpdate(options *types.UpdateOptions) error {
	if options.Source == "" && options.Info == nil {
		return fmt.Errorf("%s %s one of source or info is required", dsl.Type, options.ID)
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
		return fmt.Errorf("%s %s not found", dsl.Type, options.ID)
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

func (dsl *DSL) dbDelete(id string) error {

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
		return fmt.Errorf("%s %s not found", dsl.Type, id)
	}

	// Delete the dsl
	row := rows[0]
	return m.Delete(row["id"])
}

func (dsl *DSL) dbExists(id string) (bool, error) {

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
