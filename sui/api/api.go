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
			"path": "/:id/template",
			"method": "GET",
			"process": "sui.Template.Get",
			"in": ["$param.id"],
			"out": { "status": 200, "type": "application/json" }
		},{
			"path": "/:id/template/:template_id",
			"method": "GET",
			"process": "sui.Template.Find",
			"in": ["$param.id", "$param.template_id"],
			"out": { "status": 200, "type": "application/json" }
		},

		{
			"path": "/:id/locale/:template_id",
			"method": "GET",
			"process": "sui.Locale.Get",
			"in": ["$param.id", "$param.template_id"],
			"out": { "status": 200, "type": "application/json" }
		},{
			"path": "/:id/theme/:template_id",
			"method": "GET",
			"process": "sui.Theme.Get",
			"in": ["$param.id", "$param.template_id"],
			"out": { "status": 200, "type": "application/json" }
		},

		{
			"path": "/:id/block/:template_id",
			"method": "GET",
			"process": "sui.Block.Get",
			"in": ["$param.id", "$param.template_id"],
			"out": { "status": 200, "type": "application/json" }
		},{
			"path": "/:id/block/:template_id/:block_id",
			"method": "GET",
			"process": "sui.Block.Find",
			"in": ["$param.id", "$param.template_id", "$param.block_id"],
			"out": { "status": 200, "type": "text/javascript" }
		},

		{
			"path": "/:id/component/:template_id",
			"method": "GET",
			"process": "sui.Component.Get",
			"in": ["$param.id", "$param.template_id"],
			"out": { "status": 200, "type": "application/json" }
		},{
			"path": "/:id/component/:template_id/:component_id",
			"method": "GET",
			"process": "sui.Component.Find",
			"in": ["$param.id", "$param.template_id", "$param.component_id"],
			"out": { "status": 200, "type": "text/javascript" }
		},
		
		{
			"path": "/:id/page/:template_id/*route",
			"method": "GET",
			"process": "sui.Page.Get",
			"in": ["$param.id", "$param.template_id", "$param.route"],
			"out": { "status": 200, "type": "application/json" }
		},{
			"path": "/:id/page/tree/:template_id/*route",
			"method": "GET",
			"process": "sui.Page.Tree",
			"in": ["$param.id", "$param.template_id", "$param.route"],
			"out": { "status": 200, "type": "application/json" }
		},
		
		{
			"path": "/:id/editor/render/:template_id/*route",
			"method": "GET",
			"process": "sui.Editor.Render",
			"in": ["$param.id", "$param.template_id", "$param.route"],
			"out": { "status": 200, "type": "application/json" }
		},{
			"path": "/:id/editor/:kind/source/:template_id/*route",
			"method": "GET",
			"process": "sui.Editor.Source",
			"in": ["$param.id", "$param.template_id", "$param.route", "$param.kind"],
			"out": { "status": 200, "type": "application/json" }
		}
	],
}
`)

func registerAPI() error {
	_, err := api.LoadSource("<sui.v1>.yao", dsl, "sui.v1")
	return err
}
