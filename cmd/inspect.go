package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: L("Show app configure"),
	Long:  L("Show app configure"),
	Run: func(cmd *cobra.Command, args []string) {
		Boot()
		res := maps.Map{
			"version": share.VERSION,
			"config":  config.Conf,
		}
		utils.Dump(res)
	},
}
