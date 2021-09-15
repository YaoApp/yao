package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xiang/global"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动象传应用引擎",
	Long:  `启动象传应用引擎`,
	Run: func(cmd *cobra.Command, args []string) {
		defer global.ServiceStop(func() { log.Println("服务已关闭") })
		// 启动服务
		for _, api := range gou.APIs {
			for _, p := range api.HTTP.Paths {
				utils.Dump(api.Name + ":" + p.Path)
			}
		}
		global.ServiceStart()
	},
}
