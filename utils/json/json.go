package json

import (
	"github.com/yaoapp/gou/process"
)

// ProcessValidate utils.json.Validate
// **Warning** This process under developing, do not use it
func ProcessValidate(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	if _, ok := process.Args[0].(map[string]interface{}); !ok {
		return false
	}

	data := process.ArgsMap(0, map[string]interface{}{}).Dot()
	rules := process.ArgsRecords(1)
	for _, rule := range rules {
		for method, value := range rule {
			switch method {
			case "haskey":
				key, ok := value.(string)
				if !ok {
					return false
				}
				if !data.Has(key) {
					return false
				}
			}
		}
	}
	return true
}
