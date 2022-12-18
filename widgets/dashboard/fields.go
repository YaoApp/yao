package dashboard

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
	if layout.Dashboard != nil && layout.Dashboard.Columns != nil {
		for _, inst := range layout.Dashboard.Columns {
			if c, has := fields.Dashboard[inst.Name]; has {

				if c.Edit != nil && c.Edit.Props != nil {
					if _, has := c.Edit.Props["$on:change"]; has {
						delete(c.Edit.Props, "$on:change")
					}
				}

				if c.View != nil && c.View.Props != nil {
					if _, has := c.View.Props["$on:change"]; has {
						delete(c.View.Props, "$on:change")
					}
				}

				columns[inst.Name] = c
			}

			if inst.Rows != nil {
				for _, inst := range inst.Rows {
					if c, has := fields.Dashboard[inst.Name]; has {

						if c.Edit != nil && c.Edit.Props != nil {
							if _, has := c.Edit.Props["$on:change"]; has {
								delete(c.Edit.Props, "$on:change")
							}
						}

						if c.View != nil && c.View.Props != nil {
							if _, has := c.View.Props["$on:change"]; has {
								delete(c.View.Props, "$on:change")
							}
						}

						columns[inst.Name] = c
					}
				}
			}
		}
	}

	data, err := jsoniter.Marshal(map[string]interface{}{"filter": filters, "dashboard": columns})
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
