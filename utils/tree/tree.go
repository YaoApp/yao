package tree

import (
	"fmt"

	"github.com/yaoapp/gou/process"
)

// ProcessFlatten utils.tree.Flatten cast to array
func ProcessFlatten(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	array := process.ArgsArray(0)
	option := process.ArgsMap(1, map[string]interface{}{"primary": "id", "children": "children", "parent": "parent"})
	if _, has := option["primary"]; !has {
		option["primary"] = "id"
	}

	if _, has := option["children"]; !has {
		option["children"] = "children"
	}

	if _, has := option["parent"]; !has {
		option["parent"] = "parent"
	}

	return flatten(array, option, nil)
}

func flatten(array []interface{}, option map[string]interface{}, id interface{}) []interface{} {

	parent := fmt.Sprintf("%v", option["parent"])
	primary := fmt.Sprintf("%v", option["primary"])
	childrenField := fmt.Sprintf("%v", option["children"])
	res := []interface{}{}
	for _, v := range array {
		row, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		row[parent] = id
		children, ok := row[childrenField].([]interface{})
		delete(row, childrenField)
		res = append(res, row)

		if ok {
			res = append(res, flatten(children, option, row[primary])...)

		}
	}
	return res
}
