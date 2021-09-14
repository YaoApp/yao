package global

import (
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/yaoapp/kun/exception"
)

var watchDone = []chan bool{}
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
	defer watcher.Close()

	watchDone = append(watchDone, make(chan bool))
	last := len(watchDone) - 1
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
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(root)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("开始监听目录:", root)

	// 监听子目录
	filepath.Walk(root, func(subfolder string, info os.FileInfo, err error) error {
		if err != nil {
			exception.Err(err, 500).Throw()
			return err
		}

		if subfolder == root {
			return nil
		}

		if info.IsDir() {
			go Watch(subfolder, cb)
		}
		return nil
	})

	select {
	case v := <-watchDone[last]:
		log.Println("停止监听目录:", root)
		if v == true {
			break
		}
	}
}

// StopWatch 停止监听
func StopWatch() {
	for i := range watchDone {
		log.Println("发送停止信号:", i)
		watchDone[i] <- true
	}
	watchDone = []chan bool{}
}
