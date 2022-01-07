package helper

import (
	"os"

	"github.com/yaoapp/gou"
)

// ProcessEnvGet  xiang.helper.EnvGet 读取ENV
func ProcessEnvGet(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	return os.Getenv(name)
}

// ProcessEnvSet  xiang.helper.EnvSet 设置ENV
func ProcessEnvSet(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	value := process.ArgsString(1)
	return os.Setenv(name, value)
}
