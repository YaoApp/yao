package field

import (
	"fmt"
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/expression"
	"golang.org/x/crypto/md4"
)

// UnmarshalJSON for json UnmarshalJSON
func (filter *FilterDSL) UnmarshalJSON(data []byte) error {
	var alias aliasFilterDSL
	err := jsoniter.Unmarshal(data, &alias)
	if err != nil {
		return err
	}

	*filter = FilterDSL(alias)
	filter.ID, err = filter.Hash()
	if err != nil {
		return err
	}

	return nil
}

// Parse the column dsl, add the default value, and parse the backend only props
func (filter FilterDSL) Parse() {
	if filter.Edit != nil {
		filter.Edit.Parse()
	}
}

// Hash hash value
func (filter FilterDSL) Hash() (string, error) {
	h := md4.New()
	origin := fmt.Sprintf("FILTER::%#v", filter.Map())
	// fmt.Println(origin)
	io.WriteString(h, origin)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

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

	new.ID, err = filter.Hash()
	if err != nil {
		return nil, err
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
		"id":   filter.ID,
		"bind": filter.Bind,
	}

	if filter.Edit != nil {
		res["edit"] = filter.Edit.Map()
	}

	return res
}

// FilterBind get the bind field name of filter
func (filter FilterDSL) FilterBind() string {
	if filter.Edit != nil && filter.Edit.Bind != "" {
		return filter.Edit.Bind
	}
	return filter.Bind
}

// CPropsMerge merge the Filters cloud props
func (filters Filters) CPropsMerge(cloudProps map[string]component.CloudPropsDSL, getXpath func(name string, filter FilterDSL) (xpath string)) error {

	for name, filter := range filters {
		if filter.Edit != nil && filter.Edit.Props != nil {
			xpath := getXpath(name, filter)
			cProps, err := filter.Edit.Props.CloudProps(xpath, filter.Edit.Type)
			if err != nil {
				return err
			}
			mergeCProps(cloudProps, cProps)
		}
	}

	return nil
}
