package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/cmd/agent"
	"github.com/yaoapp/yao/cmd/mcp"
	"github.com/yaoapp/yao/cmd/robot"
	"github.com/yaoapp/yao/cmd/sui"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/pack"
	"github.com/yaoapp/yao/share"
)

var appPath string
var yazFile string
var licenseKey string

var lang = os.Getenv("YAO_LANG")
var langs = map[string]string{
	"Start Engine":                          "启动 YAO 应用引擎",
	"Get an application":                    "下载应用源码",
	"One or more arguments are not correct": "参数错误",
	"Application directory":                 "指定应用路径",
	"Environment file":                      "指定环境变量文件",
	"Help for yao":                          "显示命令帮助文档",
	"Show app configure":                    "显示应用配置信息",
	"Update database schema":                "更新数据表结构",
	"Execute process":                       "运行处理器",
	"Show version":                          "显示当前版本号",
	"Development mode":                      "使用开发模式启动",
	"Enabled unstable features":             "启用内测功能",
	"Fatal: %s":                             "失败: %s",
	"Service stopped":                       "服务已关闭",
	"API":                                   " API接口",
	"API List":                              "API列表",
	"Root":                                  "应用目录",
	"Data":                                  "数据目录",
	"Frontend":                              "前台地址",
	"Dashboard":                             "管理后台",
	"Not enough arguments":                  "参数错误: 缺少参数",
	"Run: %s":                               "运行: %s",
	"Arguments: %s":                         "参数错误: %s",
	"%s Response":                           "%s 返回结果",
	"Update schema model: %s (%s) ":         "更新表结构 model: %s (%s)",
	"Model name":                            "模型名称",
	"Initialize project":                    "项目初始化",
	"✨DONE✨":                                "✨完成✨",
	"NEXT:":                                 "下一步:",
	"Listening":                             "    监听",
	"✨LISTENING✨":                           "✨服务正在运行✨",
	"✨STOPPED✨":                             "✨服务已停止✨",
	"SessionPort":                           "会话服务端口",
	"Force migrate":                         "强制更新数据表结构",
	"Migrate is not allowed on production mode.": "Migrate 不能再生产环境下使用",
	"Upgrade yao to latest version":              "升级 yao 到最新版本",
	"Current version:":                           "当前版本:",
	"Latest version: ":                           "最新版本: ",
	"Checking latest version...":                 "正在检查最新版本...",
	"🎉Current version is the latest🎉":            "🎉当前版本是最新的🎉",
	"Do you want to update to %s ? (y/n): ":      "是否更新到 %s ? (y/n): ",
	"Invalid input":                              "输入错误",
	"Canceled upgrade":                           "已取消更新",
	"Downloading...":                             "正在下载...",
	"Progress:":                                  "进度:",
	"Available assets:":                          "可用的制品:",
	"Error occurred while updating binary: %s":   "更新二进制文件时出错: %s",
	"🎉Successfully updated to version: %s🎉":      "🎉成功更新到版本: %s🎉",
	"Print all version information":              "显示详细版本信息",
	"SUI Template Engine":                        "SUI 模板引擎命令",
	"MCP commands":                               "MCP 包管理命令",
	"MCP package management commands":            "MCP 包管理命令",
	"Robot commands":                             "Robot 包管理命令",
	"Robot package management commands":          "Robot 包管理命令",
}

// L Language switch
func L(words string) string {
	if lang == "" {
		return words
	}

	if trans, has := langs[words]; has {
		return trans
	}
	return words
}

// RootCmd export the rootCmd to support customized commands when use yao as lib
var RootCmd = rootCmd

var rootCmd = &cobra.Command{
	Use:   share.BUILDNAME,
	Short: "Yao App Engine",
	Long:  `Yao App Engine`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			switch args[0] {
			case "fuxi":
				fuxi()
				return
			}
			fmt.Fprintln(os.Stderr, L("One or more arguments are not correct"), args)
			os.Exit(1)
			return
		}
		// No arguments - show help
		cmd.Help()
	},
}

var suiCmd = &cobra.Command{
	Use:   "sui",
	Short: L("SUI Template Engine"),
	Long:  L("SUI Template Engine"),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: L("Agent commands"),
	Long:  L("Agent commands for testing and management"),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: L("MCP commands"),
	Long:  L("MCP package management commands"),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var robotCmd = &cobra.Command{
	Use:   "robot",
	Short: L("Robot commands"),
	Long:  L("Robot package management commands"),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Command initialize
func init() {

	// Sui
	suiCmd.AddCommand(sui.WatchCmd)
	suiCmd.AddCommand(sui.BuildCmd)
	suiCmd.AddCommand(sui.TransCmd)

	// Agent
	agentCmd.AddCommand(agent.TestCmd)
	agentCmd.AddCommand(agent.ExtractCmd)
	agentCmd.AddCommand(agent.AddCmd)
	agentCmd.AddCommand(agent.UpdateCmd)
	agentCmd.AddCommand(agent.PushCmd)
	agentCmd.AddCommand(agent.ForkCmd)

	// MCP
	mcpCmd.AddCommand(mcp.AddCmd)
	mcpCmd.AddCommand(mcp.UpdateCmd)
	mcpCmd.AddCommand(mcp.PushCmd)
	mcpCmd.AddCommand(mcp.ForkCmd)

	// Robot
	robotCmd.AddCommand(robot.AddCmd)

	rootCmd.AddCommand(
		versionCmd,
		migrateCmd,
		inspectCmd,
		startCmd,
		runCmd,
		loginCmd,
		logoutCmd,
		// getCmd,
		// dumpCmd,
		// restoreCmd,
		// socketCmd,
		// websocketCmd,
		// packCmd,
		suiCmd,
		agentCmd,
		mcpCmd,
		robotCmd,
		upgradeCmd,
	)
	// rootCmd.SetHelpCommand(helpCmd)
	rootCmd.PersistentFlags().StringVarP(&appPath, "app", "a", "", L("Application directory"))
	rootCmd.PersistentFlags().StringVarP(&yazFile, "file", "f", "", L("Application package file"))
	rootCmd.PersistentFlags().StringVarP(&licenseKey, "key", "k", "", L("Application license key"))
}

// Execute Command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Boot Setting
func Boot() {

	root := config.Conf.Root
	if appPath != "" {
		r, err := filepath.Abs(appPath)
		if err != nil {
			exception.New("Root error %s", 500, err.Error()).Throw()
		}
		root = r
	}

	config.Conf = config.LoadFrom(filepath.Join(root, ".env"))

	if share.BUILDIN {
		os.Setenv("YAO_APP_SOURCE", "::binary")
		config.Conf.AppSource = "::binary"
	}

	if yazFile != "" {
		os.Setenv("YAO_APP_SOURCE", yazFile)
		config.Conf.AppSource = yazFile
	}

	if config.Conf.Mode == "production" {
		config.Production()
	} else if config.Conf.Mode == "development" {
		config.Development()
	}

	// set license
	if licenseKey != "" {
		pack.SetCipher(licenseKey)
	}
}
