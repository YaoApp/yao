package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaoapp/xiang/share"
)

var domainCmd = &cobra.Command{
	Use:   "domain",
	Short: "显示绑定域名",
	Long:  `显示绑定域名`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		fmt.Println(share.DOMAIN)
	},
}
