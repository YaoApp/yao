package list

import (
	"fmt"

	"github.com/yaoapp/yao/widgets/compute"
	"github.com/yaoapp/yao/widgets/field"
)

func (dsl *DSL) getField() func(string) (*field.ColumnDSL, string, string, error) {
	return func(name string) (*field.ColumnDSL, string, string, error) {
		field, has := dsl.Fields.List[name]
		if !has {
			return nil, "fields.list", dsl.ID, fmt.Errorf("fields.list.%s does not exist", name)
		}
		return &field, "fields.list", dsl.ID, nil
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

	if dsl.Fields.List != nil && dsl.Layout.List != nil {

		for _, inst := range dsl.Layout.List.Columns {

			if field, has := dsl.Fields.List[inst.Name]; has {

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
