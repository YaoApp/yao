package main

import (
	_ "github.com/yaoapp/gou/diff"
	_ "github.com/yaoapp/gou/encoding"
	_ "github.com/yaoapp/gou/text"
	_ "github.com/yaoapp/yao/crypto"
	_ "github.com/yaoapp/yao/excel"
	_ "github.com/yaoapp/yao/helper"
	_ "github.com/yaoapp/yao/job/jsapi"
	_ "github.com/yaoapp/yao/openai"
	_ "github.com/yaoapp/yao/rss"
	_ "github.com/yaoapp/yao/seed"
	_ "github.com/yaoapp/yao/sitemap"
	_ "github.com/yaoapp/yao/tools"

	"os"

	"github.com/yaoapp/yao/cmd"
	"github.com/yaoapp/yao/utils"
	//
	// _ "net/http/pprof"
)

func main() {
	// Internal: Windows self-update helper spawned by "yao upgrade".
	if len(os.Args) >= 3 && os.Args[1] == "--finish-upgrade" {
		cmd.FinishUpgrade(os.Args[2])
		return
	}

	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()
	utils.Init()
	cmd.Execute()
}
