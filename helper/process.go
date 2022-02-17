package helper

import (
	"time"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/utils"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.helper.ArrayGet", ProcessArrayGet)
	gou.RegisterProcessHandler("xiang.helper.ArrayIndexes", ProcessArrayIndexes)
	gou.RegisterProcessHandler("xiang.helper.ArrayPluck", ProcessArrayPluck)
	gou.RegisterProcessHandler("xiang.helper.ArraySplit", ProcessArraySplit)
	gou.RegisterProcessHandler("xiang.helper.ArrayColumn", ProcessArrayColumn)
	gou.RegisterProcessHandler("xiang.helper.ArrayKeep", ProcessArrayKeep)
	gou.RegisterProcessHandler("xiang.helper.ArrayTree", ProcessArrayTree)
	gou.RegisterProcessHandler("xiang.helper.ArrayUnique", ProcessArrayUnique)
	gou.RegisterProcessHandler("xiang.helper.ArrayMapSet", ProcessArrayMapSet)

	gou.RegisterProcessHandler("xiang.helper.MapKeys", ProcessMapKeys)
	gou.RegisterProcessHandler("xiang.helper.MapValues", ProcessMapValues)
	gou.RegisterProcessHandler("xiang.helper.MapToArray", ProcessMapToArray)
	gou.RegisterProcessHandler("xiang.helper.MapGet", ProcessMapGet)
	gou.RegisterProcessHandler("xiang.helper.MapSet", ProcessMapSet)
	gou.RegisterProcessHandler("xiang.helper.MapDel", ProcessMapDel)
	gou.RegisterProcessHandler("xiang.helper.MapMultiDel", ProcessMapMultiDel)

	gou.RegisterProcessHandler("xiang.helper.StrConcat", ProcessStrConcat)

	gou.RegisterProcessHandler("xiang.helper.Captcha", ProcessCaptcha)
	gou.RegisterProcessHandler("xiang.helper.CaptchaValidate", ProcessCaptchaValidate)

	gou.RegisterProcessHandler("xiang.helper.PasswordValidate", ProcessPasswordValidate)

	gou.RegisterProcessHandler("xiang.helper.JwtMake", ProcessJwtMake)
	gou.RegisterProcessHandler("xiang.helper.JwtValidate", ProcessJwtValidate)

	gou.RegisterProcessHandler("xiang.helper.For", ProcessFor)
	gou.AliasProcess("xiang.helper.For", "xiang.flow.For")
	gou.RegisterProcessHandler("xiang.helper.Each", ProcessEach)
	gou.AliasProcess("xiang.helper.Each", "xiang.flow.Each")
	gou.RegisterProcessHandler("xiang.helper.Case", ProcessCase)
	gou.AliasProcess("xiang.helper.Case", "xiang.flow.Case")
	gou.RegisterProcessHandler("xiang.helper.IF", ProcessIF)
	gou.AliasProcess("xiang.helper.IF", "xiang.flow.IF")
	gou.RegisterProcessHandler("xiang.helper.Throw", ProcessThrow)
	gou.AliasProcess("xiang.helper.Throw", "xiang.flow.Throw")
	gou.RegisterProcessHandler("xiang.helper.Return", ProcessReturn)
	gou.AliasProcess("xiang.helper.Return", "xiang.flow.Return")

	gou.RegisterProcessHandler("xiang.helper.EnvSet", ProcessEnvSet)
	gou.AliasProcess("xiang.helper.EnvSet", "xiang.env.Set")
	gou.RegisterProcessHandler("xiang.helper.EnvGet", ProcessEnvGet)
	gou.AliasProcess("xiang.helper.EnvGet", "xiang.env.Get")
	gou.RegisterProcessHandler("xiang.helper.EnvMultiSet", ProcessEnvMultiSet)
	gou.AliasProcess("xiang.helper.EnvMultiSet", "xiang.env.MultiSet")
	gou.RegisterProcessHandler("xiang.helper.EnvMultiGet", ProcessEnvMultiGet)
	gou.AliasProcess("xiang.helper.EnvMultiGet", "xiang.env.MultiGet")

	gou.RegisterProcessHandler("xiang.helper.Print", ProcessPrint)
	gou.AliasProcess("xiang.helper.Print", "xiang.sys.Print")

	gou.RegisterProcessHandler("xiang.flow.Sleep", ProcessSleep)
	gou.AliasProcess("xiang.flow.Sleep", "xiang.sys.Sleep")
}

// ProcessPrint xiang.helper.Print 打印语句
func ProcessPrint(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	utils.Dump(process.Args...)
	return nil
}

// ProcessSleep xiang.flow.Sleep 等待
func ProcessSleep(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	ms := process.ArgsInt(0)
	time.Sleep(time.Duration((ms * int(time.Millisecond))))
	return nil
}
