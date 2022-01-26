package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yaoapp/xiang/config"
)

var appPath string
var envFile string

var rootCmd = &cobra.Command{
	Use:   "xiang",
	Short: "象传应用引擎命令行工具",
	Long:  `象传应用引擎命令行工具`,
	Args:  cobra.MinimumNArgs(1),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "参数错误", args)
		os.Exit(1)
	},
}

// 加载命令
func init() {
	rootCmd.AddCommand(
		versionCmd,
		// domainCmd,
		migrateCmd,
		inspectCmd,
		startCmd,
		// importCmd,
		runCmd,
	)
	rootCmd.SetHelpCommand(helpCmd)
	rootCmd.PersistentFlags().StringVarP(&appPath, "app", "a", "", "指定应用目录")
	rootCmd.PersistentFlags().StringVarP(&envFile, "env", "e", "", "指定环境变量文件")
}

// Execute 运行Root
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Boot 设定配置
func Boot() {

	if envFile == "" && appPath != "" {
		config.SetAppPath(appPath, filepath.Join(appPath, ".env"))
		return
	}

	if envFile != "" { // 指定环境变量文件
		config.SetEnvFile(envFile)
	}

	if appPath != "" {
		config.SetAppPath(appPath)
	}
}
