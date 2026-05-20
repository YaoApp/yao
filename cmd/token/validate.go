package token

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/openapi/oauth"
)

var validateFile string

// ValidateCmd validates an OAuth access token and prints its claims.
var ValidateCmd = &cobra.Command{
	Use:   "validate [token]",
	Short: "Validate an OAuth access token",
	Long:  "Validate an OAuth access token and print its claims",
	Args:  cobra.MaximumNArgs(1),
	Run:   runValidate,
}

func init() {
	ValidateCmd.Flags().StringVar(&validateFile, "file", "", "Read token from file")
}

func runValidate(cmd *cobra.Command, args []string) {
	var tokenStr string
	if len(args) > 0 {
		tokenStr = args[0]
	} else if validateFile != "" {
		data, err := os.ReadFile(validateFile)
		if err != nil {
			color.Red("Failed to read token file %s: %v", validateFile, err)
			os.Exit(1)
		}
		tokenStr = strings.TrimSpace(string(data))
	} else {
		color.Red("Provide a token as argument or use --file <path>")
		os.Exit(1)
	}

	if tokenStr == "" {
		color.Red("Token is empty")
		os.Exit(1)
	}

	cfg := config.Conf
	cfg.Mode = "production"
	engine.Load(cfg, engine.LoadOption{})

	if oauth.OAuth == nil {
		color.Red("OAuth service not initialized")
		os.Exit(1)
	}

	claims, err := oauth.OAuth.VerifyToken(tokenStr)
	if err != nil {
		color.Red("Invalid token: %v", err)
		os.Exit(1)
	}

	data, _ := json.MarshalIndent(claims, "", "  ")
	fmt.Println(string(data))
}
