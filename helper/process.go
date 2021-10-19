package helper

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.helper.ArrayPluck", ProcessArrayPluck)

}

// ProcessArrayPluck  xiang.helper.ArrayPluck 将多个数据记录集合，合并为一个数据记录集合
func ProcessArrayPluck(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	columnsAny := process.Args[0]
	columns := []string{}
	switch columnsAny.(type) {
	case []interface{}:
		for _, v := range columnsAny.([]interface{}) {
			value, ok := v.(string)
			if ok {
				columns = append(columns, value)
				continue
			}
			exception.New("参数错误: 第1个参数不是字符串数组", 400).Ctx(process.Args[0]).Throw()
		}
	case []string:
	default:
		exception.New("参数错误: 第1个参数不是字符串数组", 400).Ctx(process.Args[0]).Throw()
		break
	}
	pluck := process.ArgsMap(1)
	return ArrayPluck(columns, pluck)
}
