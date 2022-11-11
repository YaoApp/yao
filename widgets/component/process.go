package component

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
)

// Export process
func exportProcess() {
	gou.RegisterProcessHandler("yao.component.selectoptions", processSelectOptions)
}

func processSelectOptions(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	query := process.ArgsMap(0, map[string]interface{}{})
	if !query.Has("model") {
		exception.New("query.model required", 400).Throw()
	}

	model, ok := query.Get("model").(string)
	if !ok {
		exception.New("query.model must be a string", 400).Throw()
	}

	m := gou.Select(model)

	valueField := query.Get("value")
	if valueField == nil {
		valueField = "id"
	}
	value, ok := valueField.(string)
	if !ok {
		exception.New("query.value must be a string", 400).Throw()
	}

	labelField := query.Get("label")
	if labelField == nil {
		labelField = "name"
	}
	label, ok := labelField.(string)
	if !ok {
		exception.New("query.label must be a string", 400).Throw()
	}

	limit := 500
	if query.Get("limit") != nil {
		v := any.Of(query.Get("limit"))
		if v.IsInt() || v.IsString() {
			limit = v.CInt()
		}
	}

	wheres := []gou.QueryWhere{}
	switch input := query.Get("wheres").(type) {
	case string:
		where := gou.QueryWhere{}
		err := jsoniter.Unmarshal([]byte(input), &where)
		if err != nil {
			exception.New("query.wheres error %s", 400, err.Error()).Throw()
		}
		wheres = append(wheres, where)
		break

	case []string:
		for _, data := range input {
			where := gou.QueryWhere{}
			err := jsoniter.Unmarshal([]byte(data), &where)
			if err != nil {
				exception.New("query.wheres error %s", 400, err.Error()).Throw()
			}
			wheres = append(wheres, where)
		}
		break
	}

	if data, ok := query.Get("wheres").(string); ok {
		data = strings.TrimSpace(data)
		if strings.HasPrefix(data, "{") && strings.HasSuffix(data, "}") {
			data = fmt.Sprintf("[%s]", data)
		}
		err := jsoniter.Unmarshal([]byte(data), &wheres)
		if err != nil {
			exception.New("query.wheres error %s", 400, err.Error()).Throw()
		}
	}

	rows, err := m.Get(gou.QueryParam{
		Select: []interface{}{valueField, labelField},
		Wheres: wheres,
		Limit:  limit,
	})
	if err != nil {
		exception.New("%s", 500, err.Error()).Throw()
	}

	res := []map[string]interface{}{}
	for _, row := range rows {
		res = append(res, map[string]interface{}{
			"label": row.Get(label),
			"value": row.Get(value),
		})
	}
	return res
}
