package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xiang/global"
	"github.com/yaoapp/xiang/server"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动象传应用引擎",
	Long:  `启动象传应用引擎`,
	Run: func(cmd *cobra.Command, args []string) {
		// 启动服务
		for _, api := range gou.APIs {
			for _, p := range api.HTTP.Paths {
				utils.Dump(api.Name + ":" + p.Path)
			}
		}

		gou.ServeHTTP(gou.Server{
			Host:   global.Conf.Service.Host,
			Port:   global.Conf.Service.Port,
			Allows: global.Conf.Service.Allow,
			Root:   "/api",
		}, server.Middlewares...)
	},
}
