package cmd

import (
	"log"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/global"
)

var startAppPath string
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动象传应用引擎",
	Long:  `启动象传应用引擎`,
	Run: func(cmd *cobra.Command, args []string) {
		defer global.ServiceStop(func() { log.Println("服务已关闭") })
		log.Printf("启动象传应用引擎 v%s mode=%s", global.VERSION, global.Conf.Mode)

		// 应用目录
		if startAppPath != "" {
			global.Conf.Root = startAppPath
			global.Conf.RootAPI = filepath.Join(startAppPath, "/apis")
			global.Conf.RootFLow = filepath.Join(startAppPath, "/flows")
			global.Conf.RootModel = filepath.Join(startAppPath, "/models")
			global.Conf.RootPlugin = filepath.Join(startAppPath, "/plugins")
			global.Conf.RootTable = filepath.Join(startAppPath, "/tables")
			global.Conf.RootChart = filepath.Join(startAppPath, "/charts")
			global.Conf.RootScreen = filepath.Join(startAppPath, "/screens")
			global.Conf.RootData = filepath.Join(startAppPath, "/data")

			// 重新加载应用
			global.Reload(global.Conf)
		}

		// 启动服务
		for _, api := range gou.APIs {
			log.Printf("%s(%d)", api.Name, len(api.HTTP.Paths))
			for _, p := range api.HTTP.Paths {
				log.Println(p.Method, filepath.Join("/api", api.HTTP.Group, p.Path), "\tprocess:", p.Process)
			}
		}

		// 调试模式
		if global.Conf.Mode == "debug" {
			global.WatchChanges()
		}

		global.ServiceStart()
	},
}

func init() {
	startCmd.PersistentFlags().StringVarP(&startAppPath, "app", "a", "", "指定应用目录")
}
