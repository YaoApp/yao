package query

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/query"
	dsl "github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
)

// Load 加载查询引擎
func Load(cfg config.Config) {
	DefaultQuery()
}

// DefaultQuery register the default engine
func DefaultQuery() {
	query.Register("default", &dsl.Query{
		Query: capsule.Query(),
		GetTableName: func(s string) string {
			if mod, has := gou.Models[s]; has {
				return mod.MetaData.Table.Name
			}
			log.Error("%s model does not load", s)
			return s
		},
		AESKey: config.Conf.DB.AESKey,
	})
	query.Alias("default", "xiang")
}
