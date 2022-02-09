package query

import (
	"github.com/yaoapp/gou"
	dsl "github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
)

// Load 加载查询引擎
func Load(cfg config.Config) {
	XiangQuery()
}

// XiangQuery 注册应用引擎象 xiang
func XiangQuery() {
	gou.RegisterEngine("xiang", &dsl.Query{
		Query: capsule.Query(),
		GetTableName: func(s string) string {
			if mod, has := gou.Models[s]; has {
				return mod.MetaData.Table.Name
			}
			// exception.New("%s 数据模型尚未加载", 404).Throw()
			log.Error("%s model does not load", s)
			return s
		},
		AESKey: config.Conf.DB.AESKey,
	})
}
