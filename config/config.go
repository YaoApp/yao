package config

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
)

// Conf 配置参数
var Conf Config

// Config 系统配置
type Config struct {
	XiangConfig
	Service  ServiceConfig  `json:"service,omitempty"`
	Database DatabaseConfig `json:"database,omitempty"`
	Storage  StorageConfig  `json:"storage,omitempty"`
	JWT      JWTConfig      `json:"jwt,omitempty"`
	Log      LogConfig      `json:"log,omitempty"`
}

// XiangConfig 象传应用引擎配置
type XiangConfig struct {
	Mode       string `json:"mode,omitempty" env:"XIANG_MODE" envDefault:"release"`     // 象传引擎模式 debug/release/test
	Source     string `json:"source,omitempty" env:"XIANG_SOURCE" envDefault:"fs://."`  // 源码路径(用于单元测试载入数据)
	Path       string `json:"path,omitempty" env:"XIANG_PATH" envDefault:"bin://xiang"` // 引擎文件目录
	Root       string `json:"root,omitempty" env:"XIANG_ROOT" envDefault:"fs://."`      // 应用文件目录
	RootUI     string `json:"root_ui,omitempty" env:"XIANG_ROOT_UI"`                    // 应用界面静态文件目录
	RootDB     string `json:"root_db,omitempty" env:"XIANG_ROOT_DB"`                    // 应用SQLite数据库目录
	RootData   string `json:"root_data,omitempty" env:"XIANG_ROOT_DATA"`                // 应用数据文件目录
	RootAPI    string `json:"root_api,omitempty" env:"XIANG_ROOT_API"`                  // 应用API文件目录
	RootModel  string `json:"root_model,omitempty" env:"XIANG_ROOT_MODEL"`              // 应用模型文件目录
	RootFLow   string `json:"root_flow,omitempty" env:"XIANG_ROOT_FLOW"`                // 应用工作流文件目录
	RootPlugin string `json:"root_plugin,omitempty" env:"XIANG_ROOT_PLUGIN"`            // 应用插件文件目录
	RootTable  string `json:"root_table,omitempty" env:"XIANG_ROOT_TABLE"`              // 应用表格文件目录
	RootChart  string `json:"root_chart,omitempty" env:"XIANG_ROOT_CHART"`              // 应用图表文件目录
	RootKanban string `json:"root_kanban,omitempty" env:"XIANG_ROOT_KANBAN"`            // 应用看板文件目录
	RootScreen string `json:"root_screen,omitempty" env:"XIANG_ROOT_SCREEN"`            // 应用大屏文件目录
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Debug bool     `json:"debug,omitempty" env:"XIANG_SERVICE_DEBUG" envDefault:"false"`   // DEBUG 开关
	HTTPS bool     `json:"https,omitempty" env:"XIANG_SERVICE_HTTPS" envDefault:"false"`   // HTTPS 开关
	Cert  string   `json:"cert,omitempty" env:"XIANG_SERVICE_CERT"`                        // HTTPS 证书
	Key   string   `json:"key,omitempty" env:"XIANG_SERVICE_KEY"`                          // HTTPS 证书密钥
	Allow []string `json:"allow,omitempty" env:"XIANG_SERVICE_ALLOW" envSeparator:"|"`     // 跨域访问域名列表
	Host  string   `json:"host,omitempty" env:"XIANG_SERVICE_HOST" envDefault:"127.0.0.1"` // 服务监听IP
	Port  int      `json:"port,omitempty" env:"XIANG_SERVICE_PORT" envDefault:"5099"`      // 服务监听端口
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Debug     bool     `json:"debug,omitempty" env:"XIANG_DB_DEBUG" envDefault:"false"`                                                       // DEBUG 开关
	Driver    string   `json:"driver,omitempty" env:"XIANG_DB_DRIVER" envDefault:"sqlite3"`                                                   // 数据库驱动 ( sqlite3, mysql, postgres)
	Primary   []string `json:"primary,omitempty" env:"XIANG_DB_PRIMARY" envSeparator:"|" envDefault:"file:xiang.db?cache=shared&mode=memory"` // 主库连接DSN
	Secondary []string `json:"secondary,omitempty" env:"XIANG_DB_SECONDARY" envSeparator:"|"`                                                 // 从库连接DSN
	AESKey    string   `json:"aeskey,omitempty" env:"XIANG_DB_AESKEY"`                                                                        // 加密存储KEY
}

// StorageConfig 存储配置
type StorageConfig struct {
	Debug bool   `json:"debug,omitempty" env:"XIANG_STOR_DEBUG" envDefault:"false"`          // DEBUG 开关
	Path  string `json:"path,omitempty" env:"XIANG_STOR_PATH" envDefault:"fs:///data/xiang"` // 数据存储目录
}

// JWTConfig JWT配置
type JWTConfig struct {
	Debug  bool   `json:"debug,omitempty" env:"XIANG_JWT_DEBUG" envDefault:"false"` // DEBUG 开关
	Secret string `json:"secret,omitempty" env:"XIANG_JWT_SECRET"`                  // JWT 密钥
}

