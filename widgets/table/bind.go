package table

import (
	"fmt"

	"github.com/yaoapp/gou/model"
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
	m, has := model.Models[id]
	if !has {
		return fmt.Errorf("%s does not exist", id)
	}

	err := dsl.Fields.BindModel(m)
	if err != nil {
		return err
	}

	err = dsl.Action.BindModel(m)
	if err != nil {
		return err
	}

	err = dsl.Layout.BindModel(m, dsl.Fields, dsl.Action.Bind.Option)
	if err != nil {
		return err
	}

	return nil
}

func (dsl *DSL) bindTable() error {

	// Bind ID
	id := dsl.Action.Bind.Table
	if id == dsl.ID {
		return fmt.Errorf("bind.table %s can't bind self table", id)
	}

	// Load table
	if _, has := Tables[id]; !has {
		if err := LoadID(id); err != nil {
			return err
		}
	}

	tab, err := Get(id)
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
	err = dsl.Layout.BindTable(tab, dsl.Fields)
	if err != nil {
		return err
	}

	return nil
}

func (dsl *DSL) bindStore() error {
	id := dsl.Action.Bind.Store
	return fmt.Errorf("bind.store %s does not support yet", id)
}
