package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/share"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: L("Execute process"),
	Long:  L("Execute process"),
	Run: func(cmd *cobra.Command, args []string) {
		defer share.SessionStop()
		defer gou.KillPlugins()
		defer func() {
			err := exception.Catch(recover())
			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			}
		}()

		Boot()
		cfg := config.Conf
		cfg.Session.IsCLI = true
		engine.Load(cfg)
		if len(args) < 1 {
			fmt.Println(color.RedString(L("Not enough arguments")))
			fmt.Println(color.WhiteString(share.BUILDNAME + " help"))
			return
		}

		name := args[0]
		fmt.Println(color.GreenString(L("Run: %s"), name))
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

		process := gou.NewProcess(name, pargs...)
		res := process.Run()
		fmt.Println(color.WhiteString("--------------------------------------"))
		fmt.Println(color.WhiteString(L("%s Response"), name))
		fmt.Println(color.WhiteString("--------------------------------------"))
		utils.Dump(res)
		fmt.Println(color.WhiteString("--------------------------------------"))
		fmt.Println(color.GreenString(L("✨DONE✨")))
	},
}
