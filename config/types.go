package config

// Config 象传应用引擎配置
type Config struct {
	Mode          string   `json:"mode,omitempty" env:"YAO_ENV" envDefault:"production"`            // The start mode production/development
	AppSource     string   `json:"app,omitempty"  env:"YAO_APP_SOURCE"`                             // The Application Source Root Path default same as Root
	Root          string   `json:"root,omitempty" env:"YAO_ROOT" envDefault:"."`                    // The Application Root Path
	Lang          string   `json:"lang,omitempty" env:"YAO_LANG" envDefault:"en-us"`                // Default language setting
	TimeZone      string   `json:"timezone,omitempty" env:"YAO_TIMEZONE"`                           // Default TimeZone
	DataRoot      string   `json:"data_root,omitempty" env:"YAO_DATA_ROOT" envDefault:""`           // The data root path
	ExtensionRoot string   `json:"extension_root,omitempty" env:"YAO_EXTENSION_ROOT" envDefault:""` // Plugin, Wasm root PATH, Default is <YAO_ROOT> (<YAO_ROOT>/plugins <YAO_ROOT>/wasms)
	Host          string   `json:"host,omitempty" env:"YAO_HOST" envDefault:"0.0.0.0"`              // The server host
	Port          int      `json:"port,omitempty" env:"YAO_PORT" envDefault:"5099"`                 // The server port
	Cert          string   `json:"cert,omitempty" env:"YAO_CERT"`                                   // The HTTPS certificate path
	Key           string   `json:"key,omitempty" env:"YAO_KEY"`                                     // The HTTPS certificate key path
	Log           string   `json:"log,omitempty" env:"YAO_LOG"`                                     // The log file path
	LogMode       string   `json:"log_mode,omitempty" env:"YAO_LOG_MODE" envDefault:"TEXT"`         // The log mode TEXT|JSON
	LogMaxSize    int      `json:"log_max_size,omitempty" env:"YAO_LOG_MAX_SIZE" envDefault:"100"`  // The max log size in MB, the default is 100
	LogMaxAage    int      `json:"log_max_age,omitempty" env:"YAO_LOG_MAX_AGE" envDefault:"7"`      // The max log age in day, the default is 7
	LogMaxBackups int      `json:"log_max_backups" env:"YAO_LOG_MAX_BACKUPS" envDefault:"3"`        // The max log backups, the default is 3
	LogLocalTime  bool     `json:"log_local_time" env:"YAO_LOG_LOCAL_TIME" envDefault:"true"`
	JWTSecret     string   `json:"jwt_secret,omitempty" env:"YAO_JWT_SECRET"`                 // The JWT Secret
	DB            Database `json:"db,omitempty"`                                              // The database config
	AllowFrom     []string `json:"allowfrom,omitempty" envSeparator:"|" env:"YAO_ALLOW_FROM"` // Domain list the separator is |
	Session       Session  `json:"session,omitempty"`                                         // Session Config
	Studio        Studio   `json:"studio,omitempty"`                                          // Studio config
	Runtime       Runtime  `json:"runtime,omitempty"`                                         // Runtime config
}

// Studio the studio config
type Studio struct {
	Port   int    `json:"studio_port,omitempty" env:"YAO_STUDIO_PORT" envDefault:"5077"` // Studio port
	Secret string `json:"studio_secret,omitempty" env:"YAO_STUDIO_SECRET"`               // Studio Secret, if does not set, auto-generate a secret
	Auto   bool   `json:"-"`
}

// Database 数据库配置
type Database struct {
	Driver    string   `json:"driver,omitempty" env:"YAO_DB_DRIVER" envDefault:"sqlite3"`                        // 数据库驱动 sqlite3| mysql| postgres
	Primary   []string `json:"primary,omitempty" env:"YAO_DB_PRIMARY" envSeparator:"|" envDefault:"./db/yao.db"` // 主库连接DSN
	Secondary []string `json:"secondary,omitempty" env:"YAO_DB_SECONDARY" envSeparator:"|"`                      // 从库连接DSN
	AESKey    string   `json:"aeskey,omitempty" env:"YAO_DB_AESKEY"`                                             // 加密存储KEY
}

// Session 会话服务器
type Session struct {
	Store    string `json:"store,omitempty" env:"YAO_SESSION_STORE" envDefault:"file"`    // The session store. redis | file
	File     string `json:"file,omitempty" env:"YAO_SESSION_FILE"`                        // The file path
	Host     string `json:"host,omitempty" env:"YAO_SESSION_HOST" envDefault:"127.0.0.1"` // The redis host
	Port     string `json:"port,omitempty" env:"YAO_SESSION_PORT" envDefault:"6379"`      // The redis port
	Password string `json:"password,omitempty" env:"YAO_SESSION_PASSWORD"`                // The redis password
	Username string `json:"username,omitempty" env:"YAO_SESSION_USERNAME"`                // The redis username
	DB       string `json:"db,omitempty" env:"YAO_SESSION_DB" envDefault:"1"`             // The redis username
	IsCLI    bool   `json:"iscli,omitempty" env:"YAO_SESSION_ISCLI" envDefault:"false"`   // Command Line Start
}

// Runtime Config
type Runtime struct {
	Mode              string `json:"mode,omitempty"  env:"YAO_RUNTIME_MODE" envDefault:"standard"`                        // the mode of the runtime, the default value is "standard" and the other value is "performance". "performance" mode need more memory but will run faster
	MinSize           uint   `json:"minSize,omitempty" env:"YAO_RUNTIME_MIN" envDefault:"10"`                             // the number of V8 VM when runtime start. max value is 100, the default value is 2
	MaxSize           uint   `json:"maxSize,omitempty" env:"YAO_RUNTIME_MAX" envDefault:"100"`                            // the maximum of V8 VM should be smaller than minSize, the default value is 10
	DefaultTimeout    int    `json:"defaultTimeout,omitempty" env:"YAO_RUNTIME_TIMEOUT" envDefault:"200"`                 // the default timeout for the script, the default value is 200ms
	ContextTimeout    int    `json:"contextTimeout,omitempty" env:"YAO_RUNTIME_CONTEXT_TIMEOUT" envDefault:"200"`         // the default timeout for the context, the default value is 200ms
	HeapSizeLimit     uint64 `json:"heapSizeLimit,omitempty" env:"YAO_RUNTIME_HEAP_LIMIT" envDefault:"1518338048"`        // the isolate heap size limit should be smaller than 1.5G, and the default value is 1518338048 (1.5G)
	HeapSizeRelease   uint64 `json:"heapSizeRelease,omitempty" env:"YAO_RUNTIME_HEAP_RELEASE" envDefault:"52428800"`      // the isolate will be re-created when reaching this value, and the default value is 52428800 (50M)
	HeapAvailableSize uint64 `json:"heapAvailableSize,omitempty" env:"YAO_RUNTIME_HEAP_AVAILABLE" envDefault:"524288000"` // the isolate will be re-created when the available size is smaller than this value, and the default value is 524288000 (500M)
	Precompile        bool   `json:"precompile,omitempty" env:"YAO_RUNTIME_PRECOMPILE" envDefault:"false"`                // if true compile scripts when the VM is created. this will increase the load time, but the script will run faster. the default value is false
	Import            bool   `json:"import,omitempty"  env:"YAO_RUNTIME_IMPORT" envDefault:"true"`                        // If false the import statement will be disabled, the default value is true.
}
