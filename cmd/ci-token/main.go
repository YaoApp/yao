//go:build ci

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/openapi/oauth"
)

func main() {
	appPath := flag.String("app", envOr("YAO_CI_APP_PATH", "."), "Yao application directory")
	clientID := flag.String("client-id", envOr("YAO_CI_OAUTH_CLIENT_ID", "ci-tai"), "OAuth client ID embedded in token")
	subject := flag.String("subject", envOr("YAO_CI_OAUTH_SUBJECT", "ci-tai"), "JWT subject claim")
	scope := flag.String("scope", envOr("YAO_CI_OAUTH_SCOPE", "tai:tunnel"), "Token scope (space-separated)")
	ttl := flag.String("ttl", envOr("YAO_CI_OAUTH_TTL", "24h"), "Token TTL (e.g. 1h, 24h, 168h)")
	userID := flag.String("user-id", envOr("YAO_CI_OAUTH_USER_ID", ""), "User ID claim")
	teamID := flag.String("team-id", envOr("YAO_CI_OAUTH_TEAM_ID", ""), "Team ID claim")
	flag.Parse()

	root, err := filepath.Abs(*appPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ci-token: invalid app path: %v\n", err)
		os.Exit(1)
	}

	if err := os.Chdir(root); err != nil {
		fmt.Fprintf(os.Stderr, "ci-token: chdir %s: %v\n", root, err)
		os.Exit(1)
	}

	savedStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)

	config.Conf = config.LoadFrom(filepath.Join(root, ".env"))
	config.Conf.Root = root

	cfg := config.Conf
	cfg.Session.IsCLI = true

	warnings, err := engine.Load(cfg, engine.LoadOption{Action: "run"})
	os.Stdout = savedStdout
	if err != nil {
		fmt.Fprintf(os.Stderr, "ci-token: engine.Load failed: %v\n", err)
		os.Exit(1)
	}
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "ci-token: warning [%s]: %v\n", w.Widget, w.Error)
	}

	if oauth.OAuth == nil {
		fmt.Fprintln(os.Stderr, "ci-token: oauth service not initialized (openapi.Load may have failed)")
		os.Exit(1)
	}

	dur, err := time.ParseDuration(*ttl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ci-token: invalid --ttl %q: %v\n", *ttl, err)
		os.Exit(1)
	}
	expiresIn := int(dur.Seconds())

	extraClaims := map[string]interface{}{}
	if *userID != "" {
		extraClaims["user_id"] = *userID
	}
	if *teamID != "" {
		extraClaims["team_id"] = *teamID
	}

	token, err := oauth.OAuth.MakeAccessToken(*clientID, *scope, *subject, expiresIn, extraClaims)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ci-token: MakeAccessToken failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(token)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
