package form

import (
	"fmt"

	"github.com/yaoapp/yao/widgets/compute"
	"github.com/yaoapp/yao/widgets/field"
	"github.com/yaoapp/yao/widgets/mapping"
)

func (dsl *DSL) getField() func(string) (*field.ColumnDSL, string, string, error) {
	return func(name string) (*field.ColumnDSL, string, string, error) {
		field, has := dsl.Fields.Form[name]
		if !has {
			return nil, "fields.form", dsl.ID, fmt.Errorf("fields.form.%s does not exist", name)
		}
		return &field, "fields.form", dsl.ID, nil
	}
}

func (dsl *DSL) mapping() error {
	if dsl.Computes == nil {
		dsl.Computes = &compute.Maps{
			Filter: map[string][]compute.Unit{},
			Edit:   map[string][]compute.Unit{},
			View:   map[string][]compute.Unit{},
		}
	}

	if dsl.Mapping == nil {
		dsl.Mapping = &mapping.Mapping{}
	}

	if dsl.Mapping.Filters == nil {
		dsl.Mapping.Filters = map[string]string{}
	}

	if dsl.Mapping.Columns == nil {
		dsl.Mapping.Columns = map[string]string{}
	}

	if dsl.Fields == nil {
		return nil
	}

	// Mapping compute and id
	if dsl.Fields.Form != nil && dsl.Layout.Form != nil && dsl.Layout.Form.Sections != nil {
		dsl.Layout.listColumns(func(path string, inst Column) {

			if field, has := dsl.Fields.Form[inst.Name]; has {

				// Add the default value, and parse the backend only props
				field.Parse()

				// Mapping ID
				dsl.Mapping.Columns[field.ID] = inst.Name
				dsl.Mapping.Columns[inst.Name] = field.ID

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

		}, "", nil)
	}

	// Mapping Actions
	dsl.mappingActions()

	// Columns
	return dsl.Fields.Form.CPropsMerge(dsl.CProps, func(name string, kind string, column field.ColumnDSL) (xpath string) {
		return fmt.Sprintf("fields.form.%s.%s.props", name, kind)
	})
}

// Actions get the table actions
func (dsl *DSL) mappingActions() {

	if dsl.Mapping == nil {
		dsl.Mapping = &mapping.Mapping{}
	}

	if dsl.Mapping.Actions == nil {
		dsl.Mapping.Actions = map[string]string{}
	}

	// layout.operation.actions
	if dsl.Layout != nil &&
		dsl.Layout.Actions != nil &&
		len(dsl.Layout.Actions) > 0 {
		for idx, action := range dsl.Layout.Actions {
			xpath := fmt.Sprintf("layout.actions[%d]", idx)
			dsl.Mapping.Actions[action.ID] = xpath
			dsl.Mapping.Actions[xpath] = action.ID
		}
	}
}
