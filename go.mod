module github.com/yaoapp/xiang

go 1.16

require (
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.1286
	github.com/aliyun/aliyun-oss-go-sdk v2.1.10+incompatible
	github.com/armon/go-metrics v0.3.10 // indirect
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/buraksezer/consistent v0.9.0 // indirect
	github.com/buraksezer/olric v0.4.2
	github.com/caarlos0/env/v6 v6.7.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/elazarl/go-bindata-assetfs v1.0.1
	github.com/fatih/color v1.13.0
	github.com/fsnotify/fsnotify v1.5.1
	github.com/gin-gonic/gin v1.7.7
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/btree v1.0.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/memberlist v0.3.0 // indirect
	github.com/hashicorp/yamux v0.0.0-20211028200310-0bc27b27de87 // indirect
	github.com/jmoiron/sqlx v1.3.4 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/json-iterator/go v1.1.12
	github.com/lib/pq v1.10.4 // indirect
	github.com/mattn/go-sqlite3 v1.14.9 // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/mojocn/base64Captcha v1.3.5
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/spf13/afero v1.6.0
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	github.com/xuri/excelize/v2 v2.5.0
	github.com/yaoapp/gou v0.0.0-20211120135538-e5387704eb03
	github.com/yaoapp/kun v0.9.0
	github.com/yaoapp/xun v0.9.0
	golang.org/x/crypto v0.0.0-20220208050332-20e1d8d225ab
	golang.org/x/image v0.0.0-20210628002857-a66eb6448b8d // indirect
	golang.org/x/net v0.0.0-20211118161319-6a13c67c3ce4 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/genproto v0.0.0-20211118181313-81c1377c94b1 // indirect
	google.golang.org/grpc v1.42.0 // indirect
)

// go env -w GOPRIVATE=github.com/yaoapp/*

replace github.com/yaoapp/kun => ../kun // kun local

replace github.com/yaoapp/xun => ../xun // gou local

replace github.com/yaoapp/gou => ../gou // gou local
