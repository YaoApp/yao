package chart

import (
	"fmt"

	"github.com/yaoapp/yao/widgets/compute"
	"github.com/yaoapp/yao/widgets/field"
	"github.com/yaoapp/yao/widgets/mapping"
)

func (dsl *DSL) getField() func(string) (*field.ColumnDSL, string, string, error) {
	return func(name string) (*field.ColumnDSL, string, string, error) {
		field, has := dsl.Fields.Chart[name]
		if !has {
			return nil, "fields.chart", dsl.ID, fmt.Errorf("fields.chart.%s does not exist", name)
		}
		return &field, "fields.chart", dsl.ID, nil
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
	// Filter
	if dsl.Fields.Filter != nil && dsl.Layout.Filter != nil && dsl.Layout.Filter.Columns != nil {
		for _, inst := range dsl.Layout.Filter.Columns {
			if filter, has := dsl.Fields.Filter[inst.Name]; has {
				// Add the default value, and parse the backend only props
				filter.Parse()

				// Mapping ID
				dsl.Mapping.Filters[filter.ID] = inst.Name
				dsl.Mapping.Filters[inst.Name] = filter.ID

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

	if dsl.Fields.Chart != nil && dsl.Layout.Chart != nil && dsl.Layout.Chart.Columns != nil {
		for _, inst := range dsl.Layout.Chart.Columns {
			if field, has := dsl.Fields.Chart[inst.Name]; has {

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
			}
		}
	}

	// Mapping Actions
	dsl.mappingActions()

	// Filters
	err := dsl.Fields.Filter.CPropsMerge(dsl.CProps, func(name string, filter field.FilterDSL) (xpath string) {
		return fmt.Sprintf("fields.filter.%s.edit.props", name)
	})

	if err != nil {
		return err
	}

	// Columns
	return dsl.Fields.Chart.CPropsMerge(dsl.CProps, func(name string, kind string, column field.ColumnDSL) (xpath string) {
		return fmt.Sprintf("fields.chart.%s.%s.props", name, kind)
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
		dsl.Layout.Operation != nil &&
		dsl.Layout.Operation.Actions != nil &&
		len(dsl.Layout.Operation.Actions) > 0 {

		for idx, action := range dsl.Layout.Operation.Actions {
			xpath := fmt.Sprintf("layout.operation.actions[%d]", idx)
			dsl.Mapping.Actions[action.ID] = xpath
			dsl.Mapping.Actions[xpath] = action.ID
		}

	}

	// layout.filter.actions
	if dsl.Layout != nil &&
		dsl.Layout.Filter != nil &&
		dsl.Layout.Filter.Actions != nil &&
		len(dsl.Layout.Filter.Actions) > 0 {

		for idx, action := range dsl.Layout.Filter.Actions {
			xpath := fmt.Sprintf("layout.filter.actions[%d]", idx)
			dsl.Mapping.Actions[action.ID] = xpath
			dsl.Mapping.Actions[xpath] = action.ID
		}
	}

}
