package service

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

func TestWatch(t *testing.T) {

	share.DBConnect(config.Conf.DB)
	err := Watch(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	defer StopWatch()

	createDir(t)
	renameDir(t)

	createModel(t)
	changeModel(t)
	renameModel(t)
	removeModel(t)

	createModel(t)
	removeDir(t)
}

func TestWatchReload(t *testing.T) {
	go Start()
	defer Stop(func() {})
	share.DBConnect(config.Conf.DB)
	watchReload("", "", "", config.Conf)
}

func createDir(t *testing.T) {
	root := config.Conf.Root
	file := filepath.Join(root, "models", "watch", "test")
	fmt.Println("CREATE-DIR", file)
	err := os.MkdirAll(file, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
}

func renameDir(t *testing.T) {
	root := config.Conf.Root
	file := filepath.Join(root, "models", "watch", "test")
	new := filepath.Join(root, "models", "watch", "test_new")
	fmt.Println("RENAME-DIR", file)
	err := os.Rename(file, new)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
}

func removeDir(t *testing.T) {
	root := config.Conf.Root
	file := filepath.Join(root, "models", "watch")
	fmt.Println("REMOVE-DIR", file)
	err := os.RemoveAll(file)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
}

func createModel(t *testing.T) {
	dsl := `
	{
		"name": "watch-test",
		"table": {
		  "name": "watch_test",
		  "comment": "WatchTest",
		  "engine": "InnoDB"
		},
		"columns": [
		  { "name": "id", "type": "ID" },
		  { "label": "Name", "name": "name", "type": "string", "index": true }
		],
		"relations": {},
		"option": { "timestamps": true, "soft_deletes": true }
	}	  
	`
	root := config.Conf.Root
	file := filepath.Join(root, "models", "watch", "test_new", "watch.mod.json")
	fmt.Println("CREATE", file)
	err := ioutil.WriteFile(file, []byte(dsl), 0644)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
}

func changeModel(t *testing.T) {
	dsl := `
	{
		"name": "watch-test",
		"table": {
		  "name": "watch_test",
		  "comment": "WatchTest",
		  "engine": "InnoDB"
		},
		"columns": [
		  { "name": "id", "type": "ID" },
		  { "label": "Name", "name": "name", "type": "string", "index": true },
		  { "label": "Data", "name": "data", "type": "json", "nullable": true }
		],
		"relations": {},
		"option": { "timestamps": true, "soft_deletes": true }
	}	  
	`
	root := config.Conf.Root
	file := filepath.Join(root, "models", "watch", "test_new", "watch.mod.json")
	fmt.Println("CHANGE", file)
	err := ioutil.WriteFile(file, []byte(dsl), 0644)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
}

func renameModel(t *testing.T) {
	root := config.Conf.Root
	file := filepath.Join(root, "models", "watch", "test_new", "watch.mod.json")
	new := filepath.Join(root, "models", "watch", "test_new", "watch_new.mod.json")
	fmt.Println("RENAME", new)
	err := os.Rename(file, new)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
}

func removeModel(t *testing.T) {
	root := config.Conf.Root
	file := filepath.Join(root, "models", "watch", "test_new", "watch_new.mod.json")
	fmt.Println("REMOVE", file)
	err := os.Remove(file)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
}
