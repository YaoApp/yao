package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/plugin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	grpcclient "github.com/yaoapp/yao/grpc/client"
	ischedule "github.com/yaoapp/yao/schedule"
	"github.com/yaoapp/yao/share"
	itask "github.com/yaoapp/yao/task"
)

var runSilent = false
var runAuthPath string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: L("Execute process"),
	Long:  L("Execute process"),
	Run: func(cmd *cobra.Command, args []string) {

		// Resolve credential: --auth flag > ~/.yao/credentials > nil (local mode)
		cred := resolveCredential()

		if cred != nil {
			runGRPC(cred, args)
			return
		}

		runLocal(args)
	},
}

func init() {
	runCmd.PersistentFlags().BoolVarP(&runSilent, "silent", "s", false, L("Silent mode"))
	runCmd.PersistentFlags().StringVar(&runAuthPath, "auth", "", L("Path to credentials file"))
}

// resolveCredential loads credential from --auth flag or default path.
func resolveCredential() *Credential {
	if runAuthPath != "" {
		cred, err := LoadCredentialFrom(runAuthPath)
		if err != nil {
			color.Red("  %s %s\n", L("Failed to load credentials:"), err)
			os.Exit(1)
		}
		return cred
	}

	cred, _ := LoadCredential()
	return cred
}

// runGRPC executes a process via the remote gRPC server.
func runGRPC(cred *Credential, args []string) {
	if len(args) < 1 {
		if !runSilent {
			color.Red(L("Not enough arguments\n"))
			color.White(share.BUILDNAME + " help\n")
		} else {
			fmt.Print(L("Not enough arguments\n"))
		}
		os.Exit(1)
	}

	if cred.GRPCAddr == "" {
		color.Red("  %s\n", L("No gRPC address in credentials. Please re-login."))
		os.Exit(1)
	}

	name := args[0]
	if !runSilent {
		color.Green(L("Run: %s gRPC: %s\n"), name, cred.GRPCAddr)
	}

	pargs := parseRunArgs(args)

	argsJSON, err := jsoniter.Marshal(pargs)
	if err != nil {
		color.Red("  %s %s\n", L("Arguments:"), err.Error())
		os.Exit(1)
	}

	tm := grpcclient.NewTokenManager(cred.AccessToken, cred.RefreshToken, "")
	client, err := grpcclient.Dial(cred.GRPCAddr, tm)
	if err != nil {
		color.Red("  %s %s\n", L("gRPC connect failed:"), err.Error())
		os.Exit(1)
	}
	defer client.Close()

	data, err := client.Run(context.Background(), name, argsJSON, 0)
	if err != nil {
		if !runSilent {
			color.Red("  %s %s\n", L("Process:"), err.Error())
		} else {
			fmt.Printf("%s\n", err.Error())
		}
		os.Exit(1)
	}

	if !runSilent {
		color.White("--------------------------------------\n")
		color.White(L("%s Response\n"), name)
		color.White("--------------------------------------\n")
		var res interface{}
		if jsoniter.Unmarshal(data, &res) == nil {
			helper.Dump(res)
		} else {
			fmt.Printf("%s\n", data)
		}
		color.White("--------------------------------------\n")
		fmt.Printf("\033[32m✨DONE✨\033[0m \033[90mgRPC: %s\033[0m\n", cred.GRPCAddr)
	} else {
		fmt.Printf("%s\n", data)
	}
}

