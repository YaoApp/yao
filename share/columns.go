package share

import (
	"fmt"

	"github.com/yaoapp/gou/model"
)

var elms = map[string]Column{
	"string":               {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"char":                 {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"text":                 {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"mediumText":           {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"longText":             {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"binary":               {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"date":                 {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"datetime":             {View: Render{Type: "label"}, Edit: Render{Type: "datetime"}},
	"datetimeTz":           {View: Render{Type: "label"}, Edit: Render{Type: "datetime"}},
	"time":                 {View: Render{Type: "label"}, Edit: Render{Type: "time"}},
	"timeTz":               {View: Render{Type: "label"}, Edit: Render{Type: "time"}},
	"timestamp":            {View: Render{Type: "label"}, Edit: Render{Type: "datetime"}},
	"timestampTz":          {View: Render{Type: "label"}, Edit: Render{Type: "datetime"}},
	"tinyInteger":          {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"tinyIncrements":       {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"unsignedTinyInteger":  {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"smallInteger":         {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"smallIncrements":      {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"unsignedSmallInteger": {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"integer":              {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"increments":           {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"unsignedInteger":      {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"bigInteger":           {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"bigIncrements":        {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"unsignedBigInteger":   {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"id":                   {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"ID":                   {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"decimal":              {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"unsignedDecimal":      {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"float":                {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"unsignedFloat":        {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"double":               {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"unsignedDouble":       {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"boolean":              {View: Render{Type: "label"}, Edit: Render{Type: "checkbox"}},
	"enum":                 {View: Render{Type: "label"}, Edit: Render{Type: "select"}},
	"json":                 {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"JSON":                 {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"jsonb":                {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"JSONB":                {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"uuid":                 {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"ipAddress":            {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"macAddress":           {View: Render{Type: "label"}, Edit: Render{Type: "input"}},
	"year":                 {View: Render{Type: "label"}, Edit: Render{Type: "datetime"}},
}

// GetDefaultColumns 读取数据模型字段的呈现方式
func GetDefaultColumns(name string) map[string]Column {
	mod := model.Select(name)
	cmap := mod.Columns
	columns := map[string]Column{}

	for name, col := range cmap {
		vcol, has := elms[col.Type]
		if !has {
			continue
		}

		label := col.Label
		if label == "" {
			label = col.Comment
		}
		if label == "" {
			label = name
		}

		vcol.Label = label
		if vcol.View.Props == nil {
			vcol.View.Props = map[string]interface{}{}
		}
		if vcol.Edit.Props == nil {
			vcol.Edit.Props = map[string]interface{}{}
		}
		vcol.View.Props["value"] = fmt.Sprintf(":%s", col.Name)
		vcol.Edit.Props["value"] = fmt.Sprintf(":%s", col.Name)

		// 枚举型
		if col.Type == "enum" {
			options := []map[string]string{}
			for _, opt := range col.Option {
				options = append(options, map[string]string{
					"label": opt,
					"value": opt,
				})
			}
			vcol.Edit.Props["options"] = options
		}

		columns[name] = vcol
		columns[label] = vcol
	}
	return columns
}
