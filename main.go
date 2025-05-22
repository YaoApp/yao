package main

import (
	"fmt"

	_ "github.com/yaoapp/gou/diff"
	_ "github.com/yaoapp/gou/encoding"
	_ "github.com/yaoapp/yao/aigc"
	"github.com/yaoapp/yao/cmd"
	_ "github.com/yaoapp/yao/crypto"
	_ "github.com/yaoapp/yao/excel"
	_ "github.com/yaoapp/yao/helper"
	_ "github.com/yaoapp/yao/openai"
	"github.com/yaoapp/yao/utils"
	_ "github.com/yaoapp/yao/wework"
	// _ "net/http/pprof"
)

func main() {
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()
	utils.Init()
	cmd.Execute()
	fmt.Println("Hello, World!")
}
