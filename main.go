package main

import (
	"github.com/yaoapp/xiang/cmd"
	_ "github.com/yaoapp/xiang/user"
	_ "github.com/yaoapp/xiang/xfs"
)

// 主程序
func main() {
	cmd.Execute()
}
