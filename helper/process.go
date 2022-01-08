package helper

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/utils"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.helper.ArrayPluck", ProcessArrayPluck)
	gou.RegisterProcessHandler("xiang.helper.ArraySplit", ProcessArraySplit)
	gou.RegisterProcessHandler("xiang.helper.ArrayColumn", ProcessArrayColumn)
	gou.RegisterProcessHandler("xiang.helper.ArrayKeep", ProcessArrayKeep)
	gou.RegisterProcessHandler("xiang.helper.ArrayTree", ProcessArrayTree)
	gou.RegisterProcessHandler("xiang.helper.ArrayUnique", ProcessArrayUnique)
	gou.RegisterProcessHandler("xiang.helper.ArrayMapSet", ProcessArrayMapSet)

	gou.RegisterProcessHandler("xiang.helper.MapKeys", ProcessMapKeys)
	gou.RegisterProcessHandler("xiang.helper.MapValues", ProcessMapValues)
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
	gou.RegisterProcessHandler("xiang.helper.Each", ProcessEach)
	gou.RegisterProcessHandler("xiang.helper.Case", ProcessCase)
	gou.RegisterProcessHandler("xiang.helper.IF", ProcessIF)

	gou.RegisterProcessHandler("xiang.helper.EnvSet", ProcessEnvSet)
	gou.RegisterProcessHandler("xiang.helper.EnvGet", ProcessEnvGet)
	gou.RegisterProcessHandler("xiang.helper.EnvMultiSet", ProcessEnvMultiSet)
	gou.RegisterProcessHandler("xiang.helper.EnvMultiGet", ProcessEnvMultiGet)

	gou.RegisterProcessHandler("xiang.helper.Print", ProcessPrint)
	gou.RegisterProcessHandler("xiang.helper.Throw", ProcessThrow)

}

// ProcessPrint xiang.helper.Print 打印语句
func ProcessPrint(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	utils.Dump(process.Args...)
	return nil
}
