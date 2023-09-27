package api

import "github.com/yaoapp/gou/api"

var dsl = []byte(`
{
	"name": "SUI API",
	"description": "The API for SUI",
	"version": "1.0.0",
	"guard": "-",
	"group": "__yao/sui/v1",
	"paths": [
		{
			"path": "/:id/template/find/:template_id",
			"method": "GET",
			"process": "sui.Template.Find",
			"in": ["$param.id", "$param.template_id"],
			"out": { "status": 200, "type": "application/json" }
		},{
			"path": "/:id/template/get",
			"method": "GET",
			"process": "sui.Template.Get",
			"in": ["$param.id"],
			"out": { "status": 200, "type": "application/json" }
		}
	],
}
`)

func registerAPI() error {
	_, err := api.LoadSource("<sui.v1>.yao", dsl, "sui.v1")
	return err
}
