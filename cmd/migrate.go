package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/engine"
)

var name string
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "更新数据结构",
	Long:  `更新数据库结构`,
	Run: func(cmd *cobra.Command, args []string) {
		Boot()
		// 加载数据模型
		engine.Load(config.Conf)

		if name != "" {
			mod, has := gou.Models[name]
			if has {
				mod.Migrate(true)
			}
			return
		}

		// Do Stuff Here
		for _, mod := range gou.Models {
			fmt.Println(color.GreenString("更新数据模型 %s 绑定数据表: %s", mod.Name, mod.MetaData.Table.Name))
			mod.Migrate(true)
		}
	},
}

func init() {
	migrateCmd.PersistentFlags().StringVarP(&name, "name", "n", "", "指定模型名称")
}
