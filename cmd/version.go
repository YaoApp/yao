package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaoapp/xiang/share"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示当前版本号",
	Long:  `显示当前版本号`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		fmt.Println(share.VERSION)
	},
}
