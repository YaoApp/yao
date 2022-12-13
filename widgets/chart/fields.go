package chart

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/widgets/field"
)

// Xgen trans to xgen setting
func (fields *FieldsDSL) Xgen(layout *LayoutDSL) (map[string]interface{}, error) {
	res := map[string]interface{}{}

	filters := map[string]field.FilterDSL{}
	if layout.Filter != nil && layout.Filter.Columns != nil {
		for _, inst := range layout.Filter.Columns {
			if c, has := fields.Filter[inst.Name]; has {
				filters[inst.Name] = c
			}
		}
	}

	columns := map[string]field.ColumnDSL{}
	if layout.Chart != nil && layout.Chart.Columns != nil {
		for _, inst := range layout.Chart.Columns {
			if c, has := fields.Chart[inst.Name]; has {
				columns[inst.Name] = c
			}
		}
	}

	data, err := jsoniter.Marshal(map[string]interface{}{"filter": filters, "chart": columns})
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
