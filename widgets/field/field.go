package field

import (
	"github.com/yaoapp/yao/widgets/component"
)

// CPropsMerge merge the Filters cloud props
func (filters Filters) CPropsMerge(cloudProps map[string]component.CloudPropsDSL, getXpath func(name string, filter FilterDSL) (xpath string)) error {

	for name, filter := range filters {
		if filter.Edit != nil && filter.Edit.Props != nil {
			xpath := getXpath(name, filter)
			cProps, err := filter.Edit.Props.CloudProps(xpath)
			if err != nil {
				return err
			}
			mergeCProps(cloudProps, cProps)
		}
	}

	return nil
}

// CPropsMerge merge the Columns cloud props
func (columns Columns) CPropsMerge(cloudProps map[string]component.CloudPropsDSL, getXpath func(name string, kind string, column ColumnDSL) (xpath string)) error {

	for name, column := range columns {

		if column.Edit != nil && column.Edit.Props != nil {
			xpath := getXpath(name, "edit", column)
			cProps, err := column.Edit.Props.CloudProps(xpath)
			if err != nil {
				return err
			}
			mergeCProps(cloudProps, cProps)
		}

		if column.View != nil && column.View.Props != nil {
			xpath := getXpath(name, "view", column)
			cProps, err := column.View.Props.CloudProps(xpath)
			if err != nil {
				return err
			}
			mergeCProps(cloudProps, cProps)
		}
	}

	return nil
}

// ComputeFieldsMerge merge the compute fields
func (columns Columns) ComputeFieldsMerge(computeInFields map[string]string, computeOutFields map[string]string) {
	for name, column := range columns {

		// Compute In
		if column.In != "" {
			computeInFields[column.Bind] = column.In
			computeInFields[name] = column.In
		}

		// Compute Out
		if column.Out != "" {
			computeOutFields[column.Bind] = column.Out
			computeOutFields[name] = column.Out
		}
	}
}

func mergeCProps(cloudProps map[string]component.CloudPropsDSL, cProps map[string]component.CloudPropsDSL) {
	for k, v := range cProps {
		cloudProps[k] = v
	}
}