// runLocal executes a process locally (existing behavior).
func runLocal(args []string) {
	defer share.SessionStop()
	defer plugin.KillAll()

	defer func() {
		err := exception.Catch(recover())
		if err != nil {
			if !runSilent {
				color.Red(L("Fatal: %s\n"), err.Error())
				return
			}
			fmt.Printf("%s\n", err.Error())
		}
	}()

	// Auto-detect app root if not specified
	if appPath == "" {
		cwd, err := os.Getwd()
		if err == nil {
			if root, err := findAppRootFromPath(cwd); err == nil {
				appPath = root
			}
		}
	}

	Boot()

	// Set Runtime Mode
	config.Conf.Runtime.Mode = "standard"

	cfg := config.Conf
	cfg.Session.IsCLI = true
	if len(args) < 1 {
		if !runSilent {
			color.Red(L("Not enough arguments\n"))
			color.White(share.BUILDNAME + " help\n")
			return
		}
		fmt.Print(L("Not enough arguments\n"))
		return
	}

	loadWarnings, err := engine.Load(cfg, engine.LoadOption{Action: "run"})
	if err != nil {
		if !runSilent {
			color.Red(L("Engine: %s\n"), err.Error())
			return
		}

		fmt.Printf("%s\n", err.Error())
		return
	}

	name := args[0]
	if !runSilent {
		color.Green(L("Run: %s\n"), name)
	}

	pargs := parseRunArgs(args)

	// Start Tasks
	itask.Start()
	defer itask.Stop()

	// Start Schedules
	ischedule.Start()
	defer ischedule.Stop()

	p := process.NewWithContext(context.Background(), name, pargs...)
	res, err := p.Exec()
	if err != nil {
		if !runSilent {
			color.Red(L("Process: %s\n"), fmt.Sprintf("%s", strings.TrimPrefix(err.Error(), "Exception|404:")))
			return
		}
		fmt.Printf("%s\n", err.Error())
		return
	}

	if !runSilent {

		if len(loadWarnings) > 0 {
			fmt.Println(color.YellowString("---------------------------------"))
			fmt.Println(color.YellowString(L("Warnings")))
			fmt.Println(color.YellowString("---------------------------------"))
			for _, warning := range loadWarnings {
				fmt.Println(color.YellowString("[%s] %s", warning.Widget, warning.Error))
			}
			fmt.Printf("\n")
		}

		color.White("--------------------------------------\n")
		color.White(L("%s Response\n"), name)
		color.White("--------------------------------------\n")
		helper.Dump(res)
		color.White("--------------------------------------\n")
		color.Green(L("✨DONE✨\n"))
		return
	}

	// Silent mode output
	switch res.(type) {

	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		fmt.Printf("%v\n", res)
		return

	case string, []byte:
		fmt.Printf("%s\n", res)
		return

	default:
		txt, err := jsoniter.Marshal(res)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}
		fmt.Printf("%s\n", txt)
	}
}

// parseRunArgs parses the CLI arguments into process arguments, handling :: prefixed JSON.
func parseRunArgs(args []string) []interface{} {
	pargs := []interface{}{}
	for i, arg := range args {
		if i == 0 {
			continue
		}

		if strings.HasPrefix(arg, "::") {
			raw := strings.TrimPrefix(arg, "::")
			var v interface{}
			err := jsoniter.Unmarshal([]byte(raw), &v)
			if err != nil {
				color.Red(L("Arguments: %s\n"), err.Error())
				return pargs
			}
			pargs = append(pargs, v)
			if !runSilent {
				color.White("args[%d]: %s\n", i-1, raw)
			}
		} else if strings.HasPrefix(arg, "\\::") {
			cleaned := "::" + strings.TrimPrefix(arg, "\\::")
			pargs = append(pargs, cleaned)
			if !runSilent {
				color.White("args[%d]: %s\n", i-1, cleaned)
			}
		} else {
			pargs = append(pargs, arg)
			if !runSilent {
				color.White("args[%d]: %s\n", i-1, arg)
			}
		}
	}
	return pargs
}

// findAppRootFromPath finds the Yao application root directory by looking for app.yao
// It traverses up from the given path until it finds app.yao or reaches the filesystem root
func findAppRootFromPath(startPath string) (string, error) {
	// Get absolute path
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// If it's a file, start from its directory
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("path not found: %s", absPath)
	}

	var dir string
	if info.IsDir() {
		dir = absPath
	} else {
		dir = filepath.Dir(absPath)
	}

	// Traverse up to find app.yao
	for {
		// Check for app.yao, app.json, or app.jsonc
		for _, appFile := range []string{"app.yao", "app.json", "app.jsonc"} {
			appFilePath := filepath.Join(dir, appFile)
			if _, err := os.Stat(appFilePath); err == nil {
				return dir, nil
			}
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, no app.yao found
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no app.yao found in path hierarchy of %s", startPath)
}
