package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/utils"
)

var name string
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "更新数据结构",
	Long:  `更新数据库结构`,
	Run: func(cmd *cobra.Command, args []string) {

		if name != "" {
			mod, has := gou.Models[name]
			if has {
				mod.Migrate(true)
			}
			return
		}

		// Do Stuff Here
		for name, mod := range gou.Models {
			utils.Dump(name)
			mod.Migrate(true)
		}
	},
}

func init() {
	migrateCmd.PersistentFlags().StringVarP(&name, "name", "n", "", "指定模型名称")
}
