package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: L("Initialize project"),
	Long:  L("Initialize project"),
	Run: func(cmd *cobra.Command, args []string) {
		Boot()
		checkDir()
		makeDirs()
		makeAppJSON()
		makeEnv()
		defaultApps()
		fmt.Println(color.GreenString(L("✨DONE✨")))
	},
}

func makeDirs() {
	dirs := []string{"db", "models", "flows", "scripts", "tables", "libs", "ui"}
	for _, name := range dirs {
		dirname := filepath.Join(config.Conf.Root, name)
		if _, err := os.Stat(dirname); errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(dirname, os.ModePerm); err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
				os.Exit(1)
			}
		}
	}
}

func defaultApps() {
	makeFile(filepath.Join("models", "pet.mod.json"), `{
	"name": "Pet",
	"table": { "name": "pet", "comment": "Pet" },
	"columns": [
	  { "label": "ID", "name": "id", "type": "ID", "comment": "ID" },
	  { "label": "SN", "name": "sn", "type": "string", "unique": true },
	  { "label": "Name", "name": "name", "type": "string", "index": true },
	  {
		"label": "Kind",
		"name": "kind",
		"type": "enum",
		"option": ["cat", "dog"],
		"default": "cat",
		"index": true
	  },
	  { "label": "Description", "name": "desc", "type": "string", "comment": "Description" }
	],
	"values": [
	  { "sn": "100001", "name": "Cookie", "kind": "cat", "desc": "a cat" },
	  { "sn": "100002", "name": "Beibei", "kind": "dog", "desc": "a dog" }
	],
	"option": { "timestamps": true, "soft_deletes": true }
}`)

	makeFile(filepath.Join("tables", "pet.tab.json"), `{
		"name": "Pet",
		"version": "1.0.0",
		"decription": "Pet admin",
		"bind": { "model": "pet" },
		"apis": {},
		"columns": {
		  "ID": {
			"label": "ID",
			"view": { "type": "label", "props": { "value": ":id" } }
		  },
		  "SN": {
			"label": "SN",
			"view": { "type": "label", "props": { "value": ":sn" } },
			"edit": { "type": "input", "props": { "value": ":sn" } }
		  },
		  "Name": {
			"label": "Name",
			"view": { "type": "label", "props": { "value": ":name" } },
			"edit": { "type": "input", "props": { "value": ":name" } }
		  },
		  "Kind": {
			"label": "Kind",
			"view": { "type": "label", "props": { "value": ":kind" } },
			"edit": {
			  "type": "select",
			  "props": {
				"value": ":kind",
				"options": [
				  { "label": "cat", "value": "cat" },
				  { "label": "dog", "value": "dog" }
				]
			  }
			}
		  },
		  "Description": {
			"label": "Description",
			"view": { "type": "label", "props": { "value": ":desc" } },
			"edit": { "type": "textArea", "props": { "value": ":desc", "rows": 4 } }
		  }
		},
		"filters": {
		  "Keywords": { "@": "f.Keywords", "in": ["where.name.match"]}
		},
		"list": {
		  "primary": "id",
		  "layout": {
			"columns": [
			  { "name": "ID", "width": 80 },
			  { "name": "SN", "width": 100 },
			  { "name": "Name", "width": 200 },
			  { "name": "Kind" }
			],
			"filters": [{ "name": "Keywords" }]
		  },
		  "actions": { "pagination": { "props": { "showTotal": true } } },
		  "option": {}
		},
		"edit": {
		  "primary": "id",
		  "layout": {
			"fieldset": [
			  {
				"columns": [
				  { "name": "SN", "width": 8 },
				  { "name": "Name", "width": 8 },
				  { "name": "Kind", "width": 8 },
				  { "name": "Description", "width": 24 }
				]
			  }
			]
		  },
		  "actions": { "cancel": {}, "save": {}, "delete": {} }
		}
	  }
	  `)

	makeFile(filepath.Join("flows", "setmenu.flow.json"), `{
		"label": "System Menu",
		"version": "1.0.0",
		"description": "Initialize system menu",
		"nodes": [
		  {
			"name": "Clean menu data",
			"engine": "xiang",
			"query": {
			  "sql": { "stmt": "delete from xiang_menu" }
			}
		  },
		  {
			"name": "Add new menu",
			"process": "models.xiang.menu.Save",
			"args": [
			  {
				"name": "Pet",
				"path": "/table/pet",
				"icon": "icon-github",
				"rank": 1,
				"status": "enabled",
				"visible_menu": 0,
				"blocks": 0
			  }
			]
		  }
		],
		"output": "done"
	  }
	  `)

	makeFile(filepath.Join("scripts", "day.js"), `
function NextDay(day) {
	today = new Date(day);
	today.setDate(today.getDate() + 1);
	return today.toISOString().slice(0, 19).split("T")[0];
}
`)

	makeFile(filepath.Join("ui", "index.html"), `It works! <a href="https://yaoapps.com">https://yaoapps.com</a>`)

	makeFile(filepath.Join("libs", "f.json"), `{
	"Keywords": {
	  "__comment": "{ '@': 'f.Keywords', 'in': ['where.name.match']}",
	  "label": "Keywords",
	  "bind": "{{$in.0}}",
	  "input": {
		"type": "input",
		"props": {
		  "placeholder": "type Keywords..."
		}
	  }
	}
  }
  `)
}

func makeEnv() {
	makeFile(".env", `
YAO_ENV=development # development | production
YAO_ROOT="`+config.Conf.Root+`"
YAO_HOST="0.0.0.0"
YAO_PORT="5099"
YAO_SESSION="memory"
YAO_LOG="`+config.Conf.Root+`/logs/application.log"
YAO_LOG_MODE="TEXT"  #  TEXT | JSON
YAO_JWT_SECRET="bLp@bi!oqo-2U+hoTRUG"
YAO_DB_DRIVER=sqlite3 # sqlite3 | mysql 
YAO_DB_PRIMARY="`+config.Conf.Root+`/db/yao.db"
`)
}

func makeAppJSON() {
	makeFile("app.json", `{
	"name": "Yao",
	"short": "Yao",
	"description": "Another yao app",
	"option": {
	  "nav_user": "xiang.user",
	  "nav_menu": "xiang.menu",
	  "hide_user": false,
	  "hide_menu": false,
	  "login": {
		"entry": {
		  "admin": "/table/pet"
		}
	  }
	}
}`)
}

func checkDir() {
	dirs := []string{"db", "models", "flows", "scripts", "tables", "libs", "ui", ".env", "app.json"}
	for _, name := range dirs {
		dirname := filepath.Join(config.Conf.Root, name)
		if _, err := os.Stat(dirname); !errors.Is(err, os.ErrNotExist) {
			fmt.Println(color.RedString(L("Fatal: %s"), dirname+" already existed"))
			os.Exit(1)
		}
	}
}

func makeFile(name string, source string) {
	filename := filepath.Join(config.Conf.Root, name)
	if _, err := os.Stat(filename); !errors.Is(err, os.ErrNotExist) {
		fmt.Println(color.RedString(L("Fatal: %s"), filename+" already existed"))
		os.Exit(1)
	}
	content := []byte(source)
	err := os.WriteFile(filename, content, 0644)
	if err != nil {
		fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		os.Exit(1)
	}
}
