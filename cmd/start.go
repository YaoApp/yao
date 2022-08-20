package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/websocket"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/service"
	"github.com/yaoapp/yao/share"
)

var startDebug = false
var startAlpha = false

var startCmd = &cobra.Command{
	Use:   "start",
	Short: L("Start Engine"),
	Long:  L("Start Engine"),
	Run: func(cmd *cobra.Command, args []string) {

		// recive interrupt signal
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

		// defer service.Stop(func() { fmt.Println(L("Service stopped")) })
		Boot()

		if startDebug { // 强制 debug 模式启动
			config.Development()
		}

		mode := config.Conf.Mode
		err := engine.Load(config.Conf) // 加载脚本等
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}
		port := fmt.Sprintf(":%d", config.Conf.Port)
		if port == ":80" {
			port = ""
		}

		host := config.Conf.Host
		if host == "0.0.0.0" {
			host = "127.0.0.1"
		}

		if mode == "development" || mode == "production" {
			fmt.Println(color.WhiteString("\n---------------------------------"))
			fmt.Println(color.WhiteString(L("API List")))
			fmt.Println(color.WhiteString("---------------------------------"))

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

			fmt.Printf(color.CyanString("\n%s(%d)\n", "WebSocket", len(websocket.Upgraders)))
			for name, upgrader := range websocket.Upgraders { // WebSocket
				fmt.Println(
					colorMehtod("GET"),
					color.WhiteString(filepath.Join("/websocket", name)),
					"\tprocess:", upgrader.Process)
			}
		}

		fmt.Println(color.WhiteString("\n---------------------------------"))
		fmt.Println(color.WhiteString(share.App.Name), color.WhiteString(share.App.Version), mode)
		fmt.Println(color.WhiteString("---------------------------------"))
		if !share.BUILDIN {
			root, _ := filepath.Abs(config.Conf.Root)
			fmt.Println(color.WhiteString(L("Root")), color.GreenString(" %s", root))
		}

		fmt.Println(color.WhiteString(L("Frontend")), color.GreenString(" http://%s%s/", host, port))
		fmt.Println(color.WhiteString(L("Dashboard")), color.GreenString(" http://%s%s/xiang/login/admin", host, port))
		fmt.Println(color.WhiteString(L("API")), color.GreenString(" http://%s%s/api", host, port))
		fmt.Println(color.WhiteString(L("Listening")), color.GreenString(" %s:%d", config.Conf.Host, config.Conf.Port))

		fmt.Println("")

		// 调试模式
		if mode == "development" {
			service.Watch(config.Conf)
		}
		go service.Start()
		fmt.Println(color.GreenString(L("✨LISTENING✨")))

		for {
			select {
			case <-interrupt:
				service.Stop(func() {
					fmt.Println(color.GreenString(L("✨STOPPED✨")))
				})
				return
			}
		}
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

func init() {
	startCmd.PersistentFlags().BoolVarP(&startDebug, "debug", "", false, L("Development mode"))
	startCmd.PersistentFlags().BoolVarP(&startAlpha, "alpha", "", false, L("Enabled unstable features"))
}
