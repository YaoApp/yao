package query

import (
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query"
	dsl "github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
)

// Load 加载查询引擎
func Load(cfg config.Config) error {

	if _, has := query.Engines["default"]; !has {
		registerDefault()
	}

	// register connector
	for id, conn := range connector.Connectors {
		if _, has := query.Engines[id]; has {
			continue
		}

		if conn.Is(connector.DATABASE) {
			qb, err := conn.Query()
			if err != nil {
				log.Error("[Query] load connector error %v", err.Error())
				continue
			}
			query.Register(id, &dsl.Query{
				Query:        qb,
				GetTableName: func(s string) string { return s },
				AESKey:       config.Conf.DB.AESKey,
			})
		}
	}

	return nil
}

// Unload Query Engine
func Unload() error {
	for id := range query.Engines {
		query.Unregister(id)
	}
	return nil
}

// registerDefaultQuery register the default engine
func registerDefault() {
	if capsule.Global != nil {
		query.Register("default", &dsl.Query{
			Query: capsule.Query(),
			GetTableName: func(s string) string {
				if mod, has := model.Models[s]; has {
					return mod.MetaData.Table.Name
				}
				log.Error("%s model does not load", s)
				return s
			},
			AESKey: config.Conf.DB.AESKey,
		})
	}
}
