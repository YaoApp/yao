package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/share"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "显示当前配置信息",
	Long:  `显示当前配置信息`,
	Run: func(cmd *cobra.Command, args []string) {
		Boot()
		res := maps.Map{
			"version": share.VERSION,
			"config":  config.Conf,
		}
		utils.Dump(res)
	},
}
