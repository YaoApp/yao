package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/cmd/get"
	"github.com/yaoapp/yao/share"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: L("Get an application"),
	Long:  L("Get an application"),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println(color.RedString(L("Not enough arguments")))
			fmt.Println(color.WhiteString(share.BUILDNAME + " help"))
			return
		}

		repo := args[0]
		pkg, err := get.New(repo)
		if err != nil {
			fmt.Println(color.RedString(err.Error()))
			os.Exit(1)
		}

		fmt.Println(color.WhiteString("From Yao: %s", pkg.Remote))
		fmt.Println(color.WhiteString("Visit: https://yaoapps.com"))
		err = pkg.Download()
		if err != nil {
			fmt.Println(color.RedString(err.Error()))
			os.Exit(1)
		}

		dest, err := os.Getwd()
		if err != nil {
			fmt.Println(color.RedString(err.Error()))
			os.Exit(1)
		}

		// dest, err = os.MkdirTemp(dest, "*-unit-test")
		// if err != nil {
		// 	fmt.Println(color.RedString(err.Error()))
		// 	os.Exit(1)
		// }
		// os.MkdirAll(dest, os.ModePerm)

		app, err := pkg.Unpack(dest)
		if err != nil {
			fmt.Println(color.RedString(err.Error()))
			os.Exit(1)
		}

		fmt.Println(color.GreenString(app.Name), color.WhiteString(app.Version))
		fmt.Println(color.GreenString(L("✨DONE✨")))

	},
}
