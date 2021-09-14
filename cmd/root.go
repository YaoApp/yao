package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "xiang",
	Short: "象传应用引擎命令行工具",
	Long:  `象传应用引擎命令行工具`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "参数错误", args)
		os.Exit(1)
	},
}

// 加载命令
func init() {
	rootCmd.AddCommand(
		versionCmd,
		domainCmd,
		migrateCmd,
		infoCmd,
		startCmd,
	)
}

// Execute 运行Root
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
