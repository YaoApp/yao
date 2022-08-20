package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

var appPath string
var envFile string

var lang = os.Getenv("YAO_LANG")
var langs = map[string]string{
	"Start Engine":                          "启动象传应用引擎",
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
}

// L 多语言切换
func L(words string) string {
	if lang == "" {
		return words
	}

	if trans, has := langs[words]; has {
		return trans
	}
	return words
}

var rootCmd = &cobra.Command{
	Use:   share.BUILDNAME,
	Short: "Yao App Engine",
	Long:  `Yao App Engine`,
	Args:  cobra.MinimumNArgs(1),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			switch args[0] {
			case "fuxi":
				fuxi()
			}
		}
		fmt.Fprintln(os.Stderr, L("One or more arguments are not correct"), args)
		os.Exit(1)
	},
}

// 加载命令
func init() {
	rootCmd.AddCommand(
		versionCmd,
		migrateCmd,
		inspectCmd,
		startCmd,
		runCmd,
		initCmd,
		serviceCmd,
		dumpCmd,
		restoreCmd,
		socketCmd,
		websocketCmd,
	)
	// rootCmd.SetHelpCommand(helpCmd)
	rootCmd.PersistentFlags().StringVarP(&appPath, "app", "a", "", L("Application directory"))
	rootCmd.PersistentFlags().StringVarP(&envFile, "env", "e", "", L("Environment file"))
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
	root := config.Conf.Root
	if appPath != "" {
		r, err := filepath.Abs(appPath)
		if err != nil {
			exception.New("Root error %s", 500, err.Error()).Throw()
		}
		root = r
	}
	if envFile != "" {
		config.Conf = config.LoadFrom(envFile)
	} else {
		config.Conf = config.LoadFrom(filepath.Join(root, ".env"))
	}

	if config.Conf.Mode == "production" {
		config.Production()
	} else if config.Conf.Mode == "development" {
		config.Development()
	}
}
