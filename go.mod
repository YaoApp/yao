module github.com/yaoapp/xiang

go 1.16

require (
	github.com/caarlos0/env/v6 v6.7.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/elazarl/go-bindata-assetfs v1.0.1 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/gin-gonic/gin v1.7.4 // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/hashicorp/go-plugin v1.4.3 // indirect
	github.com/hashicorp/yamux v0.0.0-20210826001029-26ff87cf9493 // indirect
	github.com/jmoiron/sqlx v1.3.4 // indirect
	github.com/joho/godotenv v1.3.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/lib/pq v1.10.3 // indirect
	github.com/mattn/go-colorable v0.1.10 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-sqlite3 v1.14.8 // indirect
	github.com/mojocn/base64Captcha v1.3.5 // indirect
	github.com/robertkrimen/otto v0.0.0-20210927222213-f9375a256948 // indirect
	github.com/spf13/cobra v1.2.1 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/yaoapp/gou v0.0.0-20210929143406-112eed6cdfd9 // indirect
	github.com/yaoapp/kun v0.6.5
	github.com/yaoapp/xun v0.5.2 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/image v0.0.0-20210628002857-a66eb6448b8d // indirect
	golang.org/x/net v0.0.0-20210928044308-7d9f5e0b762b // indirect
	golang.org/x/sys v0.0.0-20210927094055-39ccf1dd6fa6 // indirect
	google.golang.org/genproto v0.0.0-20210928142010-c7af6a1a74c9 // indirect
	google.golang.org/grpc v1.41.0 // indirect
)

// go env -w GOPRIVATE=github.com/yaoapp/*

// replace github.com/yaoapp/kun => ../kun // kun local
// replace github.com/yaoapp/xun => ../xun // gou local
// replace github.com/yaoapp/gou => ../gou // gou local
