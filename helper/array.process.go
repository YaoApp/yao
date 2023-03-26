package helper

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/maps"
)

// ProcessArrayPluck  xiang.helper.ArrayPluck 将多个数据记录集合，合并为一个数据记录集合
func ProcessArrayPluck(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	columns := process.ArgsStrings(0)
	pluck := process.ArgsMap(1)
	return ArrayPluck(columns, pluck)
}

// ProcessArraySplit  xiang.helper.ArraySplit 将多条数记录集合，分解为一个 columns:[]string 和 values: [][]interface{}
func ProcessArraySplit(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	records := process.ArgsRecords(0)
	columns, values := ArraySplit(records)
	return map[string]interface{}{
		"columns": columns,
		"values":  values,
	}
}

// ProcessArrayColumn  xiang.helper.ArrayColumn  返回多条数据记录，指定字段数值。
func ProcessArrayColumn(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	records := process.ArgsRecords(0)
	name := process.ArgsString(1)
	values := ArrayColumn(records, name)
	return values
}

// ProcessArrayKeep  xiang.helper.ArrayKeep  仅保留指定键名的数据
func ProcessArrayKeep(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	records := process.ArgsRecords(0)
	columns := process.ArgsStrings(1)
	return ArrayKeep(records, columns)
}

// ProcessArrayTree  xiang.helper.ArrayTree  转换为属性结构
func ProcessArrayTree(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	records := process.ArgsRecords(0)
	setting := process.ArgsMap(1)
	return ArrayTree(records, setting)
}

// ProcessArrayUnique  xiang.helper.ArrayUnique 数组排重
func ProcessArrayUnique(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	if arr, ok := process.Args[0].([]interface{}); ok {
		return ArrayUnique(arr)
	}
	return process.Args[0]
}

// ProcessArrayMapSet  xiang.helper.ArrayMapSet 数组映射设定数值
func ProcessArrayMapSet(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	arr, ok := process.Args[0].([]map[string]interface{})
	if ok {
		return ArrayMapSet(arr, process.ArgsString(1), process.Args[2])
	} else if arr2, ok := process.Args[0].([]maps.MapStr); ok {
		return ArrayMapSetMapStr(arr2, process.ArgsString(1), process.Args[2])
	}
	return process.Args[0]
}

// ProcessArrayIndexes xiang.helper.ArrayIndexes 返回数组索引。
func ProcessArrayIndexes(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	records := process.ArgsArray(0)
	res := []int{}
	for index := range records {
		res = append(res, index)
	}
	return res
}

// ProcessArrayGet xiang.helper.ArrayGet 返回指定索引数据
func ProcessArrayGet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	records := process.ArgsArray(0)
	index := process.ArgsInt(1)
	if index >= len(records) {
		return nil
	}
	return records[index]
}
