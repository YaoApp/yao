package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/share"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: L("Show app configure"),
	Long:  L("Show app configure"),
	Run: func(cmd *cobra.Command, args []string) {
		Boot()
		engine.InspectExtTools()
		res := maps.Map{
			"version": share.VERSION,
			"config":  config.Conf,
		}
		if share.Tools != nil {
			res["tools"] = share.Tools
		}
		utils.Dump(res)
	},
}
