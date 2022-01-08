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

// ProcessMapGet  xiang.helper.MapGet 返回映射表给定键的值
func ProcessMapGet(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	record := process.ArgsMap(0)
	key := process.ArgsString(1)
	return MapGet(record, key)
}

// ProcessMapSet  xiang.helper.MapSet 返回映射表给定键的值
func ProcessMapSet(process *gou.Process) interface{} {
	process.ValidateArgNums(3)
	record := process.ArgsMap(0)
	key := process.ArgsString(1)
	value := process.Args[2]
	return MapSet(record, key, value)
}

// ProcessMapDel  xiang.helper.MapDel 返回映射表给定键的值
func ProcessMapDel(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	record := process.ArgsMap(0)
	key := process.ArgsString(1)
	return MapDel(record, key)
}
