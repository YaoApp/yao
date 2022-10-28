package share

import (
	"fmt"
	"time"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
)

// DBConnect 建立数据库连接
func DBConnect(dbconfig config.DBConfig) (err error) {

	if dbconfig.Primary == nil {
		return fmt.Errorf("YAO_DB_PRIMARY was not set")
	}

	manager := capsule.New()
	for i, dsn := range dbconfig.Primary {
		_, err = manager.Add(fmt.Sprintf("primary-%d", i), dbconfig.Driver, dsn, false)
		if err != nil {
			return err
		}
	}

	if dbconfig.Secondary != nil {
		for i, dsn := range dbconfig.Secondary {
			_, err = manager.Add(fmt.Sprintf("secondary-%d", i), dbconfig.Driver, dsn, true)
			if err != nil {
				return err
			}
		}
	}

	manager.SetAsGlobal()
	go func() {
		for _, c := range manager.Pool.Primary {
			err = c.Ping(5 * time.Second)
			if err != nil {
				log.Error("%s error %v", c.Config.Name, err.Error())
			}
		}
	}()

	return err
}
