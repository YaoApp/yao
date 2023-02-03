package helper

import (
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/utils"
)

func init() {
	// 注册处理器
	process.Register("xiang.helper.ArrayGet", ProcessArrayGet)         // deprecated → utils.arr.Get @/utils/process.go
	process.Register("xiang.helper.ArrayIndexes", ProcessArrayIndexes) // deprecated → utils.arr.Indexes @/utils/process.go
	process.Register("xiang.helper.ArrayPluck", ProcessArrayPluck)     // deprecated → utils.arr.Pluck @/utils/process.go
	process.Register("xiang.helper.ArraySplit", ProcessArraySplit)     // deprecated → utils.arr.Split  @/utils/process.go
	process.Register("xiang.helper.ArrayColumn", ProcessArrayColumn)   // deprecated → utils.arr.Column  @/utils/process.go
	process.Register("xiang.helper.ArrayKeep", ProcessArrayKeep)       // deprecated → utils.arr.Keep  @/utils/process.go
	process.Register("xiang.helper.ArrayTree", ProcessArrayTree)       // deprecated → utils.arr.Tree  @/utils/process.go
	process.Register("xiang.helper.ArrayUnique", ProcessArrayUnique)   // deprecated → utils.arr.Unique  @/utils/process.go
	process.Register("xiang.helper.ArrayMapSet", ProcessArrayMapSet)   // deprecated → utils.arr.MapSet  @/utils/process.go

	process.Register("xiang.helper.MapKeys", ProcessMapKeys)         // deprecated → utils.map.Keys @/utils/process.go
	process.Register("xiang.helper.MapValues", ProcessMapValues)     // deprecated → utils.map.Values @/utils/process.go
	process.Register("xiang.helper.MapToArray", ProcessMapToArray)   // deprecated → utils.map.Array @/utils/process.go
	process.Register("xiang.helper.MapGet", ProcessMapGet)           // deprecated → utils.map.Get @/utils/process.go
	process.Register("xiang.helper.MapSet", ProcessMapSet)           // deprecated → utils.map.Set @/utils/process.go
	process.Register("xiang.helper.MapDel", ProcessMapDel)           // deprecated → utils.map.Del @/utils/process.go
	process.Register("xiang.helper.MapMultiDel", ProcessMapMultiDel) // deprecated → utils.map.DelMany @/utils/process.go

	process.Register("xiang.helper.HexToString", ProcessHexToString) // deprecated → utils.str.Hex @/utils/process.go new 2022.2.3

	process.Register("xiang.helper.StrConcat", ProcessStrConcat) // deprecated → utils.str.Concat @/utils/process.go

	process.Register("xiang.helper.Captcha", ProcessCaptcha)                 // deprecated → utils.captcha.Make @/utils/process.go
	process.Register("xiang.helper.CaptchaValidate", ProcessCaptchaValidate) // deprecated → utils.captcha.Verify @/utils/process.go

	process.Register("xiang.helper.PasswordValidate", ProcessPasswordValidate) // deprecated → utils.pwd.Verify @/utils/process.go

	process.Register("xiang.helper.JwtMake", ProcessJwtMake)         // deprecated → utils.jwt.Make  @/utils/process.go
	process.Register("xiang.helper.JwtValidate", ProcessJwtValidate) // deprecated → utils.jwt.Verify  @/utils/process.go

	process.Register("xiang.helper.For", ProcessFor)          // deprecated → utils.flow.For  @/utils/process.go
	process.Alias("xiang.helper.For", "xiang.flow.For")       // deprecated
	process.Register("xiang.helper.Each", ProcessEach)        // deprecated → utils.flow.Each  @/utils/process.go
	process.Alias("xiang.helper.Each", "xiang.flow.Each")     // deprecated
	process.Register("xiang.helper.Case", ProcessCase)        // deprecated → utils.flow.Case  @/utils/process.go
	process.Alias("xiang.helper.Case", "xiang.flow.Case")     // deprecated
	process.Register("xiang.helper.IF", ProcessIF)            // deprecated → utils.flow.IF  @/utils/process.go
	process.Alias("xiang.helper.IF", "xiang.flow.IF")         // deprecated
	process.Register("xiang.helper.Throw", ProcessThrow)      // deprecated → utils.flow.Throw  @/utils/process.go
	process.Alias("xiang.helper.Throw", "xiang.flow.Throw")   // deprecated
	process.Register("xiang.helper.Return", ProcessReturn)    // deprecated → utils.flow.Return  @/utils/process.go
	process.Alias("xiang.helper.Return", "xiang.flow.Return") // deprecated

	process.Register("xiang.helper.EnvSet", ProcessEnvSet) // deprecated → utils.env.Set  @/utils/process.go
	process.Alias("xiang.helper.EnvSet", "xiang.env.Set")  // deprecated
	process.Alias("xiang.helper.EnvSet", "yao.env.Set")    // deprecated

	process.Register("xiang.helper.EnvGet", ProcessEnvGet) // deprecated → utils.env.Get  @/utils/process.go
	process.Alias("xiang.helper.EnvGet", "xiang.env.Get")  // deprecated
	process.Alias("xiang.helper.EnvGet", "yao.env.Get")    // deprecated

	process.Register("xiang.helper.EnvMultiSet", ProcessEnvMultiSet) // deprecated → utils.env.SetMany  @/utils/process.go
	process.Alias("xiang.helper.EnvMultiSet", "xiang.env.MultiSet")  // deprecated
	process.Alias("xiang.helper.EnvMultiSet", "yao.env.MultiSet")    // deprecated

	process.Register("xiang.helper.EnvMultiGet", ProcessEnvMultiGet) // deprecated → utils.env.GetMany  @/utils/process.go
	process.Alias("xiang.helper.EnvMultiGet", "xiang.env.MultiGet")  // deprecated
	process.Alias("xiang.helper.EnvMultiGet", "yao.env.MultiGet")    // deprecated

	process.Register("xiang.helper.Print", ProcessPrint)   // deprecated → utils.fmt.Println  @/utils/process.go
	process.Alias("xiang.helper.Print", "xiang.sys.Print") // deprecated

	process.Register("xiang.flow.Sleep", ProcessSleep)   // deprecated → utils.time.Sleep  @/utils/process.go
	process.Alias("xiang.flow.Sleep", "xiang.sys.Sleep") // deprecated
	process.Alias("xiang.flow.Sleep", "yao.sys.Sleep")   // deprecated

}

// ProcessPrint xiang.helper.Print 打印语句
func ProcessPrint(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	utils.Dump(process.Args...)
	return nil
}

// ProcessSleep xiang.flow.Sleep 等待
func ProcessSleep(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	ms := process.ArgsInt(0)
	time.Sleep(time.Duration((ms * int(time.Millisecond))))
	return nil
}
