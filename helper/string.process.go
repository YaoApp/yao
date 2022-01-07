package helper

import (
	"fmt"

	"github.com/yaoapp/gou"
)

// ProcessStrConcat  xiang.helper.StrConcat 连接字符串
func ProcessStrConcat(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	res := ""
	for i := range process.Args {
		res = fmt.Sprintf("%v%v", res, process.Args[i])
	}
	return res
}
