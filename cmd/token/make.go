package token

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/openapi/oauth"
)

var (
	makeTeamID   string
	makeMemberID string
	makeExpires  int
	makeSave     bool
	makeSavePath string
)

// MakeCmd generates an OAuth access token for a team member.
var MakeCmd = &cobra.Command{
	Use:   "make",
	Short: "Generate an OAuth access token",
	Long:  "Generate an OAuth access token for yao agent test or API access",
	Run:   runMake,
}

func init() {
	MakeCmd.Flags().StringVar(&makeTeamID, "team", "", "Team ID (required)")
	MakeCmd.Flags().StringVar(&makeMemberID, "member", "", "User ID - business user identifier, not email (required)")
	MakeCmd.Flags().IntVar(&makeExpires, "expires", 86400, "Token lifetime in seconds (default 24h)")
	MakeCmd.Flags().BoolVar(&makeSave, "save", false, "Save token to file instead of stdout")
	MakeCmd.Flags().StringVar(&makeSavePath, "save-path", "", "Path to save token (default ~/.yao/credentials)")
}

func runMake(cmd *cobra.Command, args []string) {
	if makeTeamID == "" || makeMemberID == "" {
		color.Red("--team and --member are required")
		os.Exit(1)
	}

	cfg := config.Conf
	cfg.Mode = "production"
	engine.Load(cfg, engine.LoadOption{})

	if oauth.OAuth == nil {
		color.Red("OAuth service not initialized")
		os.Exit(1)
	}

	claims := map[string]interface{}{
		"team_id":   makeTeamID,
		"member_id": makeMemberID,
	}

	const gRPCScopes = "grpc:run grpc:stream grpc:shell grpc:mcp grpc:llm grpc:agent"
	token, err := oauth.OAuth.MakeAccessToken(
		"yao-cli",
		gRPCScopes,
		makeMemberID,
		makeExpires,
		claims,
	)
	if err != nil {
		color.Red("Failed to generate token: %v", err)
		os.Exit(1)
	}

	if makeSave {
		savePath := makeSavePath
		if savePath == "" {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				color.Red("Failed to determine home directory: %v", homeErr)
				os.Exit(1)
			}
			savePath = filepath.Join(home, ".yao", "credentials")
		}

		grpcAddr := fmt.Sprintf("%s:%d", config.Conf.GRPC.Host, config.Conf.GRPC.Port)
		if config.Conf.GRPC.Host == "internal" || config.Conf.GRPC.Host == "" {
			grpcAddr = fmt.Sprintf("127.0.0.1:%d", config.Conf.GRPC.Port)
		}

		cred := map[string]string{
			"grpc_addr":    grpcAddr,
			"access_token": token,
		}
		credJSON, _ := json.Marshal(cred)
		encoded := base64.StdEncoding.EncodeToString(credJSON)

		dir := filepath.Dir(savePath)
		if err := os.MkdirAll(dir, 0700); err != nil {
			color.Red("Failed to create directory %s: %v", dir, err)
			os.Exit(1)
		}

		if err := os.WriteFile(savePath, []byte(encoded), 0600); err != nil {
			color.Red("Failed to write token to %s: %v", savePath, err)
			os.Exit(1)
		}

		color.Green("Token saved to %s", savePath)
		return
	}

	fmt.Println(token)
}
