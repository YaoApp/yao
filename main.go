package main

import (
	"github.com/yaoapp/yao/cmd"

	_ "github.com/yaoapp/gou/encoding"
	_ "github.com/yaoapp/yao/crypto"
	_ "github.com/yaoapp/yao/helper"
	_ "github.com/yaoapp/yao/network"
	_ "github.com/yaoapp/yao/user"
	_ "github.com/yaoapp/yao/xfs"
)

// 主程序
func main() {
	cmd.Execute()
}
