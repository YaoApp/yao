package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// **** DEPRECATED ****
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: L("Service manager"),
	Long:  L("Service manager"),
	Run: func(cmd *cobra.Command, args []string) {
		loadService(config.Conf)
		command := "ps"
		if len(args) > 0 {
			command = args[0]
		}

		if len(args) > 1 {
			args = args[1:]
		} else {
			args = []string{}
		}

		switch command {
		case "ps":
			services()
			break
		case "install", "start", "stop", "remove", "status":
			serviceManage(command, args)
			break
		default:
			serviceHelp()
		}
	},
}

func serviceHelp() {
	fmt.Println("Usage:")
	fmt.Println("  ", color.GreenString(L("%s service [ps|start|stop|install|status] [all|name]"), share.BUILDNAME))
	fmt.Println("")
}

func serviceManageAll(command string) {
	for name := range gou.Services {
		if name == "all" {
			continue
		}
		serviceManage(command, []string{name})
	}
}

func serviceManage(command string, args []string) {
	if len(args) == 0 {
		serviceHelp()
		os.Exit(1)
	}

	defer func() {
		err := exception.Catch(recover())
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		}
	}()

	name := args[0]
	if name == "all" {
		serviceManageAll(command)
		return
	}

	srv := gou.SelectService(name)
	var status = ""
	var err error
	switch command {
	case "install":
		status, err = srv.Install()
		break
	case "start":
		status, err = srv.Start()
		break
	case "stop":
		status, err = srv.Stop()
		break
	case "remove":
		status, err = srv.Remove()
		break
	case "status":
		status, err = srv.Status()
		break
	}
	if err != nil {
		fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
	}

	fmt.Println(status)
}

func services() {
	for name, srv := range gou.Services {
		status, err := srv.Status()
		if err != nil {
			status = err.Error()
		}
		fmt.Println(
			color.GreenString("%s", name),
			srv.Name, "....",
			statusInfo(status),
		)
	}
	fmt.Println("")
}

func statusInfo(message string) string {
	if strings.Contains(message, "stopped") {
		return statusStop()
	} else if strings.Contains(message, "not installed") {
		return statusNotInstalled()
	} else if strings.Contains(message, "running") {
		return statusRunning()
	}
	return statusUnknown()
}

func statusUnknown() string {
	return fmt.Sprintf("[ %s ]", color.WhiteString("UNKNOWN"))
}

func statusRunning() string {
	return fmt.Sprintf("[ %s ]", color.GreenString("RUNNING"))
}

func statusNotInstalled() string {
	return fmt.Sprintf("[ %s ]", color.YellowString("NOT INSTALLED"))
}

func statusStop() string {
	return fmt.Sprintf("[ %s ]", color.RedString("STOPED"))
}

// Load 加载API
func loadService(cfg config.Config) {
	var root = filepath.Join(cfg.Root, "services")
	loadServiceFrom(root, "")
}

func loadServiceFrom(dir string, prefix string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	err := share.Walk(dir, ".srv.json", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		content := share.ReadFile(filename)
		_, err := gou.LoadService(string(content), name)
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}
	})

	return err
}
