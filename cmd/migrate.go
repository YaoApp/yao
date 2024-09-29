package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/share"
)

var name string
var force bool = false
var resetModel bool = false
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
		err := engine.Load(config.Conf, engine.LoadOption{Action: "migrate"})
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		if name != "" {
			mod, has := model.Models[name]
			if !has {
				fmt.Println(color.RedString(L("Model: %s does not exits"), name))
				return
			}

			fmt.Printf(color.WhiteString(L("Update schema model: %s (%s) "), mod.Name, mod.MetaData.Table.Name) + "\t")
			if resetModel {
				err := mod.DropTable()
				if err != nil {
					fmt.Printf(color.RedString(L("FAILURE\n%s"), err.Error()) + "\n")
					return
				}
			}

			err := mod.Migrate(false)
			if err != nil {
				fmt.Printf(color.RedString(L("FAILURE\n%s"), err.Error()) + "\n")
				return
			}

			fmt.Printf(color.GreenString(L("SUCCESS")) + "\n")
			return
		}

		// Do Stuff Here
		for _, mod := range model.Models {
			fmt.Printf(color.WhiteString(L("Update schema model: %s (%s) "), mod.Name, mod.MetaData.Table.Name) + "\t")

			if resetModel {
				err := mod.DropTable()
				if err != nil {
					fmt.Printf(color.RedString(L("FAILURE\n%s"), err.Error()) + "\n")
					continue
				}
			}

			err := mod.Migrate(false)
			if err != nil {
				fmt.Printf(color.RedString(L("FAILURE\n%s"), err.Error()) + "\n")
				continue
			}
			fmt.Printf(color.GreenString(L("SUCCESS")) + "\n")
		}

		// After Migrate Hook
		if share.App.AfterMigrate != "" {
			option := map[string]any{"force": force, "reset": resetModel, "mode": config.Conf.Mode}
			p, err := process.Of(share.App.AfterMigrate, option)
			if err != nil {
				fmt.Println(color.RedString(L("AfterMigrate: %s %v"), share.App.AfterMigrate, err))
				return
			}

			_, err = p.Exec()
			if err != nil {
				fmt.Println(color.RedString(L("AfterMigrate: %s %v"), share.App.AfterMigrate, err))
			}
		}

		// fmt.Println(color.GreenString(L("✨DONE✨")))
	},
}

func init() {
	migrateCmd.PersistentFlags().StringVarP(&name, "name", "n", "", L("Model name"))
	migrateCmd.PersistentFlags().BoolVarP(&force, "force", "", false, L("Force migrate"))
	migrateCmd.PersistentFlags().BoolVarP(&resetModel, "reset", "", false, L("Drop the table if exist"))
}
