package agent

import (
	"os"
	"path/filepath"

	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
)

var appPath string
var envFile string

var langs = map[string]string{
	"Test an agent with input cases":                                   "使用测试用例测试智能体",
	"Test an agent with input cases from JSONL file or direct message": "使用 JSONL 文件或直接消息测试智能体",
	"Application directory":                                            "应用目录",
	"Environment file":                                                 "环境变量文件",
	"Input: JSONL file path or message (required)":                     "输入: JSONL 文件路径或消息 (必需)",
	"Path to output file (default: output-{timestamp}.jsonl)":          "输出文件路径 (默认: output-{timestamp}.jsonl)",
	"Explicit agent ID (default: auto-detect)":                         "指定智能体 ID (默认: 自动检测)",
	"Override connector":                                               "覆盖连接器",
	"Test user ID (default: test-user)":                                "测试用户 ID (默认: test-user)",
	"Test team ID (default: test-team)":                                "测试团队 ID (默认: test-team)",
	"Path to context JSON file for custom authorization":               "自定义认证信息的 JSON 文件路径",
	"Reporter agent ID for custom report":                              "自定义报告生成器智能体 ID",
	"Number of runs for stability analysis":                            "稳定性分析的运行次数",
	"Regex pattern to filter which tests to run":                       "用于过滤测试的正则表达式",
	"Default timeout per test case":                                    "每个测试用例的默认超时时间",
	"Number of parallel test cases":                                    "并行测试用例数",
	"Verbose output":                                                   "详细输出",
	"Stop on first failure":                                            "遇到第一个失败时停止",
	"Error: input is required (-i flag)":                               "错误: 需要输入 (-i 参数)",
	"Error: failed to get current directory":                           "错误: 获取当前目录失败",
	"Error: agent (-n) is required when using direct message input and not in an agent directory": "错误: 使用直接消息输入且不在智能体目录时需要指定 -n 参数",
	"Hint: Make sure you're in a Yao application directory or specify --app flag":                 "提示: 确保在 Yao 应用目录中或使用 --app 参数指定",
	"Error: invalid timeout format": "错误: 无效的超时格式",
}

// L Language switch
func L(words string) string {
	var lang = os.Getenv("YAO_LANG")
	if lang == "" {
		return words
	}

	if trans, has := langs[words]; has {
		return trans
	}
	return words
}

// Boot sets the configuration
func Boot() {
	// Use root from Init() unless appPath is explicitly specified
	root := config.Conf.Root
	if appPath != "" {
		r, err := filepath.Abs(appPath)
		if err != nil {
			exception.New("Root error %s", 500, err.Error()).Throw()
		}
		root = r
	}

	// Load .env file, preserving the correct root
	if envFile != "" {
		config.Conf = config.LoadFromWithRoot(envFile, root)
	} else {
		config.Conf = config.LoadFromWithRoot(filepath.Join(root, ".env"), root)
	}

	config.ApplyMode()
}
