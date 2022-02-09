package cmd

import (
	"github.com/spf13/cobra"
)

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: L("Help for yao"),
	Long:  L("Help for yao"),
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
