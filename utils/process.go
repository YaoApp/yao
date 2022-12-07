package utils

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/utils/datetime"
	"github.com/yaoapp/yao/utils/str"
	"github.com/yaoapp/yao/utils/tree"
)

func init() {
	gou.AliasProcess("xiang.helper.Captcha", "yao.utils.Captcha")                 // deprecated
	gou.AliasProcess("xiang.helper.CaptchaValidate", "yao.utils.CaptchaValidate") // deprecated

	// ****************************************
	// * Migrate Processes Version 0.10.2+
	// ****************************************

	// Application
	gou.AliasProcess("xiang.main.Ping", "utils.app.Ping")
	gou.AliasProcess("xiang.main.Inspect", "utils.app.Inspect")

	// FMT
	gou.AliasProcess("xiang.helper.Print", "utils.fmt.Print")

	// ENV
	gou.AliasProcess("xiang.helper.EnvSet", "utils.env.Get")
	gou.AliasProcess("xiang.helper.EnvGet", "utils.env.Get")
	gou.AliasProcess("xiang.helper.EnvMultiSet", "utils.env.SetMany")
	gou.AliasProcess("xiang.helper.EnvMultiGet", "utils.env.GetMany")

	// Flow
	gou.AliasProcess("xiang.helper.For", "utils.flow.For")
	gou.AliasProcess("xiang.helper.Each", "utils.flow.Each")
	gou.AliasProcess("xiang.helper.Case", "utils.flow.Case")
	gou.AliasProcess("xiang.helper.IF", "utils.flow.IF")
	gou.AliasProcess("xiang.helper.Throw", "utils.flow.Throw")
	gou.AliasProcess("xiang.helper.Return", "utils.flow.Return")

	// JWT
	gou.AliasProcess("xiang.helper.JwtMake", "utils.jwt.Make")
	gou.AliasProcess("xiang.helper.JwtValidate", "utils.jwt.Verify")

	// Password
	// utils.pwd.Hash
	gou.AliasProcess("xiang.helper.PasswordValidate", "utils.pwd.Verify")

	// Captcha
	gou.AliasProcess("xiang.helper.Captcha", "utils.captcha.Make")
	gou.AliasProcess("xiang.helper.CaptchaValidate", "utils.captcha.Verify")

	// String
	gou.AliasProcess("xiang.helper.StrConcat", "utils.str.Concat")
	gou.RegisterProcessHandler("utils.str.Join", str.ProcessJoin)
	gou.RegisterProcessHandler("utils.str.JoinPath", str.ProcessJoinPath)

	// Array
	gou.AliasProcess("xiang.helper.ArrayPluck", "utils.arr.Pluck")
	gou.AliasProcess("xiang.helper.ArraySplit", "utils.arr.Split")
	gou.AliasProcess("xiang.helper.ArrayTree", "utils.arr.Tree")
	gou.AliasProcess("xiang.helper.ArrayUnique", "utils.arr.Unique")
	gou.AliasProcess("xiang.helper.ArrayIndexes", "utils.arr.Indexes")
	gou.AliasProcess("xiang.helper.ArrayGet", "utils.arr.Get")
	gou.AliasProcess("xiang.helper.ArrayColumn", "utils.arr.Column") // doc
	gou.AliasProcess("xiang.helper.ArrayKeep", "utils.arr.Keep")
	gou.AliasProcess("xiang.helper.ArrayMapSet", "utils.arr.MapSet")

	// Tree
	gou.RegisterProcessHandler("utils.tree.Flatten", tree.ProcessFlatten)

	// Map
	gou.AliasProcess("xiang.helper.MapGet", "utils.map.Get")
	gou.AliasProcess("xiang.helper.MapSet", "utils.map.Set")
	gou.AliasProcess("xiang.helper.MapDel", "utils.map.Del")
	gou.AliasProcess("xiang.helper.MapDel", "utils.map.DelMany")
	gou.AliasProcess("xiang.helper.MapKeys", "utils.map.Keys")
	gou.AliasProcess("xiang.helper.MapValues", "utils.map.Values")
	gou.AliasProcess("xiang.helper.MapToArray", "utils.map.Array") // doc
	// utils.map.Merge

	// Time
	gou.AliasProcess("xiang.flow.Sleep", "utils.time.Sleep")
	gou.RegisterProcessHandler("utils.now.Time", datetime.ProcessTime)
	gou.RegisterProcessHandler("utils.now.Date", datetime.ProcessDate)
	gou.RegisterProcessHandler("utils.now.DateTime", datetime.ProcessDateTime)
	gou.RegisterProcessHandler("utils.now.Timestamp", datetime.ProcessTimestamp)
	gou.RegisterProcessHandler("utils.now.Timestampms", datetime.ProcessTimestampms)
}
