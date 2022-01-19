package helper

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

// ProcessReturn  xiang.helper.Return 返回数值
func ProcessReturn(process *gou.Process) interface{} {
	return process.Args
}

// ProcessThrow  xiang.helper.Throw 抛出异常
func ProcessThrow(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	message := process.ArgsString(0)
	code := process.ArgsInt(1)
	exception.New(message, code).Throw()
	return nil
}
