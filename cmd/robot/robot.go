package robot

import (
	"os"
	"path/filepath"

	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
)

var appPath string
var envFile string

var langs = map[string]string{
	"Install a robot package from the registry": "从注册中心安装 Robot 包",
	"Team ID (required)":                        "团队 ID (必填)",
	"Package version or dist-tag":               "包版本或 dist-tag",
	"Application directory":                     "应用目录",
	"Environment file":                          "环境变量文件",
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
	root := config.Conf.Root
	if appPath != "" {
		r, err := filepath.Abs(appPath)
		if err != nil {
			exception.New("Root error %s", 500, err.Error()).Throw()
		}
		root = r
	}

	if envFile != "" {
		config.Conf = config.LoadFromWithRoot(envFile, root)
	} else {
		config.Conf = config.LoadFromWithRoot(filepath.Join(root, ".env"), root)
	}

	config.ApplyMode()
}
