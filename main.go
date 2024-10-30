package main

import (
	"github.com/yaoapp/yao/cmd"
	"github.com/yaoapp/yao/utils"

	_ "github.com/yaoapp/gou/encoding"
	_ "github.com/yaoapp/yao/aigc"
	_ "github.com/yaoapp/yao/crypto"
	_ "github.com/yaoapp/yao/helper"
	_ "github.com/yaoapp/yao/openai"
	_ "github.com/yaoapp/yao/wework"
	// _ "net/http/pprof"
)

func main() {
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()
	utils.Init()
	cmd.Execute()
}
