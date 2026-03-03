package mcp

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/registry"
	mcpmgr "github.com/yaoapp/yao/registry/manager/mcp"
)

// ForkCmd implements "yao mcp fork @scope/name [@target-scope]"
var ForkCmd = &cobra.Command{
	Use:   "fork [package] [target-scope]",
	Short: L("Fork an MCP to a local scope"),
	Long:  L("Fork an MCP for local modification. Example: yao mcp fork @yao/rag-tools"),
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		Boot()
		pkgID := args[0]
		var targetScope string
		if len(args) > 1 {
			targetScope = args[1]
		}

		client := registry.New(config.Conf.Registry,
			registry.WithAuth(
				os.Getenv("YAO_REGISTRY_USER"),
				os.Getenv("YAO_REGISTRY_PASS"),
			),
		)

		mgr := mcpmgr.New(client, config.Conf.Root, nil)
		if err := mgr.Fork(pkgID, mcpmgr.ForkOptions{
			TargetScope: targetScope,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	ForkCmd.PersistentFlags().StringVarP(&appPath, "app", "a", "", L("Application directory"))
	ForkCmd.PersistentFlags().StringVarP(&envFile, "env", "e", "", L("Environment file"))
}
