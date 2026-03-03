package agent

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/registry"
	agentmgr "github.com/yaoapp/yao/registry/manager/agent"
)

// UpdateCmd implements "yao agent update @scope/name"
var UpdateCmd = &cobra.Command{
	Use:   "update [package]",
	Short: L("Update an installed assistant package"),
	Long:  L("Update an installed assistant to a newer version. Example: yao agent update @yao/keeper"),
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
		if err := mgr.Update(pkgID, agentmgr.UpdateOptions{
			Version: version,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	UpdateCmd.Flags().StringP("version", "v", "latest", L("Target version or dist-tag"))
	UpdateCmd.PersistentFlags().StringVarP(&appPath, "app", "a", "", L("Application directory"))
	UpdateCmd.PersistentFlags().StringVarP(&envFile, "env", "e", "", L("Environment file"))
}
