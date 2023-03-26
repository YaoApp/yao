package helper

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessReturn  xiang.helper.Return 返回数值
func ProcessReturn(process *process.Process) interface{} {
	return process.Args
}

// ProcessThrow  xiang.helper.Throw 抛出异常
func ProcessThrow(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	message := process.ArgsString(0)
	code := process.ArgsInt(1)
	exception.New(message, code).Throw()
	return nil
}
