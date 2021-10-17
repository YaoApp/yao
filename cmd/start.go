package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/global"
	"github.com/yaoapp/xiang/share"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动象传应用引擎",
	Long:  `启动象传应用引擎`,
	Run: func(cmd *cobra.Command, args []string) {
		defer global.ServiceStop(func() { fmt.Println("服务已关闭") })

		Boot()
		mode := config.Conf.Mode
		if mode == "debug" {
			mode = color.RedString("调试模式")
		} else {
			mode = ""
		}

		fmt.Printf(color.GreenString("\n象传应用引擎 v%s %s", share.VERSION, mode))

		// 加载数据模型 API 等
		global.Load(config.Conf)

		// 打印应用目录信息
		fmt.Printf(color.WhiteString("\n---------------------------------"))
		fmt.Printf(color.GreenString("\n应用名称: %s v%s", share.App.Name, share.App.Version))
		fmt.Printf(color.GreenString("\n应用根目录: %s", config.Conf.Root))
		fmt.Printf(color.GreenString("\n数据存储目录: %s", config.Conf.RootData))
		fmt.Printf(color.GreenString("\n数据存储引擎: %s", share.App.Storage.Default))
		fmt.Printf(color.WhiteString("\n---------------------------------\n\n"))

		fmt.Printf(color.GreenString("\n已注册API"))
		fmt.Printf(color.WhiteString("\n---------------------------------"))

		for _, api := range gou.APIs { // API信息
			if len(api.HTTP.Paths) <= 0 {
				continue
			}

			fmt.Printf(color.CyanString("\n%s(%d)\n", api.Name, len(api.HTTP.Paths)))
			for _, p := range api.HTTP.Paths {
				fmt.Println(
					colorMehtod(p.Method),
					color.WhiteString(filepath.Join("/api", api.HTTP.Group, p.Path)),
					"\tprocess:", p.Process)
			}

		}

		domain := share.DOMAIN
		if domain == "*.iqka.com" {
			domain = "local.iqka.com"
		}

		port := fmt.Sprintf(":%d", config.Conf.Service.Port)
		if port == ":80" {
			port = ""
		}

		fmt.Printf(color.GreenString("\n\n访问入口"))
		fmt.Printf(color.WhiteString("\n---------------------------------"))
		fmt.Printf(color.GreenString("\n前台界面: http://%s%s/\n", domain, port))
		fmt.Printf(color.GreenString("管理后台: http://%s%s/xiang/login\n", domain, port))
		fmt.Printf(color.GreenString("API 接口: http://%s%s/api\n", domain, port))
		fmt.Printf(color.GreenString("跨域访问: %s\n\n", strings.Join(config.Conf.Service.Allow, ",")))

		// 调试模式
		if config.Conf.Mode == "debug" {
			global.WatchChanges()
		}

		global.ServiceStart()
	},
}

func colorMehtod(method string) string {
	method = strings.ToUpper(method)
	switch method {
	case "GET":
		return color.GreenString("GET")
	case "POST":
		return color.YellowString("POST")
	default:
		return color.WhiteString(method)
	}
}
