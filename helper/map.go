package helper

import "github.com/yaoapp/gou"

// MapValues 返回映射的数值
func MapValues(record map[string]interface{}) []interface{} {
	values := []interface{}{}
	for _, value := range record {
		values = append(values, value)
	}
	return values
}

// MapKeys 返回映射的键
func MapKeys(record map[string]interface{}) []string {
	keys := []string{}
	for key := range record {
		keys = append(keys, key)
	}
	return keys
}

// ProcessMapValues  xiang.helper.MapValues 返回映射的数值
func ProcessMapValues(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	record := process.ArgsMap(0)
	return MapValues(record)
}

// ProcessMapKeys  xiang.helper.MapKeys 返回映射的键
func ProcessMapKeys(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	record := process.ArgsMap(0)
	return MapKeys(record)
}
