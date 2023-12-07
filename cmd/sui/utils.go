package sui

import (
	"os"
	"path/filepath"

	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
)

var appPath string
var envFile string

var langs = map[string]string{
	"Auto-build when the template file changes": "模板文件变化时自动构建",
	"Session Data": "会话数据",
}

// L 多语言切换
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
