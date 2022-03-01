package share

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/yaoapp/kun/exception"
)

var watchGroup sync.WaitGroup
var watchOp = map[fsnotify.Op]string{
	fsnotify.Create: "create",
	fsnotify.Write:  "write",
	fsnotify.Remove: "remove",
	fsnotify.Rename: "rename",
	fsnotify.Chmod:  "chmod",
}

// Watch 监听目录
func Watch(root string, cb func(op string, file string)) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		watcher.Close()
		watchGroup.Done()
	}()

	watchGroup.Add(1)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// 监听子目录
				if event.Op == fsnotify.Create {
					file, err := os.Open(event.Name)
					if err == nil {
						fi, err := file.Stat()
						file.Close()
						if err == nil && fi.IsDir() {
							Watch(event.Name, cb)
						}
					}
				}

				cb(watchOp[event.Op], event.Name)

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error:", err)
			}
		}
	}()

	err = watcher.Add(root)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(color.GreenString("Watching: %s", root))

	// 监听子目录
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			exception.Err(err, 500).Throw()
			return err
		}

		if path == root {
			return nil
		}

		if d.IsDir() {
			go Watch(path, cb)
		}
		return nil
	})

	watchGroup.Wait()

}
