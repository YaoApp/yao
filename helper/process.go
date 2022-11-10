package helper

import (
	"time"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/utils"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.helper.ArrayGet", ProcessArrayGet)         // deprecated → utils.arr.Get @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.ArrayIndexes", ProcessArrayIndexes) // deprecated → utils.arr.Indexes @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.ArrayPluck", ProcessArrayPluck)     // deprecated → utils.arr.Pluck @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.ArraySplit", ProcessArraySplit)     // deprecated → utils.arr.Split  @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.ArrayColumn", ProcessArrayColumn)   // deprecated → utils.arr.Column  @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.ArrayKeep", ProcessArrayKeep)       // deprecated → utils.arr.Keep  @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.ArrayTree", ProcessArrayTree)       // deprecated → utils.arr.Tree  @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.ArrayUnique", ProcessArrayUnique)   // deprecated → utils.arr.Unique  @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.ArrayMapSet", ProcessArrayMapSet)

	gou.RegisterProcessHandler("xiang.helper.MapKeys", ProcessMapKeys)         // deprecated → utils.map.Keys @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.MapValues", ProcessMapValues)     // deprecated → utils.map.Values @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.MapToArray", ProcessMapToArray)   // deprecated → utils.map.Array @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.MapGet", ProcessMapGet)           // deprecated → utils.map.Get @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.MapSet", ProcessMapSet)           // deprecated → utils.map.Set @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.MapDel", ProcessMapDel)           // deprecated → utils.map.Del @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.MapMultiDel", ProcessMapMultiDel) // deprecated → utils.map.DelMany @/utils/process.go

	gou.RegisterProcessHandler("xiang.helper.HexToString", ProcessHexToString) // deprecated

	gou.RegisterProcessHandler("xiang.helper.StrConcat", ProcessStrConcat) // deprecated → utils.str.Concat @/utils/process.go

	gou.RegisterProcessHandler("xiang.helper.Captcha", ProcessCaptcha)                 // deprecated → utils.captcha.Make @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.CaptchaValidate", ProcessCaptchaValidate) // deprecated → utils.captcha.Verify @/utils/process.go

	gou.RegisterProcessHandler("xiang.helper.PasswordValidate", ProcessPasswordValidate) // deprecated → utils.pwd.Verify @/utils/process.go

	gou.RegisterProcessHandler("xiang.helper.JwtMake", ProcessJwtMake)         // deprecated → utils.jwt.Make  @/utils/process.go
	gou.RegisterProcessHandler("xiang.helper.JwtValidate", ProcessJwtValidate) // deprecated → utils.jwt.Verify  @/utils/process.go

	gou.RegisterProcessHandler("xiang.helper.For", ProcessFor)       // deprecated → utils.flow.For  @/utils/process.go
	gou.AliasProcess("xiang.helper.For", "xiang.flow.For")           // deprecated
	gou.RegisterProcessHandler("xiang.helper.Each", ProcessEach)     // deprecated → utils.flow.Each  @/utils/process.go
	gou.AliasProcess("xiang.helper.Each", "xiang.flow.Each")         // deprecated
	gou.RegisterProcessHandler("xiang.helper.Case", ProcessCase)     // deprecated → utils.flow.Case  @/utils/process.go
	gou.AliasProcess("xiang.helper.Case", "xiang.flow.Case")         // deprecated
	gou.RegisterProcessHandler("xiang.helper.IF", ProcessIF)         // deprecated → utils.flow.IF  @/utils/process.go
	gou.AliasProcess("xiang.helper.IF", "xiang.flow.IF")             // deprecated
	gou.RegisterProcessHandler("xiang.helper.Throw", ProcessThrow)   // deprecated → utils.flow.Throw  @/utils/process.go
	gou.AliasProcess("xiang.helper.Throw", "xiang.flow.Throw")       // deprecated
	gou.RegisterProcessHandler("xiang.helper.Return", ProcessReturn) // deprecated → utils.flow.Return  @/utils/process.go
	gou.AliasProcess("xiang.helper.Return", "xiang.flow.Return")     // deprecated

	gou.RegisterProcessHandler("xiang.helper.EnvSet", ProcessEnvSet) // deprecated → utils.env.Set  @/utils/process.go
	gou.AliasProcess("xiang.helper.EnvSet", "xiang.env.Set")         // deprecated
	gou.AliasProcess("xiang.helper.EnvSet", "yao.env.Set")           // deprecated

	gou.RegisterProcessHandler("xiang.helper.EnvGet", ProcessEnvGet) // deprecated → utils.env.Get  @/utils/process.go
	gou.AliasProcess("xiang.helper.EnvGet", "xiang.env.Get")         // deprecated
	gou.AliasProcess("xiang.helper.EnvGet", "yao.env.Get")           // deprecated

	gou.RegisterProcessHandler("xiang.helper.EnvMultiSet", ProcessEnvMultiSet) // deprecated → utils.env.SetMany  @/utils/process.go
	gou.AliasProcess("xiang.helper.EnvMultiSet", "xiang.env.MultiSet")         // deprecated
	gou.AliasProcess("xiang.helper.EnvMultiSet", "yao.env.MultiSet")           // deprecated

	gou.RegisterProcessHandler("xiang.helper.EnvMultiGet", ProcessEnvMultiGet) // deprecated → utils.env.GetMany  @/utils/process.go
	gou.AliasProcess("xiang.helper.EnvMultiGet", "xiang.env.MultiGet")         // deprecated
	gou.AliasProcess("xiang.helper.EnvMultiGet", "yao.env.MultiGet")           // deprecated

	gou.RegisterProcessHandler("xiang.helper.Print", ProcessPrint) // deprecated → utils.fmt.Println  @/utils/process.go
	gou.AliasProcess("xiang.helper.Print", "xiang.sys.Print")      // deprecated

	gou.RegisterProcessHandler("xiang.flow.Sleep", ProcessSleep) // deprecated → utils.time.Sleep  @/utils/process.go
	gou.AliasProcess("xiang.flow.Sleep", "xiang.sys.Sleep")      // deprecated
	gou.AliasProcess("xiang.flow.Sleep", "yao.sys.Sleep")        // deprecated

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
