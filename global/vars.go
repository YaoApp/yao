package global

import (
	"net/http"
	"strings"

	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/data"
)

// VERSION 版本号
const VERSION = "0.8.0"

// DOMAIN 许可域
const DOMAIN = "*.iqka.com"

// AllowHosts 解析后的许可域
var AllowHosts = []string{}

// Conf 配置文件
var Conf config.Config

// FileServer 静态服务
var FileServer http.Handler = http.FileServer(data.AssetFS())

// 初始化配置
func init() {

	// 解析许可Host
	domains := strings.Split(DOMAIN, "|")
	for _, domain := range domains {

		if !strings.Contains(domain, ".") {
			continue
		}

		if strings.HasPrefix(domain, "*.") {
			domain = strings.TrimPrefix(domain, "*.")
		}
		AllowHosts = append(AllowHosts, domain)
	}
}
