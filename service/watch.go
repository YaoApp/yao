package service

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/share"
)

var watchShutdown = make(chan bool, 1) // shutdown signal
var watchReady = make(chan bool, 1)    // ready signal
var excludes = map[string]bool{"ui": true, "db": true, "data": true}
var handlers = map[string]func(root string, file string, event string, cfg config.Config){
	"models": watchModel,
}

// Watch the application code change for hot update
func Watch(cfg config.Config) (err error) {
	go func() { err = watchStart(cfg) }()
	select {
	case <-watchReady:
		return nil
	}
}

// StopWatch stop watching the code change
func StopWatch() {
	watchShutdown <- true
	time.Sleep(200 * time.Millisecond)
}

func watchStart(cfg config.Config) error {

	root := cfg.Root

	// recive interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	shutdown := make(chan bool, 1)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}

	dirs, err := ioutil.ReadDir(root)
	if err != nil {
		return err
	}

	// Add path
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		name := dir.Name()
		if _, has := excludes[name]; has {
			continue
		}

		if strings.HasPrefix(name, ".") {
			continue
		}

		filename := filepath.Join(root, name)
		err := watcher.Add(filename)
		if err != nil {
			log.Error("[Watch] %s", err.Error())
			return err
		}

		fmt.Println(color.GreenString("[Watch] Watching %s", name))
		log.Info("[Watch] Watching: %s", filename)

		// sub dir
		depth := 0
		err = filepath.WalkDir(filename, func(path string, d fs.DirEntry, err error) error {
			depth = depth + 1
			if depth == 1 {
				return nil
			}

			if !d.IsDir() {
				return nil
			}

			log.Info("[Watch] Watching: %s", path)
			err = watcher.Add(path)
			if err != nil {
				log.Error("[Watch] %s", err.Error())
			}
			return nil
		})

		if err != nil {
			log.Error("[Watch] %s", err.Error())
		}
	}

	// event handler
	go func() {
		for {
			select {
			case <-shutdown:
				log.Info("[Watch] The event handler exit")
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				relpath := strings.TrimPrefix(event.Name, root)
				if strings.HasPrefix(relpath, string(os.PathSeparator)) {
					relpath = strings.TrimPrefix(relpath, string(os.PathSeparator))
				}

				pi := strings.Split(relpath, string(os.PathSeparator))
				widget := pi[0]

				watchHanler := watchReload
				if handler, has := handlers[widget]; has {
					watchHanler = handler
				}

				if _, has := excludes[widget]; !has {
					base := filepath.Base(event.Name)
					isdir := true
					if strings.HasSuffix(base, ".yao") || strings.HasSuffix(base, ".json") || strings.HasSuffix(base, ".js") {
						isdir = false
					}

					events := strings.Split(event.Op.String(), "|")
					for _, eventType := range events {

						// ADD / REMOVE Watching dir
						if isdir {
							switch eventType {
							case "CREATE":
								log.Info("[Watch] Watching: %s", event.Name)
								watcher.Add(event.Name)
								break
							case "REMOVE":
								log.Info("[Watch] Unwatching: %s", event.Name)
								watcher.Remove(event.Name)
								break
							}
							continue
						}

						log.Info("[Watch] %s %s", eventType, event.Name)
						watchHanler(root, event.Name, eventType, cfg)
					}
				}
				break

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println(color.RedString("[Watch] %s", err.Error()))
				log.Error("[Watch] %s", err.Error())
				break
			}
		}
	}()

	fmt.Println(color.GreenString("[Watch] Started"))
	watchReady <- true

	for {
		select {
		case <-watchShutdown:
			shutdown <- true
			log.Info("[Watch] Stopped")
			fmt.Println(color.YellowString("[Watch] Stopped"))
			return nil

		case <-interrupt:
			shutdown <- true
			log.Info("[Watch] Stopped")
			fmt.Println(color.YellowString("[Watch] Stopped"))
			return nil
		}
	}
}

func watchModel(root string, file string, event string, cfg config.Config) {
	name := share.SpecName(root, file)
	switch event {
	case "CREATE":
		content, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Println(color.RedString("[Watch] Model: %s %s", name, err.Error()))
			return
		}
		_, err = gou.LoadModelReturn(string(content), name)
		if err != nil {
			fmt.Println(color.RedString("[Watch] Model: %s %s", name, err.Error()))
			return
		}

		// mod.Migrate(true)
		fmt.Println(color.GreenString("[Watch] Model: %s Created (Please run yao migrate manually)", name))
		break

	case "WRITE":
		content, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Println(color.RedString("[Watch] Model: %s %s", name, err.Error()))
			return
		}
		_, err = gou.LoadModelReturn(string(content), name)
		if err != nil {
			fmt.Println(color.RedString("[Watch] Model: %s %s", name, err.Error()))
			return
		}
		// mod.Migrate(false)
		fmt.Println(color.GreenString("[Watch] Model: %s Reloaded (Please run yao migrate manually)", name))
		break

	case "REMOVE", "RENAME":
		delete(gou.Models, name)
		fmt.Println(color.GreenString("[Watch] Model: %s Removed", name))
		break
	}
}

func watchReload(root string, file string, event string, cfg config.Config) {

	switch event {
	case "CREATE", "WRITE", "REMOVE":

		err := share.DBClose()
		if err != nil {
			fmt.Println(color.RedString("[Watch] Reload: %s", err.Error()))
		}

		err = engine.Load(config.Conf) // 加载脚本等
		if err != nil {
			fmt.Println(color.RedString("[Watch] Reload: %s", err.Error()))
		}

		// Restart Server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		StopWithContext(ctx, func() {
			go Start()
			fmt.Println(color.GreenString("[Watch] Reload Completed"))
		})
	}
}
