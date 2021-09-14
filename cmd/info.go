package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xiang/global"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "显示当前配置信息",
	Long:  `显示当前配置信息`,
	Run: func(cmd *cobra.Command, args []string) {
		res := maps.Map{
			"version": global.VERSION,
			"domain":  global.DOMAIN,
			"config":  global.Conf,
		}
		utils.Dump(res)
	},
}
