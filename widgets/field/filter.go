package field

import (
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/expression"
)

// Replace replace with data
func (filter FilterDSL) Replace(data map[string]interface{}) (*FilterDSL, error) {
	new := filter
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

	return &new, nil
}

// Clone column
func (filter *FilterDSL) Clone() *FilterDSL {
	new := FilterDSL{
		Key:  filter.Key,
		Bind: filter.Bind,
	}
	if filter.Edit != nil {
		new.Edit = filter.Edit.Clone()
	}
	return &new
}

// Map cast to map[string]inteface{}
func (filter FilterDSL) Map() map[string]interface{} {
	res := map[string]interface{}{
		"key":  filter.Key,
		"bind": filter.Bind,
	}

	if filter.Edit != nil {
		res["edit"] = filter.Edit.Map()
	}

	return res
}

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
