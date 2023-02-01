package helper

import (
	"errors"
	"fmt"
	"os"

	"github.com/yaoapp/gou/process"
)

// ProcessEnvGet  xiang.helper.EnvGet 读取ENV
func ProcessEnvGet(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	return os.Getenv(name)
}

// ProcessEnvSet  xiang.helper.EnvSet 设置ENV
func ProcessEnvSet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	value := process.ArgsString(1)
	return os.Setenv(name, value)
}

// ProcessEnvMultiGet  xiang.helper.MultiGet 读取ENV
func ProcessEnvMultiGet(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	res := map[string]string{}
	for i := range process.Args {
		name := fmt.Sprintf("%v", process.Args[i])
		res[name] = os.Getenv(name)
	}
	return res
}

// ProcessEnvMultiSet xiang.helper.MultiSet 设置ENV
func ProcessEnvMultiSet(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	envs := process.ArgsMap(0)
	message := ""
	for name, value := range envs {
		err := os.Setenv(name, fmt.Sprintf("%v", value))
		if err != nil {
			message = message + ";" + err.Error()
		}
	}
	if message != "" {
		return errors.New(message)
	}
	return nil
}
