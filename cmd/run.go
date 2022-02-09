package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: L("Execute process"),
	Long:  L("Execute process"),
	Run: func(cmd *cobra.Command, args []string) {
		defer gou.KillPlugins()
		Boot()
		cfg := config.Conf
		// cfg.Session.IsCLI = true
		engine.Load(cfg)
		if len(args) < 1 {
			fmt.Println(color.RedString("参数错误: 未指定处理名称"))
			fmt.Println(color.WhiteString("xiang run <处理器名称> [参数表...]"))
			return
		}

		name := args[0]
		fmt.Println(color.GreenString("运行处理器: %s", name))
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
					fmt.Println(color.RedString("参数错误: %s", err.Error()))
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
		fmt.Println(color.WhiteString("\n--------------------------------------"))
		fmt.Println(color.WhiteString("%s 返回结果", name))
		fmt.Println(color.WhiteString("--------------------------------------"))
		utils.Dump(res)
	},
}
