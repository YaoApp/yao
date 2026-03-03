package agent

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/registry"
	agentmgr "github.com/yaoapp/yao/registry/manager/agent"
)

// PushCmd implements "yao agent push scope.name --version x.y.z"
var PushCmd = &cobra.Command{
	Use:   "push [yao-id]",
	Short: L("Push an assistant package to the registry"),
	Long:  L("Package and push an assistant to the registry. Example: yao agent push max.keeper --version 1.0.0"),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Boot()

		yaoID := args[0]
		version, _ := cmd.Flags().GetString("version")
		force, _ := cmd.Flags().GetBool("force")

		client := registry.New(config.Conf.Registry,
			registry.WithAuth(
				os.Getenv("YAO_REGISTRY_USER"),
				os.Getenv("YAO_REGISTRY_PASS"),
			),
		)

		mgr := agentmgr.New(client, config.Conf.Root, nil)
		if err := mgr.Push(yaoID, agentmgr.PushOptions{
			Version: version,
			Force:   force,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	PushCmd.Flags().StringP("version", "v", "", L("Package version (required)"))
	PushCmd.Flags().Bool("force", false, L("Overwrite existing version"))
	PushCmd.PersistentFlags().StringVarP(&appPath, "app", "a", "", L("Application directory"))
	PushCmd.PersistentFlags().StringVarP(&envFile, "env", "e", "", L("Environment file"))
}
