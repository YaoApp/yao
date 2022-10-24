package form

import (
	"fmt"

	"github.com/yaoapp/gou"
)

// Bind model / store / table / ...
func (dsl *DSL) Bind() error {

	if dsl.Action.Bind == nil {
		return nil
	}

	if dsl.Action.Bind.Model != "" {
		return dsl.bindModel()
	}

	if dsl.Action.Bind.Store != "" {
		return dsl.bindStore()
	}

	if dsl.Action.Bind.Table != "" {
		return dsl.bindTable()
	}

	return nil
}

func (dsl *DSL) bindModel() error {

	id := dsl.Action.Bind.Model
	m, has := gou.Models[id]
	if !has {
		return fmt.Errorf("%s does not exist", id)
	}

	dsl.Action.BindModel(m)
	dsl.Fields.BindModel(m)
	dsl.Layout.BindModel(m, dsl.ID, dsl.Fields, dsl.Action.Bind.Option)
	return nil
}

func (dsl *DSL) bindTable() error {
	id := dsl.Action.Bind.Table
	return fmt.Errorf("bind.table %s does not support yet", id)
}

func (dsl *DSL) bindStore() error {
	id := dsl.Action.Bind.Store
	return fmt.Errorf("bind.store %s does not support yet", id)
}
