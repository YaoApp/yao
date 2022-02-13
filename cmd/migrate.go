package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/share"
)

var name string
var force bool = false
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: L("Update database schema"),
	Long:  L("Update database schema"),
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			err := exception.Catch(recover())
			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			}
		}()

		Boot()

		if !force && config.Conf.Mode == "production" {
			fmt.Println(color.WhiteString(L("TRY:")), color.GreenString("%s migrate --force", share.BUILDNAME))
			exception.New(L("Migrate is not allowed on production mode."), 403).Throw()
		}

		// 加载数据模型
		err := engine.Load(config.Conf)
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
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
			fmt.Println(color.GreenString(L("Update schema model: %s (%s) "), mod.Name, mod.MetaData.Table.Name))
			mod.Migrate(true)
		}

		fmt.Println(color.GreenString(L("✨DONE✨")))
	},
}

func init() {
	migrateCmd.PersistentFlags().StringVarP(&name, "name", "n", "", L("Model name"))
	migrateCmd.PersistentFlags().BoolVarP(&force, "force", "", false, L("Force migrate"))
}
