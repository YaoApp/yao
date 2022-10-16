package config

// Config 象传应用引擎配置
type Config struct {
	Mode      string        `json:"mode,omitempty" env:"YAO_ENV" envDefault:"production"`      // 象传引擎启动模式 production/development
	Root      string        `json:"root,omitempty" env:"YAO_ROOT" envDefault:"."`              // 应用根目录
	Lang      string        `json:"lang,omitempty" env:"YAO_LANG" envDefault:"en-us"`          // Default language setting
	TimeZone  string        `json:"timezone,omitempty" env:"YAO_TIMEZONE"`                     // Default TimeZone
	DataRoot  string        `json:"data_root,omitempty" env:"YAO_DATA_ROOT" envDefault:""`     // DATA PATH
	Host      string        `json:"host,omitempty" env:"YAO_HOST" envDefault:"0.0.0.0"`        // 服务监听地址
	Port      int           `json:"port,omitempty" env:"YAO_PORT" envDefault:"5099"`           // 服务监听端口
	Cert      string        `json:"cert,omitempty" env:"YAO_CERT"`                             // HTTPS 证书文件地址
	Key       string        `json:"key,omitempty" env:"YAO_KEY"`                               // HTTPS 证书密钥地址
	Log       string        `json:"log,omitempty" env:"YAO_LOG"`                               // 服务日志地址
	LogMode   string        `json:"log_mode,omitempty" env:"YAO_LOG_MODE" envDefault:"TEXT"`   // 服务日志模式 JSON|TEXT
	JWTSecret string        `json:"jwt_secret,omitempty" env:"YAO_JWT_SECRET"`                 // JWT 密钥
	DB        DBConfig      `json:"db,omitempty"`                                              // 数据库配置
	AllowFrom []string      `json:"allowfrom,omitempty" envSeparator:"|" env:"YAO_ALLOW_FROM"` // Domain list the separator is |
	Session   SessionConfig `json:"session,omitempty"`                                         // Session Config
	Studio    StudioConfig  `json:"studio,omitempty"`                                          // Studio config
}

// StudioConfig the studio config
type StudioConfig struct {
	Port   int    `json:"studio_port,omitempty" env:"YAO_STUDIO_PORT" envDefault:"5077"` // Studio port
	Secret []byte `json:"studio_secret,omitempty" env:"YAO_STUDIO_SECRET"`               // Studio Secret, if does not set, auto-generate a secret
	Auto   bool   `json:"-"`
}

// DBConfig 数据库配置
type DBConfig struct {
	Driver    string   `json:"driver,omitempty" env:"YAO_DB_DRIVER" envDefault:"sqlite3"`                        // 数据库驱动 sqlite3| mysql| postgres
	Primary   []string `json:"primary,omitempty" env:"YAO_DB_PRIMARY" envSeparator:"|" envDefault:"./db/yao.db"` // 主库连接DSN
	Secondary []string `json:"secondary,omitempty" env:"YAO_DB_SECONDARY" envSeparator:"|"`                      // 从库连接DSN
	AESKey    string   `json:"aeskey,omitempty" env:"YAO_DB_AESKEY"`                                             // 加密存储KEY
}

// SessionConfig 会话服务器
type SessionConfig struct {
	Store    string `json:"store,omitempty" env:"YAO_SESSION_STORE" envDefault:"file"`    // The session store. redis | file
	File     string `json:"file,omitempty" env:"YAO_SESSION_FILE"`                        // The file path
	Host     string `json:"host,omitempty" env:"YAO_SESSION_HOST" envDefault:"127.0.0.1"` // The redis host
	Port     string `json:"port,omitempty" env:"YAO_SESSION_PORT" envDefault:"6379"`      // The redis port
	Password string `json:"password,omitempty" env:"YAO_SESSION_PASSWORD"`                // The redis password
	Username string `json:"username,omitempty" env:"YAO_SESSION_USERNAME"`                // The redis username
	DB       string `json:"db,omitempty" env:"YAO_SESSION_DB" envDefault:"1"`             // The redis username
	IsCLI    bool   `json:"iscli,omitempty" env:"YAO_SESSION_ISCLI" envDefault:"false"`   // Command Line Start
}
