package expression

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
)

// Export process
func exportProcess() {
	process.Register("yao.expression.selectoption", processSelectOption)
	process.Register("yao.expression.trimspace", processTrimSpace)
}

func processSelectOption(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	input := process.Args[0]
	switch input.(type) {

	case string:
		options := []map[string]interface{}{}
		opts := strings.Split(input.(string), ",")
		for _, opt := range opts {
			options = append(options, map[string]interface{}{
				"label": fmt.Sprintf("::%s", strings.TrimSpace(opt)),
				"value": strings.TrimSpace(opt),
			})
		}
		return options

	case []interface{}:
		options := []map[string]interface{}{}
		opts := input.([]interface{})
		for _, opt := range opts {
			switch opt.(type) {
			case string, int, int64, int32, int8, float32, float64:
				options = append(options, map[string]interface{}{
					"label": fmt.Sprintf("::%s", strings.TrimSpace(fmt.Sprintf("%v", opt))),
					"value": strings.TrimSpace(fmt.Sprintf("%v", opt)),
				})
				break

			case map[string]interface{}, maps.MapStr:
				key := "name"
				value := "id"

				if process.NumOfArgs() > 1 {
					key = process.ArgsString(1)
				}

				if process.NumOfArgs() > 2 {
					value = process.ArgsString(2)
				}

				row := any.Of(opt).MapStr()
				options = append(options, map[string]interface{}{
					"label": fmt.Sprintf("::%s", row.Get(key)),
					"value": row.Get(value),
				})
				break
			}
		}
		return options
	}

	return []map[string]interface{}{}
}

func processTrimSpace(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	input := process.ArgsString(0)
	return strings.TrimSpace(input)
}
