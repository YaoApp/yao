package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/share"
)

var printAllVersion bool
var versionTemplate = `Version:          %s
Go version:       %s
Yao commit:       %s
Cui version:      %s
Cui commit:       %s
Built:            %s
OS/Arch:          %s/%s
`
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: L("Show version"),
	Long:  L("Show version"),
	Run: func(cmd *cobra.Command, args []string) {
		if printAllVersion {
			commit := strings.Split(share.PRVERSION, "-")[0]
			buildTime := strings.TrimPrefix(share.PRVERSION, commit+"-")
			cuiCommit := strings.Split(share.PRCUI, "-")[0]

			fmt.Printf("%s", color.WhiteString("Yao version:    "))
			fmt.Printf("%s\n", color.GreenString(share.VERSION))

			fmt.Printf("%s", color.WhiteString("Yao commit:     "))
			fmt.Printf("%s\n", color.YellowString(commit))

			fmt.Printf("%s", color.WhiteString("Cui commit:     "))
			fmt.Printf("%s\n", color.YellowString(cuiCommit))

			fmt.Printf("%s", color.WhiteString("Built:          "))
			fmt.Printf("%s\n", color.BlueString(buildTime))

			fmt.Printf("%s", color.WhiteString("OS/Arch:        "))
			fmt.Printf("%s\n", color.MagentaString("%s/%s", runtime.GOOS, runtime.GOARCH))

			fmt.Printf("%s", color.WhiteString("Go version:     "))
			fmt.Printf("%s\n", color.CyanString(runtime.Version()))
			return
		}
		fmt.Printf("%s", color.WhiteString("Yao version: "))
		fmt.Printf("%s\n", color.GreenString(share.VERSION))
	},
}

func init() {
	versionCmd.PersistentFlags().BoolVarP(&printAllVersion, "all", "", false, L("Print all version information"))
}
