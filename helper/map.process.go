package helper

import (
	"fmt"

	"github.com/yaoapp/gou/process"
)

// ProcessMapValues  xiang.helper.MapValues 返回映射表的数值
func ProcessMapValues(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	record := process.ArgsMap(0)
	return MapValues(record)
}

// ProcessMapKeys  xiang.helper.MapKeys 返回映射表的键
func ProcessMapKeys(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	record := process.ArgsMap(0)
	return MapKeys(record)
}

// ProcessMapGet  xiang.helper.MapGet 返回映射表给定键的值
func ProcessMapGet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	record := process.ArgsMap(0)
	key := process.ArgsString(1)
	return MapGet(record, key)
}

// ProcessMapSet  xiang.helper.MapSet 设定键值,返回映射表给定键的值
func ProcessMapSet(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	record := process.ArgsMap(0)
	key := process.ArgsString(1)
	value := process.Args[2]
	return MapSet(record, key, value)
}

// ProcessMapDel  xiang.helper.MapDel 删除给定键, 返回映射表
func ProcessMapDel(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	record := process.ArgsMap(0)
	key := process.ArgsString(1)
	return MapDel(record, key)
}

// ProcessMapMultiDel  xiang.helper.MapMultiDel  删除一组给定键, 返回映射表
func ProcessMapMultiDel(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	record := process.ArgsMap(0)
	keys := []string{}
	for _, key := range process.Args {
		keys = append(keys, fmt.Sprintf("%v", key))
	}
	return MapMultiDel(record, keys...)
}

// ProcessMapToArray  xiang.helper.MapToArray  映射转换为 KeyValue 数组
func ProcessMapToArray(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	m := process.ArgsMap(0)
	res := []map[string]interface{}{}
	for key, value := range m {
		res = append(res, map[string]interface{}{
			"key":   key,
			"value": value,
		})
	}
	return res
}
