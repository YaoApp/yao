package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/blang/semver"
	"github.com/fatih/color"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/share"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: L("Upgrade yao app to latest version"),
	Long:  L("Upgrade yao app to latest version"),
	Run: func(cmd *cobra.Command, args []string) {
		Boot()
		latest, found, err := selfupdate.DetectLatest("yaoapp/yao")
		if err != nil {
			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
				os.Exit(1)
			}
		}
		currentVersion := semver.MustParse(share.VERSION)
		if !found || latest.Version.LTE(currentVersion) {
			fmt.Println(color.GreenString(L("🎉Current version is the latest🎉")))
			os.Exit(0)
		}
		fmt.Println(color.WhiteString(L("Do you want to update to %s ? (y/n): "), latest.Version))
		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}
		if input != "y\n" && input != "Y\n" && input != "n\n" && input != "N\n" {
			fmt.Println(color.RedString(L("Fatal: %s"), L("Invalid input")))
			os.Exit(1)
		}
		if input == "n\n" || input == "N\n" {
			fmt.Println(color.YellowString(L("Canceled upgrade")))
			return
		}
		exe, err := os.Executable()
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}
		if err := selfupdate.UpdateTo(latest.AssetURL, exe); err != nil {
			fmt.Println(color.RedString(L("Error occurred while updating binary: %s"), err.Error()))
			os.Exit(1)
		}
		fmt.Println(color.GreenString(L("🎉Successfully updated to version: %s🎉"), latest.Version))
	},
}
