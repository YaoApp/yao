package global

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/xun/capsule"
)

// VERSION 版本号
const VERSION = "0.6.0"

// DOMAIN 许可域
const DOMAIN = "*.iqka.com"

// Conf 配置文件
var Conf Config

// 初始化配置
func init() {
	Conf = NewConfig()

	// 数据库连接
	capsule.AddConn("primary", "mysql", Conf.Database.Primary[0]).SetAsGlobal()

	// 加密密钥
	gou.LoadCrypt(fmt.Sprintf(`{"key":"%s"}`, Conf.Database.AESKey), "AES")
	gou.LoadCrypt(`{}`, "PASSWORD")

	// 加载数据
	Load(Conf)
}