// LogConfig 日志配置
type LogConfig struct {
	Access string `json:"access,omitempty" env:"XIANG_LOG_ACCESS" envDefault:"os://stdout"` // 服务访问日志
	Error  string `json:"error,omitempty" env:"XIANG_LOG_ERROR" envDefault:"os://stderr"`   // 服务错误日志
	DB     string `json:"database,omitempty" env:"XIANG_LOG_DB" envDefault:"os://stdout"`   // 数据库日志
	Plugin string `json:"plugin,omitempty" env:"XIANG_LOG_PLUGIN" envDefault:"os://stdout"` // 插件日志
}

// NewConfig 创建配置文件
func NewConfig(envfile ...string) Config {

	filename := os.Getenv("XIANG_ENV_FILE")
	if filename == "" {
		filename = ".env"
	}

	if len(envfile) > 0 {
		file, err := filepath.Abs(envfile[0])
		if err != nil {
			log.Printf("加载环境配置文件%s出错 %s\n", envfile[0], err.Error())
		} else {
			filename = file
		}
	}

	err := godotenv.Overload(filename)
	if err != nil {
		log.Printf("加载环境配置文件%s出错 %s\n", filename, err.Error())
	}

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		exception.New("解析配置文件出错 %s", 500, err.Error()).Throw()
	}

	cfg.SetDefaults()
	return cfg
}

// NewConfigFrom 创建配置文件
func NewConfigFrom(input io.Reader) Config {
	cfg := Config{}
	err := helper.UnmarshalFile(input, &cfg)
	if err != nil {
		exception.New("解析配置文件出错 %s", 500, err.Error()).Throw()
	}
	cfg.SetDefaults()
	return cfg
}

// SetDefaults 设定默认值
func (cfg *Config) SetDefaults() {
	if cfg.RootAPI == "" {
		cfg.RootAPI = cfg.Root + "/apis"
	}
	if cfg.RootFLow == "" {
		cfg.RootFLow = cfg.Root + "/flows"
	}
	if cfg.RootModel == "" {
		cfg.RootModel = cfg.Root + "/models"
	}
	if cfg.RootPlugin == "" {
		cfg.RootPlugin = cfg.Root + "/plugins"
	}

	if cfg.RootTable == "" {
		cfg.RootTable = cfg.Root + "/tables"
	}
	if cfg.RootChart == "" {
		cfg.RootChart = cfg.Root + "/charts"
	}
	if cfg.RootScreen == "" {
		cfg.RootScreen = cfg.Root + "/screens"
	}

	if cfg.RootData == "" {
		cfg.RootData = cfg.Root + "/data"
	}

	if cfg.RootDB == "" {
		cfg.RootDB = cfg.Root + "/db"
		cfg.RootDB = strings.TrimPrefix(cfg.RootDB, "fs://")
		cfg.RootDB = strings.TrimPrefix(cfg.RootDB, "file://")
	}
	if cfg.RootUI == "" {
		cfg.RootUI = cfg.Root + "/ui"
		cfg.RootUI = strings.TrimPrefix(cfg.RootUI, "fs://")
		cfg.RootUI = strings.TrimPrefix(cfg.RootUI, "file://")

	}

}

// SetEnvFile 指定ENV文件
func SetEnvFile(filename string) {
	Conf = NewConfig(filename)
}

// SetAppPath 设定应用目录
func SetAppPath(root string, envfile ...string) {

	fullpath, err := filepath.Abs(root)
	if err != nil {
		log.Panicf("设定应用目录%s出错 %s\n", root, err.Error())
	}

	// 创建应用目录
	pathinfo, err := os.Stat(fullpath)
	if os.IsNotExist(err) {
		err := os.MkdirAll(fullpath, os.ModePerm)
		if err != nil {
			log.Panicf("创建目录失败(%s) %s", root, err)
		}
	}
	pathinfo, err = os.Stat(fullpath)
	if !pathinfo.IsDir() {
		log.Panicf("检查应用目录失败(%s) ", err)
	}

	if !pathinfo.IsDir() {
		log.Panicf("应用目录不是文件夹(%s) ", root)
	}

	// Set ENV
	if len(envfile) > 0 {
		Conf = NewConfig(envfile[0])
	}

	// 从加载配置文件
	Conf.Root = fullpath
	Conf.RootAPI = filepath.Join(fullpath, "/apis")
	Conf.RootFLow = filepath.Join(fullpath, "/flows")
	Conf.RootModel = filepath.Join(fullpath, "/models")
	Conf.RootPlugin = filepath.Join(fullpath, "/plugins")
	Conf.RootTable = filepath.Join(fullpath, "/tables")
	Conf.RootChart = filepath.Join(fullpath, "/charts")
	Conf.RootScreen = filepath.Join(root, "/screens")
	Conf.RootData = filepath.Join(fullpath, "/data")
	Conf.RootUI = filepath.Join(fullpath, "/ui")
	Conf.RootDB = filepath.Join(fullpath, "/db")
}

// IsDebug 是否为调试模式
func IsDebug() bool {
	return Conf.Mode == "debug"
}

func init() {
	Conf = NewConfig()
}
