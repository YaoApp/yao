package global

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xun/capsule"
)

// VERSION 版本号
const VERSION = "0.6.3"

// DOMAIN 许可域
const DOMAIN = "*.iqka.com"

// AllowHosts 解析后的许可域
var AllowHosts = []string{}

// Conf 配置文件
var Conf config.Config

// FileServer 静态服务
var FileServer http.Handler = http.FileServer(assetFS())

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

	Conf = config.Conf

	// 数据库连接
	if len(Conf.Database.Primary) > 0 {
		capsule.AddConn("primary", "mysql", Conf.Database.Primary[0]).SetAsGlobal()
	}

	// 加密密钥
	gou.LoadCrypt(fmt.Sprintf(`{"key":"%s"}`, Conf.Database.AESKey), "AES")
	gou.LoadCrypt(`{}`, "PASSWORD")

	// 加载数据
	Load(Conf)
}

func assetFS() *assetfs.AssetFS {
	assetInfo := func(path string) (os.FileInfo, error) {
		return os.Stat(path)
	}
	for k := range _bintree.Children {
		k = "ui"
		return &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: assetInfo, Prefix: k, Fallback: "index.html"}
	}
	panic("unreachable")
}
