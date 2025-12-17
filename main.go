package main

import (
	_ "github.com/yaoapp/gou/diff"
	_ "github.com/yaoapp/gou/encoding"
	_ "github.com/yaoapp/gou/text"
	_ "github.com/yaoapp/yao/aigc"
	_ "github.com/yaoapp/yao/crypto"
	_ "github.com/yaoapp/yao/excel"
	_ "github.com/yaoapp/yao/helper"
	_ "github.com/yaoapp/yao/openai"
	_ "github.com/yaoapp/yao/seed"
	_ "github.com/yaoapp/yao/trace/jsapi"
	_ "github.com/yaoapp/yao/wework"

	"github.com/yaoapp/yao/cmd"
	"github.com/yaoapp/yao/utils"
	//
	// _ "net/http/pprof"
)

func main() {
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()
	utils.Init()
	cmd.Execute()
}
