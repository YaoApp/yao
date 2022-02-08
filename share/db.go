package share

import (
	"time"

	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xun/capsule"
)

// DBConnect 建立数据库连接
func DBConnect(dbconfig config.DBConfig) {

	// 连接主库
	for i, dsn := range dbconfig.Primary {
		db := capsule.AddConn("primary", dbconfig.Driver, dsn, 5*time.Second)
		if i == 0 {
			db.SetAsGlobal()
		}
	}

	// 连接从库
	for _, dsn := range dbconfig.Secondary {
		capsule.AddReadConn("secondary", dbconfig.Driver, dsn, 5*time.Second)
	}
}
