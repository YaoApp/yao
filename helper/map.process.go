package helper

import "github.com/yaoapp/gou"

// ProcessMapValues  xiang.helper.MapValues 返回映射表的数值
func ProcessMapValues(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	record := process.ArgsMap(0)
	return MapValues(record)
}

// ProcessMapKeys  xiang.helper.MapKeys 返回映射表的键
func ProcessMapKeys(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	record := process.ArgsMap(0)
	return MapKeys(record)
}

// ProcessMapGet  xiang.helper.MapKey 返回映射表给定键的值
func ProcessMapGet(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	record := process.ArgsMap(0)
	key := process.ArgsString(1)
	return MapGet(record, key)
}
