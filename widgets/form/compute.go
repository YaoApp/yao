package form

import (
	"fmt"

	"github.com/yaoapp/yao/widgets/compute"
	"github.com/yaoapp/yao/widgets/field"
)

func (dsl *DSL) getField() func(string) (*field.ColumnDSL, string, error) {
	return func(name string) (*field.ColumnDSL, string, error) {
		field, has := dsl.Fields.Form[name]
		if !has {
			return nil, "fields.table", fmt.Errorf("fields.table.%s does not exist", name)
		}
		return &field, "fields.table", nil
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

	if dsl.Fields.Form != nil && dsl.Layout.Form != nil && dsl.Layout.Form.Sections != nil {
		for _, section := range dsl.Layout.Form.Sections {

			for _, inst := range section.Columns {

				if field, has := dsl.Fields.Form[inst.Name]; has {

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
	}

	return nil
}
