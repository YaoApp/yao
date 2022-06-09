package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/share"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: L("Show version"),
	Long:  L("Show version"),
	Run: func(cmd *cobra.Command, args []string) {

		version := share.VERSION
		if share.PRVERSION != "" {
			version = fmt.Sprintf("%s-%s", share.VERSION, share.PRVERSION)
		}

		// Do Stuff Here
		fmt.Println(version)
	},
}
