package global

import (
	"io"
	"log"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
)

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
	Debug     bool     `json:"debug,omitempty" env:"XIANG_DB_DEBUG" envDefault:"false"`       // DEBUG 开关
	Primary   []string `json:"primary,omitempty" env:"XIANG_DB_PRIMARY" envSeparator:"|"`     // 主库连接DSN
	Secondary []string `json:"secondary,omitempty" env:"XIANG_DB_SECONDARY" envSeparator:"|"` // 从库连接DSN
	AESKey    string   `json:"aeskey,omitempty" env:"XIANG_DB_AESKEY"`                        // 加密存储KEY
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
func NewConfig() Config {
	filename := os.Getenv("XIANG_ENV_FILE")
	if filename == "" {
		filename = ".env"
	}

	err := godotenv.Load(filename)
	if err != nil {
		log.Printf("读取环境配置文件%s出错 %s\n", filename, err.Error())
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
}
