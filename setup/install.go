package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/fs/dsl"
	"github.com/yaoapp/gou/fs/system"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/app"
)

// Install app install
//
//	{
//	  "env": {
//	    "YAO_LANG": "中文",
//	    "YAO_ENV": "开发模式(推荐)",
//	    "YAO_PORT": "5099",
//	    "YAO_STUDIO_PORT": "5077"
//	  },
//	  "db": {
//	    "type": "sqlite",
//	    "option.file": "db/yao.db"
//	  }
//	}
//
//	{
//		"env": {
//		  "YAO_LANG": "中文",
//		  "YAO_ENV": "开发模式(推荐)",
//		  "YAO_PORT": "5099",
//		  "YAO_STUDIO_PORT": "5077"
//		},
//		"db": {
//		  "type": "mysql",
//		  "option.db": "yao",
//		  "option.host.host": "127.0.0.1",
//		  "option.host.port": "3306",
//		  "option.host.user": "root",
//		  "option.host.pass": "123456"
//		}
//	}
func Install(payload map[string]map[string]string) error {

	dbOption, err := getDBOption(payload)
	if err != nil {
		return err
	}

	err = ValidateDB(dbOption)
	if err != nil {
		return err
	}

	envOption, err := getENVOption(payload)
	if err != nil {
		return err
	}

	err = ValidateHosting(envOption)
	if err != nil {
		return err
	}

	root := appRoot()
	err = makeService(root, "0.0.0.0", envOption["YAO_PORT"], envOption["YAO_STUDIO_PORT"], envOption["YAO_LANG"])
	if err != nil {
		return err
	}

	err = makeDB(root, dbOption)
	if err != nil {
		return err
	}

	err = makeSession(root)
	if err != nil {
		return err
	}

	err = makeLog(root)
	if err != nil {
		return err
	}

	err = makeDirs(root)
	if err != nil {
		return err
	}

	err = makeDemoWidgets(root)
	if err != nil {
		return err
	}

	err = makeMigrate(root)
	if err != nil {
		return err
	}

	err = makeSetup(root)
	if err != nil {
		return err
	}

	return nil
}

func makeService(root string, host string, port string, studioPort string, lang string) error {
	file := filepath.Join(root, ".env")
	err := envSet(file, "YAO_HOST", host)
	if err != nil {
		return err
	}

	err = envSet(file, "YAO_PORT", port)
	if err != nil {
		return err
	}

	err = envSet(file, "YAO_STUDIO_PORT", studioPort)
	if err != nil {
		return err
	}

	return envSet(file, "YAO_LANG", lang)
}

func makeDB(root string, option map[string]string) error {

	driver, dsn, err := getDSN(option)
	if driver != "mysql" && driver != "sqlite3" {
		return fmt.Errorf("数据库驱动应该为: mysql/sqlite3")
	}

	file := filepath.Join(root, ".env")
	if err != nil {
		return err
	}

	err = envSet(file, "YAO_DB_DRIVER", driver)
	if err != nil {
		return err
	}

	return envSet(file, "YAO_DB_PRIMARY", dsn)
}

