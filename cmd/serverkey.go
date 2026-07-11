package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/setting"
)

var serverKeyName string
var serverKeyTTL string

var serverKeyCmd = &cobra.Command{
	Use:   "server-key",
	Short: "Server Key management for cloud Tai nodes",
	Long:  "Create, list, and revoke server keys (yao-sk: prefix) for cloud Tai node authentication",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var serverKeyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new server key",
	Long:  "Create a new server key for cloud Tai node authentication. The plaintext key is shown once.",
	Run: func(cmd *cobra.Command, args []string) {
		Boot()
		cfg := config.Conf
		engine.Load(cfg, engine.LoadOption{})

		if serverKeyName == "" {
			serverKeyName = "cloud-node"
		}

		var opts []time.Duration
		if serverKeyTTL != "" {
			d, err := time.ParseDuration(serverKeyTTL)
			if err != nil {
				color.Red("Invalid TTL %q: %v", serverKeyTTL, err)
				os.Exit(1)
			}
			opts = append(opts, d)
		}

		plainKey, keyID, err := setting.CreateServerKey(serverKeyName, opts...)
		if err != nil {
			color.Red("Failed to create server key: %v", err)
			os.Exit(1)
		}
		setting.Global.Flush()

		fmt.Println()
		color.Green("  Server key created successfully")
		fmt.Println()
		color.White("  Key ID:  %s", keyID)
		color.White("  Name:    %s", serverKeyName)
		if serverKeyTTL != "" {
			color.White("  TTL:     %s", serverKeyTTL)
		}
		color.Yellow("  Key:     %s", plainKey)
		fmt.Println()
		color.Red("  ⚠ Save this key now. It will not be shown again.")
		fmt.Println()
	},
}

var serverKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all server keys",
	Run: func(cmd *cobra.Command, args []string) {
		Boot()
		cfg := config.Conf
		engine.Load(cfg, engine.LoadOption{})

		keys, err := setting.ListServerKeys()
		if err != nil {
			color.Red("Failed to list server keys: %v", err)
			os.Exit(1)
		}

		if len(keys) == 0 {
			color.White("  No server keys found")
			return
		}

		fmt.Println()
		for _, k := range keys {
			status := color.GreenString("active")
			if k.Revoked {
				status = color.RedString("revoked")
			}
			color.White("  %s  %s  [%s]  created: %s", k.ID, k.Name, status, k.CreatedAt)
			if k.ExpiresAt != "" {
				color.White("    expires: %s", k.ExpiresAt)
			}
			if k.LastUsed != "" {
				color.White("    last used: %s", k.LastUsed)
			}
		}
		fmt.Println()
	},
}

var serverKeyRevokeCmd = &cobra.Command{
	Use:   "revoke <key-id>",
	Short: "Revoke a server key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Boot()
		cfg := config.Conf
		engine.Load(cfg, engine.LoadOption{})

		keyID := args[0]
		if err := setting.RevokeServerKey(keyID); err != nil {
			color.Red("Failed to revoke server key: %v", err)
			os.Exit(1)
		}
		setting.Global.Flush()

		color.Green("  Server key %s revoked", keyID)
	},
}

func init() {
	serverKeyCreateCmd.Flags().StringVar(&serverKeyName, "name", "", "Name/label for the server key")
	serverKeyCreateCmd.Flags().StringVar(&serverKeyTTL, "ttl", "", "Time-to-live (e.g. 720h for 30 days; empty = never expires)")
	serverKeyCmd.AddCommand(serverKeyCreateCmd)
	serverKeyCmd.AddCommand(serverKeyListCmd)
	serverKeyCmd.AddCommand(serverKeyRevokeCmd)
}
