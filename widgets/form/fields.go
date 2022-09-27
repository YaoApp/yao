package form

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
)

// BindModel bind model
func (fields *FieldsDSL) BindModel(m *gou.Model) {
}

// Xgen trans to xgen setting
func (fields *FieldsDSL) Xgen() (map[string]interface{}, error) {
	res := map[string]interface{}{}
	data, err := jsoniter.Marshal(fields)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
