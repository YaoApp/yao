package table

import (
	"fmt"

	"github.com/yaoapp/yao/widgets/compute"
	"github.com/yaoapp/yao/widgets/field"
)

func (dsl *DSL) getField() func(string) (*field.ColumnDSL, string, string, error) {
	return func(name string) (*field.ColumnDSL, string, string, error) {
		field, has := dsl.Fields.Table[name]
		if !has {
			return nil, "fields.table", dsl.ID, fmt.Errorf("fields.table.%s does not exist", name)
		}
		return &field, "fields.table", dsl.ID, nil
	}
}

func (dsl *DSL) getFilter() func(string) (*field.FilterDSL, string, string, error) {
	return func(name string) (*field.FilterDSL, string, string, error) {
		field, has := dsl.Fields.Filter[name]
		if !has {
			return nil, "fields.filter", dsl.ID, fmt.Errorf("fields.filter.%s does not exist", name)
		}
		return &field, "fields.filter", dsl.ID, nil
	}
}

func (dsl *DSL) computeMapping() error {
	if dsl.Computes == nil {
		dsl.Computes = &compute.Maps{
			Filter: map[string][]compute.Unit{},
			Edit:   map[string][]compute.Unit{},
			View:   map[string][]compute.Unit{},
		}
	}

	if dsl.Fields == nil {
		return nil
	}

	// Filter
	if dsl.Fields.Filter != nil && dsl.Layout.Filter != nil && dsl.Layout.Filter.Columns != nil {
		for _, inst := range dsl.Layout.Filter.Columns {
			if filter, has := dsl.Fields.Filter[inst.Name]; has {
				if filter.Edit != nil && filter.Edit.Compute != nil {
					bind := filter.FilterBind()
					if _, has := dsl.Computes.Filter[bind]; !has {
						dsl.Computes.Filter[bind] = []compute.Unit{}
					}
					dsl.Computes.Filter[bind] = append(dsl.Computes.Filter[bind], compute.Unit{Name: inst.Name, Kind: compute.Filter})
				}
			}
		}
	}

	if dsl.Fields.Table != nil && dsl.Layout.Table != nil && dsl.Layout.Table.Columns != nil {
		for _, inst := range dsl.Layout.Table.Columns {
			if field, has := dsl.Fields.Table[inst.Name]; has {
				// View
				if field.View != nil && field.View.Compute != nil {
					bind := field.ViewBind()
					if _, has := dsl.Computes.View[bind]; !has {
						dsl.Computes.View[bind] = []compute.Unit{}
					}
					dsl.Computes.View[bind] = append(dsl.Computes.View[bind], compute.Unit{Name: inst.Name, Kind: compute.View})
				}

				// Edit
				if field.Edit != nil && field.Edit.Compute != nil {
					bind := field.EditBind()
					if _, has := dsl.Computes.Edit[bind]; !has {
						dsl.Computes.Edit[bind] = []compute.Unit{}
					}
					dsl.Computes.Edit[bind] = append(dsl.Computes.Edit[bind], compute.Unit{Name: inst.Name, Kind: compute.Edit})
				}
			}
		}
	}

	return nil
}
