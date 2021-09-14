package cmd

import (
	"github.com/spf13/cobra"
)

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "显示命令帮助文档",
	Long:  `显示命令帮助文档`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
