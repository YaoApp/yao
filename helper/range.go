package helper

import (
	"reflect"
	"regexp"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
)

var reVar = regexp.MustCompile("::([a-z]+)") // ::key, ::value

// Process 处理器参数
type Process struct {
	Process string        `json:"process"`
	Args    []interface{} `json:"args,omitempty"`
}

// Range 过程控制
func Range(v interface{}, process Process) {
	value := reflect.ValueOf(v)
	value = reflect.Indirect(value)
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		For(0, any.Of(v).CInt(), process)
		return
	case reflect.Array, reflect.Slice:
		data, err := jsoniter.Marshal(v)
		if err != nil {
			exception.New("数值格式不能使用Range %s", 400, err.Error()).Throw()
		}
		v := []interface{}{}
		err = jsoniter.Unmarshal(data, &v)
		rangeArray(v, process)
		return
	case reflect.String:
		return
	case reflect.Map:
		data, err := jsoniter.Marshal(v)
		if err != nil {
			exception.New("数值格式不能使用Range %s", 400, err.Error()).Throw()
		}
		v := map[string]interface{}{}
		err = jsoniter.Unmarshal(data, &v)
		rangeMap(v, process)
		return
	case reflect.Struct:
		data, err := jsoniter.Marshal(v)
		if err != nil {
			exception.New("数值格式不能使用Range %s", 400, err.Error()).Throw()
		}
		v := map[string]interface{}{}
		err = jsoniter.Unmarshal(data, &v)
		rangeMap(v, process)
		return
	}

	exception.New("数值格式不能使用Range", 400).Ctx([]interface{}{v, value.Kind()}).Throw()
}

// For 过程控制
func For(from int, to int, p Process) {
	for i := from; i < to; i++ {
		bindings := map[string]interface{}{
			"key":   i,
			"value": i,
		}
		args := bindArgs(p.Args, bindings)
		process.New(p.Process, args...).Run()
	}
}

func bindArgs(args []interface{}, bindings map[string]interface{}) []interface{} {
	new := []interface{}{}
	for i := range args {
		new = append(new, helper.Bind(args[i], bindings, reVar))
	}
	return new
}

func rangeString(v string, p Process) {
	var bytes = []byte(v)
	for key, value := range bytes {
		bindings := map[string]interface{}{
			"key":   key,
			"value": value,
		}
		args := bindArgs(p.Args, bindings)
		process.New(p.Process, args...).Run()
	}

}

func rangeMap(v map[string]interface{}, p Process) {
	for key, value := range v {
		bindings := map[string]interface{}{
			"key":   key,
			"value": value,
		}
		args := bindArgs(p.Args, bindings)
		process.New(p.Process, args...).Run()
	}
}

func rangeArray(v []interface{}, p Process) {
	for key, value := range v {
		bindings := map[string]interface{}{
			"key":   key,
			"value": value,
		}
		args := bindArgs(p.Args, bindings)
		process.New(p.Process, args...).Run()
	}
}

// ProcessOf 转换映射表
func ProcessOf(v map[string]interface{}) Process {
	process, ok := v["process"]
	if !ok {
		exception.New("参数错误: 缺少 process", 400).Throw()
	}

	processStr, ok := process.(string)
	if !ok {
		exception.New("参数错误: process 应该为字符串 ", 400).Throw()
	}

	if args, ok := v["args"].([]interface{}); ok {
		return Process{
			Process: processStr,
			Args:    args,
		}
	}
	return Process{
		Process: processStr,
		Args:    []interface{}{},
	}
}

// ProcessEach  xiang.helper.Each 循环过程控制
func ProcessEach(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	v := process.Args[0]
	p := ProcessOf(process.ArgsMap(1))
	Range(v, p)
	return nil
}

// ProcessFor xiang.helper.For 循环过程控制
func ProcessFor(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	from := process.ArgsInt(0)
	to := process.ArgsInt(1)
	p := ProcessOf(process.ArgsMap(2))
	For(from, to, p)
	return nil
}
