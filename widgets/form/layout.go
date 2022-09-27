package form

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
)

// BindModel bind model
func (layout *LayoutDSL) BindModel(m *gou.Model) {
	layout.Primary = m.PrimaryKey
}

// Xgen trans to Xgen setting
func (layout *LayoutDSL) Xgen() (map[string]interface{}, error) {
	res := map[string]interface{}{}
	data, err := jsoniter.Marshal(layout)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
