package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/plugin"
	"github.com/yaoapp/gou/socket"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/share"
)

var socketCmd = &cobra.Command{
	Use:   "socket",
	Short: L("Open a socket connection"),
	Long:  L("Open a socket connection"),
	Run: func(cmd *cobra.Command, args []string) {
		defer share.SessionStop()
		defer plugin.KillAll()
		defer func() {
			err := exception.Catch(recover())
			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			}
		}()

		Boot()
		cfg := config.Conf
		cfg.Session.IsCLI = true
		engine.Load(cfg, engine.LoadOption{Action: "socket"})
		if len(args) < 1 {
			fmt.Println(color.RedString(L("Not enough arguments")))
			fmt.Println(color.WhiteString(share.BUILDNAME + " help"))
			return
		}

		name := args[0]
		pargs := []interface{}{}
		for i, arg := range args {
			if i == 0 {
				continue
			}

			// 解析参数
			if strings.HasPrefix(arg, "::") {
				arg := strings.TrimPrefix(arg, "::")
				var v interface{}
				err := jsoniter.Unmarshal([]byte(arg), &v)
				if err != nil {
					fmt.Println(color.RedString(L("Arguments: %s"), err.Error()))
					return
				}
				pargs = append(pargs, v)
				fmt.Println(color.WhiteString("args[%d]: %s", i-1, arg))
			} else if strings.HasPrefix(arg, "\\::") {
				arg := "::" + strings.TrimPrefix(arg, "\\::")
				pargs = append(pargs, arg)
				fmt.Println(color.WhiteString("args[%d]: %s", i-1, arg))
			} else {
				pargs = append(pargs, arg)
				fmt.Println(color.WhiteString("args[%d]: %s", i-1, arg))
			}

		}

		socket, has := socket.Sockets[name]
		if !has {
			fmt.Println(color.RedString(L("%s not exists!"), name))
			return
		}

		if socket.Mode != "client" {
			fmt.Println(color.RedString(L("%s not supported yet!"), socket.Mode))
			return
		}

		host := socket.Host
		port := socket.Port
		argsLen := len(pargs)
		if argsLen > 0 {
			if inputHost, ok := pargs[0].(string); ok {
				host = inputHost
			}
		}

		if argsLen > 1 {
			if inputPort, ok := pargs[1].(string); ok {
				port = inputPort
			}
		}

		fmt.Println(color.WhiteString("\n---------------------------------"))
		fmt.Println(color.WhiteString(socket.Name))
		fmt.Println(color.WhiteString("---------------------------------"))
		fmt.Println(color.GreenString("Mode: %s", socket.Mode))
		fmt.Println(color.GreenString("Host: %s://%s", socket.Protocol, host))
		fmt.Println(color.GreenString("Port: %s", port))
		fmt.Println(color.WhiteString("--------------------------------------"))
		err := socket.Open(pargs...)
		if err != nil {
			fmt.Println(color.RedString(L("%s"), err.Error()))
			return
		}

	},
}
