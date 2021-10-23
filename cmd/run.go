package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/engine"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "运行处理器",
	Long:  `运行处理器`,
	Run: func(cmd *cobra.Command, args []string) {
		defer gou.KillPlugins()
		Boot()
		engine.Load(config.Conf)
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
			pargs = append(pargs, arg)
			fmt.Println(color.WhiteString("args[%d]: %s", i-1, arg))
		}

		process := gou.NewProcess(name, pargs...)
		res := process.Run()
		fmt.Println(color.WhiteString("\n--------------------------------------"))
		fmt.Println(color.WhiteString("%s 返回结果", name))
		fmt.Println(color.WhiteString("--------------------------------------"))
		utils.Dump(res)
	},
}
