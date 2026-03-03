package agent

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/registry"
	agentmgr "github.com/yaoapp/yao/registry/manager/agent"
)

var agentAddForce bool

// AddCmd implements "yao agent add @scope/name"
var AddCmd = &cobra.Command{
	Use:   "add [package]",
	Short: L("Install an assistant package from the registry"),
	Long:  L("Install an assistant package from the registry. Example: yao agent add @yao/keeper"),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Boot()

		pkgID := args[0]
		version, _ := cmd.Flags().GetString("version")

		client := registry.New(config.Conf.Registry,
			registry.WithAuth(
				os.Getenv("YAO_REGISTRY_USER"),
				os.Getenv("YAO_REGISTRY_PASS"),
			),
		)

		mgr := agentmgr.New(client, config.Conf.Root, nil)
		if err := mgr.Add(pkgID, agentmgr.AddOptions{
			Version: version,
			Force:   agentAddForce,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	AddCmd.Flags().StringP("version", "v", "latest", L("Package version or dist-tag"))
	AddCmd.Flags().BoolVarP(&agentAddForce, "force", "", false, L("Force reinstall"))
	AddCmd.PersistentFlags().StringVarP(&appPath, "app", "a", "", L("Application directory"))
	AddCmd.PersistentFlags().StringVarP(&envFile, "env", "e", "", L("Environment file"))
}
