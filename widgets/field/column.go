package field

import (
	"fmt"
	"strings"

	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/expression"
)

// Replace replace with data
func (column ColumnDSL) Replace(data map[string]interface{}) (*ColumnDSL, error) {
	new := column
	err := expression.Replace(&new.Key, data)
	if err != nil {
		return nil, err
	}

	err = expression.Replace(&new.Bind, data)
	if err != nil {
		return nil, err
	}

	if new.Edit != nil {
		err = expression.Replace(&new.Edit.Props, data)
		if err != nil {
			return nil, err
		}
	}

	if new.View != nil {
		err = expression.Replace(&new.View.Props, data)
		if err != nil {
			return nil, err
		}
	}

	return &new, nil
}

// Clone column
func (column *ColumnDSL) Clone() *ColumnDSL {
	new := ColumnDSL{
		Key:  column.Key,
		Bind: column.Bind,
		Link: column.Link,
		In:   column.In,
		Out:  column.Out,
	}

	if column.View != nil {
		new.View = column.View.Clone()
	}

	if column.Edit != nil {
		new.Edit = column.Edit.Clone()
	}
	return &new
}

// Map cast to map[string]inteface{}
func (column ColumnDSL) Map() map[string]interface{} {
	res := map[string]interface{}{
		"bind": column.Bind,
	}

	if column.Link != "" {
		res["link"] = column.Link
	}

	if column.View != nil {
		res["view"] = column.View.Map()
	}

	if column.Edit != nil {
		res["edit"] = column.Edit.Map()
	}
	return res
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
			if !strings.Contains(string(column.In), ".") {
				column.In = Compute(fmt.Sprintf("yao.component.%s", column.In))
			}
			computeInFields[column.Bind] = string(column.In)
			computeInFields[name] = string(column.In)
		}

		// Compute Out
		if column.Out != "" {
			if !strings.Contains(string(column.Out), ".") {
				column.In = Compute(fmt.Sprintf("yao.component.%s", column.Out))
			}
			computeOutFields[column.Bind] = string(column.Out)
			computeOutFields[name] = string(column.Out)
		}
	}
}

func mergeCProps(cloudProps map[string]component.CloudPropsDSL, cProps map[string]component.CloudPropsDSL) {
	for k, v := range cProps {
		cloudProps[k] = v
	}
}
