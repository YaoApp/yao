package table

import (
	"fmt"

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

	// replace import
	if layout.Header != nil && layout.Header.Preset != nil && layout.Header.Preset.Import != nil {
		name := layout.Header.Preset.Import.Name
		operation := layout.Header.Preset.Import.Operation
		res["header"].(map[string]interface{})["preset"].(map[string]interface{})["import"] = map[string]interface{}{
			"api": map[string]interface{}{
				"setting":               fmt.Sprintf("/api/xiang/import/%s/setting", name),
				"mapping":               fmt.Sprintf("/api/xiang/import/%s/mapping", name),
				"preview":               fmt.Sprintf("/api/xiang/import/%s/data", name),
				"import":                fmt.Sprintf("/api/xiang/import/%s", name),
				"mapping_setting_model": fmt.Sprintf("import_%s_mapping", name),
				"preview_setting_model": fmt.Sprintf("import_%s_preview", name),
			},
			"operation": operation,
		}
	}

	return res, nil
}
