package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
)

var name string
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: L("Update database schema"),
	Long:  L("Update database schema"),
	Run: func(cmd *cobra.Command, args []string) {
		Boot()
		// 加载数据模型
		err := engine.Load(config.Conf)
		if err != nil {
			fmt.Printf(color.RedString("加载失败: %s\n", err.Error()))
			os.Exit(1)
		}

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
