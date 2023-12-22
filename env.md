# Environment Variables

## Config

 - `YAO_ENV` (default: `production`) - The start mode production/development
 - `YAO_APP_SOURCE` - The Application Source Root Path default same as Root
 - `YAO_ROOT` (default: `.`) - The Application Root Path
 - `YAO_LANG` (default: `en-us`) - Default language setting
 - `YAO_TIMEZONE` - Default TimeZone
 - `YAO_DATA_ROOT` - The data root path
 - `YAO_EXTENSION_ROOT` - Plugin, Wasm root PATH, Default is <YAO_ROOT> (<YAO_ROOT>/plugins <YAO_ROOT>/wasms)
 - `YAO_HOST` (default: `0.0.0.0`) - The server host
 - `YAO_PORT` (default: `5099`) - The server port
 - `YAO_CERT` - The HTTPS certificate path
 - `YAO_KEY` - The HTTPS certificate key path
 - `YAO_LOG` - The log file path
 - `YAO_LOG_MODE` (default: `TEXT`) - The log mode TEXT|JSON
 - `YAO_JWT_SECRET` - The JWT Secret
 - `YAO_ALLOW_FROM` (separated by `|`) - Domain list the separator is |

## Studio

 - `YAO_STUDIO_PORT` (default: `5077`) - Studio port
 - `YAO_STUDIO_SECRET` - Studio Secret, if does not set, auto-generate a secret

## Database

 - `YAO_DB_DRIVER` (default: `sqlite3`) - 数据库驱动 sqlite3| mysql| postgres
 - `YAO_DB_PRIMARY` (separated by `|`, default: `./db/yao.db`) - 主库连接DSN
 - `YAO_DB_SECONDARY` (separated by `|`) - 从库连接DSN
 - `YAO_DB_AESKEY` - 加密存储KEY

## Session

 - `YAO_SESSION_STORE` (default: `file`) - The session store. redis | file
 - `YAO_SESSION_FILE` - The file path
 - `YAO_SESSION_HOST` (default: `127.0.0.1`) - The redis host
 - `YAO_SESSION_PORT` (default: `6379`) - The redis port
 - `YAO_SESSION_PASSWORD` - The redis password
 - `YAO_SESSION_USERNAME` - The redis username
 - `YAO_SESSION_DB` (default: `1`) - The redis username
 - `YAO_SESSION_ISCLI` (default: `false`) - Command Line Start

## Runtime

 - `YAO_RUNTIME_MODE` (default: `normal`) - the mode of the runtime, the default value is "normal" and the other value is "performance". "performance" mode need more memory but will run faster
 - `YAO_RUNTIME_MIN` (default: `10`) - the number of V8 VM when runtime start. max value is 100, the default value is 2
 - `YAO_RUNTIME_MAX` (default: `100`) - the maximum of V8 VM should be smaller than minSize, the default value is 10
 - `YAO_RUNTIME_TIMEOUT` (default: `200`) - the default timeout for the script, the default value is 200ms
 - `YAO_RUNTIME_CONTEXT_TIMEOUT` (default: `200`) - the default timeout for the context, the default value is 200ms
 - `YAO_RUNTIME_HEAP_LIMIT` (default: `1518338048`) - the isolate heap size limit should be smaller than 1.5G, and the default value is 1518338048 (1.5G)
 - `YAO_RUNTIME_HEAP_RELEASE` (default: `52428800`) - the isolate will be re-created when reaching this value, and the default value is 52428800 (50M)
 - `YAO_RUNTIME_HEAP_AVAILABLE` (default: `524288000`) - the isolate will be re-created when the available size is smaller than this value, and the default value is 524288000 (500M)
 - `YAO_RUNTIME_PRECOMPILE` (default: `false`) - if true compile scripts when the VM is created. this will increase the load time, but the script will run faster. the default value is false
