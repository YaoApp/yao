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
		// Do Stuff Here
		fmt.Println(share.VERSION)
	},
}