func makeSession(root string) error {
	file := filepath.Join(root, ".env")
	if has, _ := envHas(file, "YAO_SESSION_STORE"); has {
		return nil
	}

	if has, _ := envHas(file, "YAO_SESSION_FILE"); has {
		return nil
	}

	err := os.MkdirAll(filepath.Join(root, "data"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = envSet(file, "YAO_SESSION_STORE", "file")
	if err != nil {
		return err
	}

	ssfile := filepath.Join(root, "db", ".session")
	return envSet(file, "YAO_SESSION_FILE", ssfile)
}

func makeLog(root string) error {
	file := filepath.Join(root, ".env")
	if has, _ := envHas(file, "YAO_LOG_MODE"); has {
		return nil
	}

	if has, _ := envHas(file, "YAO_LOG"); has {
		return nil
	}

	err := os.MkdirAll(filepath.Join(root, "data"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = envSet(file, "YAO_LOG_MODE", "TEXT")
	if err != nil {
		return err
	}

	logfile := filepath.Join(root, "logs", "application.log")
	return envSet(file, "YAO_LOG", logfile)
}

func makeDirs(root string) error {

	err := os.MkdirAll(filepath.Join(root, "public"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = os.MkdirAll(filepath.Join(root, "services"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = os.MkdirAll(filepath.Join(root, "studio"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = os.MkdirAll(filepath.Join(root, "logs"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = os.MkdirAll(filepath.Join(root, "db"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	return nil
}

func makeDemoWidgets(root string) error {

	dsl := dsl.New(root)

	// make app
	ok, err := makeAppWidget(dsl, root)
	if err != nil || !ok {
		return err
	}

	// make flows
	err = makeMenuFlowWidget(dsl, root)
	if err != nil {
		return err
	}

	// make logins
	err = makeLoginWidget(dsl, root)
	if err != nil {
		return err
	}

	// make models
	err = makeModelWidget(dsl, root)
	if err != nil {
		return err
	}

	// make tables & forms
	err = makeTableFormWidget(dsl, root)
	if err != nil {
		return err
	}

	// make chart
	err = makeChartWidget(dsl, root)
	if err != nil {
		return err
	}

	// make scripts
	err = makeScript(root)
	if err != nil {
		return err
	}

	// make scripts
	err = makeLang(root)
	if err != nil {
		return err
	}

	// make index page
	err = makeIndexPage(root)
	if err != nil {
		return err
	}

	return nil
}

func makeAppWidget(dsl fs.FileSystem, root string) (bool, error) {

	file := filepath.Join("app.json")
	if _, err := os.Stat(filepath.Join(root, file)); err == nil { // exists
		return false, nil
	}

	_, err := dsl.WriteFile(file, []byte(fmt.Sprintf(`
	{
		"xgen": "1.0",
		"name": "::Demo Application",
		"short": "::Demo",
		"description": "::Another yao application",
		"version": "%s",
		"adminRoot": "admin",
		"setup": "scripts.demo.Data",
		"menu": {
		  "process": "flows.app.menu",
		  "args": ["demo"]
		},
		"optional": {
		  "hideNotification": true,
		  "hideSetting": false
		}
	}
	`, share.VERSION)), 0644)

	if err != nil {
		return false, err
	}

	return true, nil
}

func makeMenuFlowWidget(dsl fs.FileSystem, root string) error {

	file := filepath.Join("flows", "app", "menu.flow.json")
	if _, err := os.Stat(filepath.Join(root, file)); err == nil { // exists
		return nil
	}

	_, err := dsl.WriteFile(file, []byte(`
	{
		"name": "APP Menu",
		"nodes": [],
		"output": [
		  {
			"blocks": 0,
			"icon": "icon-activity",
			"id": 1,
			"name": "图表",
			"parent": null,
			"path": "/x/Chart/dashboard",
			"visible_menu": 0
		  },
		  {
			"blocks": 0,
			"icon": "icon-book",
			"id": 2,
			"name": "表格",
			"parent": null,
			"path": "/x/Table/pet",
			"visible_menu": 1,
			"children": [
			  {
				"blocks": 0,
				"icon": "icon-book",
				"name": "宠物列表",
				"id": 2010,
				"parent": 2,
				"path": "/x/Table/pet",
				"visible_menu": 1
			  }
			]
		  },
		  {
			"blocks": 0,
			"icon": "icon-clipboard",
			"id": 2,
			"name": "表单",
			"parent": null,
			"path": "/x/Form/pet/1/edit",
			"visible_menu": 1,
			"children": [
			  {
				"blocks": 0,
				"icon": "icon-clipboard",
				"name": "编辑模式",
				"id": 2010,
				"parent": 2,
				"path": "/x/Form/pet/1/edit",
				"visible_menu": 1
			  },
			  {
				"blocks": 0,
				"icon": "icon-clipboard",
				"name": "查看模式",
				"id": 2010,
				"parent": 2,
				"path": "/x/Form/pet/1/view",
				"visible_menu": 1
			  }
			]
		  }
		]
	}	  
	`), 0644)

	if err != nil {
		return err
	}

	return nil
}

func makeLoginWidget(dsl fs.FileSystem, root string) error {
	file := filepath.Join("logins", "admin.login.json")
	if _, err := os.Stat(filepath.Join(root, file)); err == nil { // exists
		return nil
	}
	_, err := dsl.WriteFile(file, []byte(`
	{
		"name": "::Admin Login",
		"action": {
		  "process": "yao.login.Admin",
		  "args": [":payload"]
		},
		"layout": {
		  "entry": "/x/Chart/dashboard",
		  "slogan": "::Make Your Dream With Yao App Engine",
		  "site": "https://yaoapps.com?from=instance-admin-login"
		}
	}
	`), 0644)

	if err != nil {
		return err
	}

	file = filepath.Join("logins", "user.login.json")
	if _, err := os.Stat(filepath.Join(root, file)); err == nil { // exists
		return nil
	}
	_, err = dsl.WriteFile(file, []byte(`
	{
		"name": "::User Login",
		"action": {
		  "process": "scripts.user.Login",
		  "args": [":payload"]
		},
		"layout": {
		  "entry": "/x/Table/pet",
		  "slogan": "::Make Your Dream With Yao App Engine",
		  "site": "https://yaoapps.com/from=instance-user-login"
		}
	}	  
	`), 0644)
	if err != nil {
		return err
	}

	return nil
}

func makeModelWidget(dsl fs.FileSystem, root string) error {
	file := filepath.Join("models", "pet.mod.json")
	if _, err := os.Stat(filepath.Join(root, file)); err == nil { // exists
		return nil
	}
	_, err := dsl.WriteFile(file, []byte(`
	{
		"name": "Pet",
		"table": { "name": "pet", "comment": "宠物表" },
		"columns": [
		  { "name": "id", "comment": "ID", "type": "ID" },
		  {
			"name": "name",
			"comment": "昵称",
			"type": "string",
			"length": 80,
			"index": true,
			"nullable": true
		  },
		  {
			"name": "type",
			"comment": "类型",
			"type": "enum",
			"option": ["cat", "dog", "others"],
			"index": true
		  },
		  {
			"name": "status",
			"comment": "入院状态",
			"type": "enum",
			"option": ["checked", "curing", "cured"],
			"index": true
		  },
		  {
			"name": "mode",
			"comment": "状态",
			"type": "enum",
			"option": ["enabled", "disabled"],
			"index": true
		  },
		  {
			"name": "online",
			"comment": "是否在线",
			"type": "boolean",
			"default": false,
			"index": true
		  },
		  {
			"name": "curing_status",
			"comment": "治疗状态",
			"type": "enum",
			"default": "0",
			"option": ["0", "1", "2"],
			"index": true
		  },
		  {
			"name": "stay",
			"comment": "入院时间",
			"type": "integer"
		  },
		  {
			"name": "cost",
			"comment": "花费",
			"type": "integer"
		  },
		  {
			"name": "doctor_id",
			"type": "integer",
			"comment": "相关职工",
			"nullable": true
		  },
		  {
			"name": "images",
			"type": "json",
			"comment": "相关图片",
			"nullable": true
		  },
		  {
			"name": "test_string",
			"comment": "测试字段（字符串）",
			"type": "string",
			"nullable": true
		  },
		  {
			"name": "test_number",
			"comment": "测试字段（数字）",
			"type": "integer",
			"nullable": true
		  },
		  {
			"name": "test_array",
			"comment": "测试字段（数组）",
			"type": "json",
			"nullable": true
		  }
		],
		"relations": {},
		"values": [],
		"indexes": [],
		"option": { "timestamps": true,"soft_deletes": true }
	}	  
	`), 0644)

	if err != nil {
		return err
	}

	return nil
}

func makeTableFormWidget(dsl fs.FileSystem, root string) error {
	file := filepath.Join("tables", "pet.tab.json")
	if _, err := os.Stat(filepath.Join(root, file)); err == nil { // exists
		return nil
	}
	_, err := dsl.WriteFile(file, []byte(`
	{
		"name": "::Pet Admin Bind Model",
		"action": {
		  "bind": { "model": "pet", "option": {} }
		}
	}
	`), 0644)

	if err != nil {
		return err
	}

	file = filepath.Join("forms", "pet.form.json")
	if _, err := os.Stat(filepath.Join(root, file)); err == nil { // exists
		return nil
	}
	_, err = dsl.WriteFile(file, []byte(`
	{
		"name": "::Pet Admin Bind Table",
		"action": {
		  "bind": { "table": "pet", "option": {} }
		}
	}
	`), 0644)
	if err != nil {
		return err
	}

	return nil
}

func makeChartWidget(dsl fs.FileSystem, root string) error {

	file := filepath.Join("charts", "dashboard.chart.json")
	if _, err := os.Stat(filepath.Join(root, file)); err == nil { // exists
		return nil
	}

	_, err := dsl.WriteFile(file, []byte(`
	{
		"name": "宠物医院数据图表",
	  
		"config":{"full":false},
	  
		"action": {
		  "before:data": "scripts.stat.BeforeData",
		  "data": { "process": "scripts.stat.Data", "default": ["2022-09-20"] },
		  "after:data": "scripts.stat.AfterData"
		},
	  
		"layout": {
		  "operation": {
			"actions": [
			  {
				"title": "跳转至大屏",
				"icon": "icon-airplay",
				"action": { "Common.historyPush": { "pathname": "/x/Cool/demo" } }
			  }
			]
		  },
	  
		  "filter": {
			"columns": [{ "name": "时间区间", "width": 6 }]
		  },
	  
		  "chart": {
			"columns": [
			  { "name": "宠物数量", "width": 6 },
			  { "name": "宠物类型", "width": 6 },
			  { "name": "当月收入", "width": 6 },
			  { "name": "医师数量", "width": 6 },
			  { "name": "宠物数量_上月", "width": 6 },
			  { "name": "宠物类型_上月", "width": 6 },
			  { "name": "当月收入_上月", "width": 6 },
			  { "name": "医师数量_上月", "width": 6 },
			  { "name": "收入", "width": 8 },
			  { "name": "支出", "width": 8 },
			  { "name": "综合评分", "width": 8 },
			  { "name": "收入_折线图", "width": 8 },
			  { "name": "支出_折线图", "width": 8 },
			  { "name": "综合评分_折线图", "width": 8 },
			  { "name": "类型排布", "width": 12 },
			  { "name": "状态分布", "width": 12 },
			  { "name": "综合消费", "width": 24 }
			]
		  }
		},
	  
		"fields": {
		  "filter": {
			"时间区间": {
			  "bind": "range",
			  "edit": { "type": "RangePicker", "props": {} }
			},
			"状态": {
			  "bind": "status",
			  "edit": {
				"type": "Select",
				"props": {
				  "xProps": {
					"$remote": {
					  "process": "models.pet.Get",
					  "query": { "select": "name,status", "limit": 2 }
					}
				  }
				}
			  }
			}
		  },
		  "chart": {
			"收入": {
			  "bind": "income",
			  "link": "/x/Table/pet",
			  "out": "scripts.stat.Income",
			  "view": {
				"type": "NumberChart",
				"props": {
				  "chartHeight": 150,
				  "prefix": "¥",
				  "decimals": 2,
				  "nameKey": "date",
				  "valueKey": "value"
				}
			  }
			},
			"支出": {
			  "bind": "cost",
			  "link": "/x/Table/pet",
			  "view": {
				"type": "NumberChart",
				"props": {
				  "chartHeight": 150,
				  "color": "red",
				  "prefix": "¥",
				  "decimals": 2,
				  "nameKey": "date",
				  "valueKey": "value"
				}
			  }
			},
			"综合评分": {
			  "bind": "rate",
			  "link": "/x/Table/pet",
			  "view": {
				"type": "NumberChart",
				"props": {
				  "chartHeight": 150,
				  "color": "orange",
				  "unit": "分",
				  "decimals": 1,
				  "nameKey": "date",
				  "valueKey": "value"
				}
			  }
			},
			"收入_折线图": {
			  "bind": "income",
			  "link": "/x/Table/pet",
			  "view": {
				"type": "NumberChart",
				"props": {
				  "cardStyle": { "padding": 0 },
				  "type": "line",
				  "chartHeight": 120,
				  "prefix": "¥",
				  "decimals": 2,
				  "nameKey": "date",
				  "valueKey": "value"
				}
			  }
			},
			"支出_折线图": {
			  "bind": "cost",
			  "link": "/x/Table/pet",
			  "view": {
				"type": "NumberChart",
				"props": {
				  "cardStyle": { "padding": 0 },
				  "type": "line",
				  "chartHeight": 120,
				  "color": "red",
				  "prefix": "¥",
				  "decimals": 2,
				  "nameKey": "date",
				  "valueKey": "value"
				}
			  }
			},
			"综合评分_折线图": {
			  "bind": "rate",
			  "link": "/x/Table/pet",
			  "view": {
				"type": "NumberChart",
				"props": {
				  "cardStyle": { "padding": 0 },
				  "type": "line",
				  "chartHeight": 120,
				  "color": "orange",
				  "unit": "分",
				  "decimals": 1,
				  "nameKey": "date",
				  "valueKey": "value"
				}
			  }
			},
			"宠物数量": {
			  "bind": "pet_count",
			  "link": "/x/Table/pet",
			  "view": { "type": "Number", "props": { "unit": "个" } }
			},
			"宠物类型": {
			  "bind": "pet_type",
			  "view": { "type": "Number", "props": { "unit": "种" } }
			},
			"当月收入": {
			  "bind": "income_monthly",
			  "view": { "type": "Number", "props": { "unit": "元" } }
			},
			"医师数量": {
			  "bind": "doctor_count",
			  "view": { "type": "Number", "props": { "unit": "个" } }
			},
			"宠物数量_上月": {
			  "bind": "prev_pet_count",
			  "link": "/x/Table/pet",
			  "view": {
				"type": "Number",
				"props": {
				  "cardStyle": { "padding": 0 },
				  "unit": "个",
				  "prev_title": "上月数据"
				}
			  }
			},
			"宠物类型_上月": {
			  "bind": "prev_pet_type",
			  "link": "/x/Table/pet",
			  "view": {
				"type": "Number",
				"props": {
				  "cardStyle": { "padding": 0 },
				  "unit": "种",
				  "prev_title": "上月数据"
				}
			  }
			},
			"当月收入_上月": {
			  "bind": "prev_income_monthly",
			  "link": "/x/Table/pet",
			  "view": {
				"type": "Number",
				"props": {
				  "cardStyle": { "padding": 0 },
				  "unit": "元",
				  "prev_title": "上月数据"
				}
			  }
			},
			"医师数量_上月": {
			  "bind": "prev_doctor_count",
			  "link": "/x/Table/pet",
			  "view": {
				"type": "Number",
				"props": {
				  "cardStyle": { "padding": 0 },
				  "unit": "个",
				  "prev_title": "上月数据"
				}
			  }
			},
			"类型排布": {
			  "bind": "datasource_type",
			  "link": "/x/Table/pet",
			  "view": {
				"type": "Pie",
				"props": {
				  "height": 240,
				  "nameKey": "type",
				  "series": [
					{
					  "valueKey": "count",
					  "roseType": "area",
					  "radius": [10, 100],
					  "center": ["60%", "50%"],
					  "itemStyle": { "borderRadius": 6 }
					}
				  ]
				}
			  },
			  "refer": {
				"type": "Table",
				"props": {
				  "columns": [
					{ "title": "类型", "dataIndex": "type" },
					{ "title": "数量", "dataIndex": "count" }
				  ]
				}
			  }
			},
			"状态分布": {
			  "bind": "datasource_status",
			  "view": {
				"type": "Bar",
				"props": {
				  "height": 240,
				  "nameKey": "status",
				  "axisLabel": { "interval": 0, "fontSize": 12 },
				  "series": [
					{
					  "valueKey": "count",
					  "type": "bar",
					  "colorBy": "data",
					  "itemStyle": { "borderRadius": 6 },
					  "splitLine": { "show": false },
					  "axisLabel": { "show": false }
					}
				  ]
				}
			  },
			  "refer": {
				"type": "Table",
				"props": {
				  "columns": [
					{ "title": "状态", "dataIndex": "status" },
					{ "title": "数量", "dataIndex": "count" }
				  ]
				}
			  }
			},
			"综合消费": {
			  "bind": "datasource_cost",
			  "view": {
				"type": "LineBar",
				"props": {
				  "height": 240,
				  "nameKey": "name",
				  "axisLabel": { "interval": 0, "fontSize": 12 },
				  "series": [
					{
					  "valueKey": "stay",
					  "type": "line",
					  "smooth": true,
					  "symbolSize": 8,
					  "itemStyle": { "borderRadius": 6 },
					  "splitLine": { "show": false },
					  "axisLabel": { "show": false }
					},
					{
					  "valueKey": "cost",
					  "type": "bar",
					  "itemStyle": { "borderRadius": 6 },
					  "splitLine": { "show": false },
					  "axisLabel": { "show": false }
					}
				  ]
				}
			  }
			}
		  }
		}
	}	  
	`), 0644)

	if err != nil {
		return err
	}

	return nil
}

func makeScript(root string) error {

	fs := system.New(filepath.Join(root, "scripts"))
	file := "demo.js"
	if _, err := os.Stat(filepath.Join(root, "scripts", file)); err == nil { // exists
		return nil
	}
	_, err := fs.WriteFile(file, []byte(`
/**
 * Demo Data
 */
function Data() {
	return Process(
	  "yao.table.Insert",
	  "pet",
	  ["name", "type", "status", "mode", "stay", "cost", "doctor_id"],
	  [
		["Cookie", "cat", "checked", "enabled", 200, 105, 1],
		["Baby", "dog", "checked", "enabled", 186, 24, 1],
		["Poo", "others", "checked", "enabled", 199, 66, 1],
	  ]
	);
}	  
	`), 0644)

	if err != nil {
		return err
	}

	file = filepath.Join("stat.js")
	if _, err := os.Stat(filepath.Join(root, "scripts", file)); err == nil { // exists
		return nil
	}
	_, err = fs.WriteFile(file, []byte(`
/**
 * before:data hook
 * @param {*} params
 * @returns
 */
function BeforeData(params) {
 log.Info("[chart] before data hook: %s", JSON.stringify(params));
 return [params];
}
   
/**
 * after:data hook
 * @param {*} data
 * @returns
 */
function AfterData(data) {
 log.Info("[chart] after data hook: %s", JSON.stringify(data));
 return data;
}
   
/**
 * Get Data
 * @param {*} params
 */
function Data(params) {
 log.Info("[chart] process data query: %s", JSON.stringify(params));
 return {
   income: [
	 { value: 40300, date: "2022-1-1" },
	 { value: 50800, date: "2022-2-1" },
	 { value: 31300, date: "2022-3-1" },
	 { value: 48800, date: "2022-4-1" },
	 { value: 69900, date: "2022-5-1" },
	 { value: 37800, date: "2022-6-1" },
   ],
   cost: [
	 { value: 28100, date: "2022-1-1" },
	 { value: 23000, date: "2022-2-1" },
	 { value: 29300, date: "2022-3-1" },
	 { value: 26700, date: "2022-4-1" },
	 { value: 26400, date: "2022-5-1" },
	 { value: 31200, date: "2022-6-1" },
   ],
   rate: [
	 { value: 8.0, date: "2022-1-1" },
	 { value: 7.6, date: "2022-2-1" },
	 { value: 9.1, date: "2022-3-1" },
	 { value: 8.4, date: "2022-4-1" },
	 { value: 6.9, date: "2022-5-1" },
	 { value: 9.0, date: "2022-6-1" },
   ],
   pet_count: 54,
   pet_type: 8,
   income_monthly: 68900,
   doctor_count: 23,
   prev_pet_count: { current: 54, prev: 45 },
   prev_pet_type: { current: 8, prev: 13 },
   prev_income_monthly: { current: 68900, prev: 92000 },
   prev_doctor_count: { current: 23, prev: 27 },
   datasource_type: [
	 { type: "猫猫", count: 18 },
	 { type: "狗狗", count: 6 },
	 { type: "其他", count: 3 },
   ],
   datasource_status: [
	 { status: "已查看", count: 3 },
	 { status: "治疗中", count: 12 },
	 { status: "已治愈", count: 9 },
   ],
   datasource_cost: [
	 { name: "毛毛", stay: 3, cost: 2000 },
	 { name: "阿布", stay: 6, cost: 4200 },
	 { name: "咪咪", stay: 7, cost: 6000 },
	 { name: "狗蛋", stay: 1, cost: 1000 },
   ],
 };
}

/**
 * Compute out
 * @param {*} field
 * @param {*} value
 * @param {*} data
 * @returns
 */
function Income(field, value, data) {
 log.Info(
   "[chart] Income Compute: %s",
   JSON.stringify({ field: field, value: value, data: data })
 );
 return value;
}   
	`), 0644)
	if err != nil {
		return err
	}

	return nil
}

func makeLang(root string) error {
	fs := system.New(filepath.Join(root, "langs"))
	file := filepath.Join("zh-cn", "global.yml")
	if _, err := os.Stat(filepath.Join(root, "langs", file)); err == nil { // exists
		return nil
	}

	_, err := fs.WriteFile(file, []byte(`
Demo: 演示
Demo Application: 示例应用
Another yao application: 又一个 YAO 应用`), 0644)

	if err != nil {
		return err
	}

	file = filepath.Join("zh-hk", "langs", "global.yml")
	if _, err := os.Stat(filepath.Join(root, file)); err == nil { // exists
		return nil
	}
	_, err = fs.WriteFile(file, []byte(`
Demo: 演示
Demo Application: 示例應用
Another yao application: 又一個YAO應用`), 0644)

	if err != nil {
		return err
	}

	return nil
}

func makeIndexPage(root string) error {
	fs := system.New(filepath.Join(root, "public"))
	file := "index.html"
	if _, err := os.Stat(filepath.Join(root, "public", file)); err == nil { // exists
		return nil
	}

	_, err := fs.WriteFile(file, []byte(`
It works!`), 0644)

	if err != nil {
		return err
	}

	return nil
}

func makeMigrate(root string) error {

	cfg, err := getConfig()
	if err != nil {
		return err
	}

	err = engine.Load(cfg)
	if err != nil {
		return err
	}

	// Do Stuff Here
	for _, mod := range gou.Models {
		has, err := mod.HasTable()
		if err != nil {
			return err
		}

		if has {
			log.Warn("%s (%s) table already exists", mod.ID, mod.MetaData.Table.Name)
			continue
		}

		err = mod.Migrate(false)
		if err != nil {
			return err
		}
	}

	return nil
}

func makeSetup(root string) error {

	cfg, err := getConfig()
	if err != nil {
		return err
	}

	err = engine.Load(cfg)
	if err != nil {
		return err
	}

	if app.Setting != nil && app.Setting.Setup != "" {

		if strings.HasPrefix(app.Setting.Setup, "studio.") {
			names := strings.Split(app.Setting.Setup, ".")
			if len(names) < 3 {
				return fmt.Errorf("setup studio script %s error", app.Setting.Setup)
			}

			service := strings.Join(names[1:len(names)-1], ".")
			method := names[len(names)-1]
			req := gou.Yao.New(service, method)
			_, err := req.RootCall(cfg)
			if err != nil {
				return err
			}
			return nil
		}

		p, err := gou.ProcessOf(app.Setting.Setup, cfg)
		if err != nil {
			return err
		}
		_, err = p.Exec()
		if err != nil {
			return err
		}
	}

	return nil
}
