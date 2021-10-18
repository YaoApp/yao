module github.com/yaoapp/xiang

go 1.16

require (
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.1286
	github.com/aliyun/aliyun-oss-go-sdk v2.1.10+incompatible
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/caarlos0/env/v6 v6.7.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/elazarl/go-bindata-assetfs v1.0.1
	github.com/fatih/color v1.13.0
	github.com/fsnotify/fsnotify v1.5.1
	github.com/gin-gonic/gin v1.7.4
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/hashicorp/go-hclog v1.0.0 // indirect
	github.com/hashicorp/go-plugin v1.4.3 // indirect
	github.com/hashicorp/yamux v0.0.0-20210826001029-26ff87cf9493 // indirect
	github.com/jmoiron/sqlx v1.3.4 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/json-iterator/go v1.1.12
	github.com/lib/pq v1.10.3 // indirect
	github.com/mattn/go-sqlite3 v1.14.8 // indirect
	github.com/mojocn/base64Captcha v1.3.5
	github.com/robertkrimen/otto v0.0.0-20211004134430-12a632260352 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/spf13/afero v1.6.0
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	github.com/yaoapp/gou v0.0.0-20211007044811-e0834e575aa0
	github.com/yaoapp/kun v0.6.7
	github.com/yaoapp/xun v0.6.0
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/image v0.0.0-20210628002857-a66eb6448b8d // indirect
	golang.org/x/net v0.0.0-20211006190231-62292e806868 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/genproto v0.0.0-20211005153810-c76a74d43a8e // indirect
	google.golang.org/grpc v1.41.0 // indirect
)

// go env -w GOPRIVATE=github.com/yaoapp/*

replace github.com/yaoapp/kun => ../kun // kun local

replace github.com/yaoapp/xun => ../xun // gou local

replace github.com/yaoapp/gou => ../gou // gou local
