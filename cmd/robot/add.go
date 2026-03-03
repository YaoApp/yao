package robot

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/registry"
	robotmgr "github.com/yaoapp/yao/registry/manager/robot"
)

// AddCmd implements "yao robot add @scope/name --team TEAM_ID"
var AddCmd = &cobra.Command{
	Use:   "add [package]",
	Short: L("Install a robot package from the registry"),
	Long:  L("Install a robot and its dependencies. Example: yao robot add @yao/keeper --team team-123"),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Boot()
		pkgID := args[0]
		version, _ := cmd.Flags().GetString("version")
		teamID, _ := cmd.Flags().GetString("team")

		client := registry.New(config.Conf.Registry,
			registry.WithAuth(
				os.Getenv("YAO_REGISTRY_USER"),
				os.Getenv("YAO_REGISTRY_PASS"),
			),
		)

		mgr := robotmgr.New(client, config.Conf.Root, nil)
		robot, err := mgr.Add(pkgID, robotmgr.AddOptions{
			Version: version,
			TeamID:  teamID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// The actual member record creation requires database access.
		// In P0 we print the robot config for the CLI layer to handle.
		fmt.Printf("Robot ready: %s (display_name: %s)\n", pkgID, robot.DisplayName)
		fmt.Println("Note: Member record must be created via Mission Control or database.")
	},
}

func init() {
	AddCmd.Flags().StringP("version", "v", "latest", L("Package version or dist-tag"))
	AddCmd.Flags().StringP("team", "t", "", L("Team ID (required)"))
	AddCmd.PersistentFlags().StringVarP(&appPath, "app", "a", "", L("Application directory"))
	AddCmd.PersistentFlags().StringVarP(&envFile, "env", "e", "", L("Environment file"))
}
