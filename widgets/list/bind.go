package list

import (
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/widgets/table"
)

// Bind model / store / table / ...
func (dsl *DSL) Bind() error {

	// Support bind in future version

	return nil

	// if dsl.Action.Bind == nil {
	// 	return nil
	// }

	// if dsl.Action.Bind.Model != "" {
	// 	return dsl.bindModel()
	// }

	// if dsl.Action.Bind.Store != "" {
	// 	return dsl.bindStore()
	// }

	// if dsl.Action.Bind.Table != "" {
	// 	return dsl.bindTable()
	// }

	// return nil
}

func (dsl *DSL) bindModel() error {

	id := dsl.Action.Bind.Model
	m, has := model.Models[id]
	if !has {
		return fmt.Errorf("%s does not exist", id)
	}

	dsl.Action.BindModel(m)
	dsl.Fields.BindModel(m)
	// dsl.Layout.BindModel(m, dsl.ID, dsl.Fields, dsl.Action.Bind.Option)
	return nil
}

func (dsl *DSL) bindTable() error {
	id := dsl.Action.Bind.Table

	// Load table
	if _, has := table.Tables[id]; !has {
		if err := table.LoadID(id); err != nil {
			return err
		}
	}

	tab, err := table.Get(id)
	if err != nil {
		return err
	}

	// Bind Fields
	err = dsl.Fields.BindTable(tab)
	if err != nil {
		return err
	}

	// Bind Actions
	err = dsl.Action.BindTable(tab)
	if err != nil {
		return err
	}

	// Bind Layout
	err = dsl.Layout.BindTable(tab, dsl.ID, dsl.Fields)
	if err != nil {
		return err
	}

	return nil
}

func (dsl *DSL) bindStore() error {
	id := dsl.Action.Bind.Store
	return fmt.Errorf("bind.store %s does not support yet", id)
}
