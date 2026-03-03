package agent

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/registry"
	agentmgr "github.com/yaoapp/yao/registry/manager/agent"
)

// ForkCmd implements "yao agent fork @scope/name [@target-scope]"
var ForkCmd = &cobra.Command{
	Use:   "fork [package] [target-scope]",
	Short: L("Fork an assistant to a local scope"),
	Long:  L("Fork an assistant for local modification. Example: yao agent fork @yao/keeper"),
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

		mgr := agentmgr.New(client, config.Conf.Root, nil)
		if err := mgr.Fork(pkgID, agentmgr.ForkOptions{
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
