package utils

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/utils/datetime"
	"github.com/yaoapp/yao/utils/fmt"
	"github.com/yaoapp/yao/utils/json"
	"github.com/yaoapp/yao/utils/str"
	"github.com/yaoapp/yao/utils/throw"
	"github.com/yaoapp/yao/utils/tree"
	"github.com/yaoapp/yao/utils/url"
)

// Init the utils
func Init() {
	process.Alias("xiang.helper.Captcha", "yao.utils.Captcha")                 // deprecated
	process.Alias("xiang.helper.CaptchaValidate", "yao.utils.CaptchaValidate") // deprecated

	// ****************************************
	// * Processes Version 0.10.4+
	// ****************************************
	process.Register("utils.throw.Forbidden", throw.Forbidden)
	process.Register("utils.throw.Unauthorized", throw.Unauthorized)
	process.Register("utils.throw.NotFound", throw.NotFound)
	process.Register("utils.throw.BadRequest", throw.BadRequest)
	process.Register("utils.throw.InternalError", throw.InternalError)
	process.Register("utils.throw.Exception", throw.Exception)

	// ****************************************
	// * Migrate Processes Version 0.10.2+
	// ****************************************

	// FMT
	process.Alias("xiang.helper.Print", "utils.fmt.Print")
	process.Register("utils.fmt.Printf", fmt.ProcessPrintf)
	process.Register("utils.fmt.ColorPrintf", fmt.ProcessColorPrintf)

	// ENV
	process.Alias("xiang.helper.EnvSet", "utils.env.Set")
	process.Alias("xiang.helper.EnvGet", "utils.env.Get")
	process.Alias("xiang.helper.EnvMultiSet", "utils.env.SetMany")
	process.Alias("xiang.helper.EnvMultiGet", "utils.env.GetMany")

	// Flow
	process.Alias("xiang.helper.For", "utils.flow.For")
	process.Alias("xiang.helper.Each", "utils.flow.Each")
	process.Alias("xiang.helper.Case", "utils.flow.Case")
	process.Alias("xiang.helper.IF", "utils.flow.IF")
	process.Alias("xiang.helper.Throw", "utils.flow.Throw")
	process.Alias("xiang.helper.Return", "utils.flow.Return")

	// JWT
	process.Alias("xiang.helper.JwtMake", "utils.jwt.Make")
	process.Alias("xiang.helper.JwtValidate", "utils.jwt.Verify")

	// Password
	// utils.pwd.Hash
	process.Alias("xiang.helper.PasswordValidate", "utils.pwd.Verify")

	// Captcha
	process.Alias("xiang.helper.Captcha", "utils.captcha.Make")
	process.Alias("xiang.helper.CaptchaValidate", "utils.captcha.Verify")

	// String
	process.Alias("xiang.helper.StrConcat", "utils.str.Concat")
	process.Alias("xiang.helper.HexToString", "utils.str.Hex")
	process.Register("utils.str.Join", str.ProcessJoin)
	process.Register("utils.str.JoinPath", str.ProcessJoinPath)
	process.Register("utils.str.UUID", str.ProcessUUID)

	// Array
	process.Alias("xiang.helper.ArrayPluck", "utils.arr.Pluck")
	process.Alias("xiang.helper.ArraySplit", "utils.arr.Split")
	process.Alias("xiang.helper.ArrayTree", "utils.arr.Tree")
	process.Alias("xiang.helper.ArrayUnique", "utils.arr.Unique")
	process.Alias("xiang.helper.ArrayIndexes", "utils.arr.Indexes")
	process.Alias("xiang.helper.ArrayGet", "utils.arr.Get")
	process.Alias("xiang.helper.ArrayColumn", "utils.arr.Column") // doc
	process.Alias("xiang.helper.ArrayKeep", "utils.arr.Keep")
	process.Alias("xiang.helper.ArrayMapSet", "utils.arr.MapSet")

	// Tree
	process.Register("utils.tree.Flatten", tree.ProcessFlatten)

	// Map
	process.Alias("xiang.helper.MapGet", "utils.map.Get")
	process.Alias("xiang.helper.MapSet", "utils.map.Set")
	process.Alias("xiang.helper.MapDel", "utils.map.Del")
	process.Alias("xiang.helper.MapDel", "utils.map.DelMany")
	process.Alias("xiang.helper.MapKeys", "utils.map.Keys")
	process.Alias("xiang.helper.MapValues", "utils.map.Values")
	process.Alias("xiang.helper.MapToArray", "utils.map.Array") // doc
	// utils.map.Merge

	// Time
	process.Alias("xiang.flow.Sleep", "utils.time.Sleep")
	process.Register("utils.now.Time", datetime.ProcessTime)
	process.Register("utils.now.Date", datetime.ProcessDate)
	process.Register("utils.now.DateTime", datetime.ProcessDateTime)
	process.Register("utils.now.Timestamp", datetime.ProcessTimestamp)
	process.Register("utils.now.Timestampms", datetime.ProcessTimestampms)

	// URL
	process.Register("utils.url.ParseQuery", url.ProcessParseQuery)
	process.Register("utils.url.QueryParam", url.ProcessQueryParam)
	process.Register("utils.url.ParseURL", url.ProcessParseURL)

	// JSON
	process.Register("utils.json.Validate", json.ProcessValidate)
}
