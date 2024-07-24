package sui

import (
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/sui/core"
)

var watched sync.Map

// WatchCmd command
var WatchCmd = &cobra.Command{
	Use:   "watch",
	Short: L("Auto-build when the template file changes"),
	Long:  L("Auto-build when the template file changes"),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, color.RedString(L("yao sui watch <sui> <template> [data]")))
			return
		}

		Boot()

		cfg := config.Conf
		err := engine.Load(cfg, engine.LoadOption{Action: "sui.watch"})
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			return
		}

		id := args[0]
		template := args[1]

		var sessionData map[string]interface{}
		err = jsoniter.UnmarshalFromString(strings.TrimPrefix(data, "::"), &sessionData)
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			return
		}

		sid := uuid.New().String()
		if sessionData != nil && len(sessionData) > 0 {
			session.Global().ID(sid).SetMany(sessionData)
		}

		sui, has := core.SUIs[id]
		if !has {
			fmt.Fprintf(os.Stderr, color.RedString(("the sui " + id + " does not exist")))
			return
		}
		sui.WithSid(sid)

		exitSignal := make(chan os.Signal, 1)
		signal.Notify(exitSignal, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
		watchDone := make(chan uint8, 1)

		// -
		tmpl, err := sui.GetTemplate(template)
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			return
		}
		root := filepath.Join(cfg.DataRoot, tmpl.GetRoot())
		publicRoot, err := sui.PublicRootWithSid(sid)
		assetRoot := filepath.Join(publicRoot, "assets")
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			return
		}

		go watch(root, func(event, name string) {
			if event == "WRITE" || event == "CREATE" || event == "RENAME" {
				// @Todo build single page and sync single asset file to public
				fmt.Printf(color.WhiteString("Building...  "))

				tmpl, err := sui.GetTemplate(template)
				if err != nil {
					fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
					return
				}

				// Timecost
				start := time.Now()
				warnings, err := tmpl.Build(&core.BuildOption{SSR: true, AssetRoot: assetRoot})
				if err != nil {
					fmt.Fprint(os.Stderr, color.RedString(fmt.Sprintf("Failed: %s\n", err.Error())))
					return
				}

				if len(warnings) > 0 {
					fmt.Fprintln(os.Stderr, color.YellowString("\nWarnings:"))
					for _, warning := range warnings {
						fmt.Fprintln(os.Stderr, color.YellowString(warning))
					}
				}
				end := time.Now()
				timecost := end.Sub(start).Truncate(time.Millisecond)
				fmt.Printf(color.GreenString("Success (%s)\n"), timecost.String())
			}
		}, watchDone)

		fmt.Println(color.WhiteString("-----------------------"))
		fmt.Println(color.WhiteString("Public Root: /public%s", publicRoot))
		fmt.Println(color.WhiteString("   Template: %s", tmpl.GetRoot()))
		fmt.Println(color.WhiteString("    Session: %s", strings.TrimLeft(data, "::")))
		fmt.Println(color.WhiteString("-----------------------"))
		fmt.Println(color.GreenString("Watching..."))
		fmt.Println(color.GreenString("Press Ctrl+C to exit"))

		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			return
		}

		select {
		case <-exitSignal:
			watchDone <- 1
			return
		}
	},
}

func watch(root string, handler func(event string, name string), interrupt chan uint8) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	shutdown := make(chan bool, 1)

	err = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			if filepath.Base(path) == ".tmp" {
				return filepath.SkipDir
			}

			err = watcher.Add(path)
			if err != nil {
				return err
			}
			log.Info("[Watch] Watching: %s", strings.TrimPrefix(path, root))
			watched.Store(path, true)
		}
		return nil
	})

	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-shutdown:
				log.Info("[Watch] handler exit")
				return

			case event, ok := <-watcher.Events:
				if !ok {
					interrupt <- 1
					return
				}

				basname := filepath.Base(event.Name)
				isdir := true
				if strings.Contains(basname, ".") {
					isdir = false
				}

				events := strings.Split(event.Op.String(), "|")
				for _, eventType := range events {
					// ADD / REMOVE Watching dir
					if isdir {
						switch eventType {
						case "CREATE":
							log.Info("[Watch] Watching: %s", strings.TrimPrefix(event.Name, root))
							watcher.Add(event.Name)
							watched.Store(event.Name, true)
							break

						case "REMOVE":
							log.Info("[Watch] Unwatching: %s", strings.TrimPrefix(event.Name, root))
							watcher.Remove(event.Name)
							watched.Delete(event.Name)
							break
						}
						continue
					}

					file := strings.TrimLeft(event.Name, root)
					handler(eventType, file)
					log.Info("[Watch] %s %s", eventType, file)
				}

				break

			case err, ok := <-watcher.Errors:
				if !ok {
					interrupt <- 2
					return
				}
				log.Error("[Watch] Error: %s", err.Error())
				break
			}
		}
	}()

	for {
		select {
		case code := <-interrupt:
			shutdown <- true
			log.Info("[Watch] Exit(%d)", code)
			fmt.Println(color.YellowString("[Watch] Exit(%d)", code))
			return nil
		}
	}

}
