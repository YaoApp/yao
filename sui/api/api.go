package api

import "github.com/yaoapp/gou/api"

var dsl = []byte(`
{
	"name": "SUI API",
	"description": "The API for SUI",
	"version": "1.0.0",
	"guard": "bearer-jwt",
	"group": "__yao/sui/v1",
	"paths": [
		{
			"label": "Render",
			"description": "Render the frontend page",
			"path": "/render/*route",
			"method": "POST",
			"guard": "-",
			"process": "sui.Render",
			"in": [":context", "$param.route", ":payload"],
			"out": { "status": 200, "type": "text/html; charset=utf-8" }
		},
		{
			"label": "Run",
			"description": "Run the backend script, with Api prefix method",
			"path": "/run/*route",
			"guard": "-",
			"method": "POST",
			"process": "sui.Run",
			"in": [":context", "$param.route", ":payload"],
			"out": { "status": 200, "type": "application/json" }
		},
		// 
		// 
		// Remove the following code
		// Developer can create the API by using the process of sui.* directly
		// 
		// {
		// 	"path": "/:id/setting",
		// 	"method": "GET",
		// 	"process": "sui.Setting",
		// 	"in": ["$param.id"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },
		
		// {
		// 	"path": "/:id/template",
		// 	"method": "GET",
		// 	"process": "sui.Template.Get",
		// 	"in": ["$param.id"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/template/:template_id",
		// 	"method": "GET",
		// 	"process": "sui.Template.Find",
		// 	"in": ["$param.id", "$param.template_id"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },

		// {
		// 	"path": "/:id/locale/:template_id",
		// 	"method": "GET",
		// 	"process": "sui.Locale.Get",
		// 	"in": ["$param.id", "$param.template_id"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/theme/:template_id",
		// 	"method": "GET",
		// 	"process": "sui.Theme.Get",
		// 	"in": ["$param.id", "$param.template_id"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },

		// {
		// 	"path": "/:id/block/:template_id",
		// 	"method": "GET",
		// 	"process": "sui.Block.Get",
		// 	"in": ["$param.id", "$param.template_id"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/block/export/:template_id",
		// 	"method": "GET",
		// 	"process": "sui.Block.Export",
		// 	"in": ["$param.id", "$param.template_id"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/block/:template_id/:block_id",
		// 	"guard": "query-jwt",
		// 	"method": "GET",
		// 	"process": "sui.Block.Find",
		// 	"in": ["$param.id", "$param.template_id", "$param.block_id"],
		// 	"out": { "status": 200, "type": "text/javascript" }
		// },{
		// 	"path": "/:id/block/:template_id/:block_id/media",
		// 	"guard": "query-jwt",
		// 	"method": "GET",
		// 	"process": "sui.Block.Media",
		// 	"in": ["$param.id", "$param.template_id", "$param.block_id"],
		// 	"out": {
		// 		"status": 200,
		// 		"body": "?:content",
		// 		"headers": { "Content-Type": "?:type"}
		// 	}
		// },

		// {
		// 	"path": "/:id/component/:template_id",
		// 	"method": "GET",
		// 	"process": "sui.Component.Get",
		// 	"in": ["$param.id", "$param.template_id"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/component/:template_id/:component_id",
		// 	"guard": "query-jwt",
		// 	"method": "GET",
		// 	"process": "sui.Component.Find",
		// 	"in": ["$param.id", "$param.template_id", "$param.component_id"],
		// 	"out": { "status": 200, "type": "text/javascript" }
		// },
		
		// {
		// 	"path": "/:id/page/:template_id/*route",
		// 	"method": "GET",
		// 	"process": "sui.Page.Get",
		// 	"in": ["$param.id", "$param.template_id", "$param.route"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/page/tree/:template_id/*route",
		// 	"method": "GET",
		// 	"process": "sui.Page.Tree",
		// 	"in": ["$param.id", "$param.template_id", "$param.route"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/page/save/:template_id/*route",
		// 	"method": "POST",
		// 	"process": "sui.Page.Save",
		// 	"in": ["$param.id", "$param.template_id", "$param.route", ":context"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/page/temp/:template_id/*route",
		// 	"method": "POST",
		// 	"process": "sui.Page.SaveTemp",
		// 	"in": ["$param.id", "$param.template_id", "$param.route", ":context"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/page/create/:template_id/*route",
		// 	"method": "POST",
		// 	"process": "sui.Page.Create",
		// 	"in": ["$param.id", "$param.template_id", "$param.route", ":context", ":payload"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/page/duplicate/:template_id/*route",
		// 	"method": "POST",
		// 	"process": "sui.Page.Duplicate",
		// 	"in": ["$param.id", "$param.template_id", "$param.route", ":payload"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/page/rename/:template_id/*route",
		// 	"method": "POST",
		// 	"process": "sui.Page.Rename",
		// 	"in": ["$param.id", "$param.template_id", "$param.route", ":payload"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/page/exist/:template_id/*route",
		// 	"method": "GET",
		// 	"process": "sui.Page.Exist",
		// 	"in": ["$param.id", "$param.template_id", "$param.route"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/page/remove/:template_id/*route",
		// 	"method": "POST",
		// 	"process": "sui.Page.Remove",
		// 	"in": ["$param.id", "$param.template_id", "$param.route"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },
		
		// {
		// 	"path": "/:id/editor/render/:template_id/*route",
		// 	"method": "GET",
		// 	"process": "sui.Editor.Render",
		// 	"in": ["$param.id", "$param.template_id", "$param.route", ":query"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/editor/render/:template_id/*route",
		// 	"method": "POST",
		// 	"process": "sui.Editor.RenderAfterSaveTemp",
		// 	"in": ["$param.id", "$param.template_id", "$param.route", ":context", ":query"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/editor/:kind/source/:template_id/*route",
		// 	"method": "GET",
		// 	"process": "sui.Editor.Source",
		// 	"in": ["$param.id", "$param.template_id", "$param.route", "$param.kind", ":query"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/editor/:kind/source/:template_id/*route",
		// 	"method": "POST",
		// 	"process": "sui.Editor.SourceAfterSaveTemp",
		// 	"in": ["$param.id", "$param.template_id", "$param.route", ":context", "$param.kind", ":query"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },

		// {
		// 	"path": "/:id/asset/:template_id/@assets/*path",
		// 	"method": "GET",
		// 	"guard": "-",
		// 	"process": "sui.Template.Asset",
		// 	"in": ["$param.id", "$param.template_id", "$param.path", "$query.w", "$query.h"],
		// 	"out": {
		// 		"status": 200,
		// 		"body": "?:content",
		// 		"headers": { "Content-Type": "?:type"}
		// 	}
		// },{
		// 	"path": "/:id/asset/:template_id/upload",
		// 	"method": "POST",
		// 	"process": "sui.Template.AssetUpload",
		// 	"in": ["$param.id", "$param.template_id", ":context"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },{
		// 	"path": "/:id/asset/:template_id/@pages/*path",
		// 	"guard": "query-jwt",
		// 	"method": "GET",
		// 	"process": "sui.Page.Asset",
		// 	"in": ["$param.id", "$param.template_id", "$param.path", "$query.w", "$query.h"],
		// 	"out": {
		// 		"status": 200,
		// 		"body": "?:content",
		// 		"headers": { "Content-Type": "?:type"}
		// 	}
		// },

		// {
		// 	"path": "/:id/media/:driver/search",
		// 	"method": "GET",
		// 	"process": "sui.Media.Search",
		// 	"in": ["$param.id", "$param.driver", ":query"],
		// 	"out": { "status": 200, "type": "application/json" }
		// },

		// {
		// 	"path": "/:id/preview/:template_id/*route",
		// 	"guard": "query-jwt",
		// 	"method": "GET",
		// 	"process": "sui.Preview.Render",
		// 	"in": ["$param.id", "$param.template_id", "$param.route", "$header.Referer"],
		// 	"out": {"status": 200, "type": "text/html; charset=utf-8"}
		// },

		// {
		// 	"path": "/:id/build/:template_id",
		// 	"method": "POST",
		// 	"process": "sui.Build.All",
		// 	"in": ["$param.id", "$param.template_id", ":payload"],
		// 	"out": {"status": 200}
		// },{
		// 	"path": "/:id/build/:template_id/*route",
		// 	"method": "POST",
		// 	"process": "sui.Build.Page",
		// 	"in": ["$param.id", "$param.template_id", "$param.route", ":payload"],
		// 	"out": {"status": 200}
		// }
	],
}
`)

func registerAPI() error {
	_, err := api.LoadSource("<sui.v1>.yao", dsl, "sui.v1")
	return err
}
