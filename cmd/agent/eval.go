package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/plugin"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/eval"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	grpcclient "github.com/yaoapp/yao/grpc/client"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/share"
)

// EvalCmd is the agent eval command.
var EvalCmd = &cobra.Command{
	Use:   "eval <agent> [input]",
	Short: "Evaluate an agent with messages, test cases, or script tests",
	Long: `Evaluate an agent by sending messages, running JSONL test cases, or
executing TypeScript unit tests. Automatically bootstraps the runtime
environment (DB, V8, gRPC, Tai) — no manual setup required.

Input:
  "message"       Direct message string.
  @file.jsonl     Read test cases from file (@ prefix, like curl).
                  Supports relative and absolute paths on all platforms.

Modes:
  E2E mode        Send a message or run JSONL test cases against an agent.
  Script mode     Run *_test.ts unit tests for an agent's scripts (--scripts).

Output:
  Default         Human-friendly terminal output with colors and symbols.
  --json          NDJSON event stream for AI agents and scripts.

Identity:
  Auto-selects the first active owner from the member table.
  Override with -u/-t or provide full context via --ctx.`,
	Example: `  # Send a single message
  yao agent eval myagent "what is the weather"

  # Run JSONL test cases (@ = read from file)
  yao agent eval myagent @tests/inputs.jsonl

  # JSON output for AI consumption
  yao agent eval myagent "submit $500 expense" --json

  # Run TypeScript unit tests
  yao agent eval myagent --scripts tools

  # Override connector and user identity
  yao agent eval myagent "hello" -c deepseek.v3 -u user-001 -t team-alpha

  # Stability test: 5 rounds
  yao agent eval myagent @tests/inputs.jsonl --runs 5 --json`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runEval,
}

func init() {
	f := EvalCmd.Flags()

	// Identity & context
	f.StringP("user", "u", "", "User ID for the test session")
	f.StringP("team", "t", "", "Team ID for the test session")
	f.String("ctx", "", "Path to ContextConfig JSON (authorized, metadata, locale)")
	f.String("locale", "", "Override locale (e.g. zh-cn, en-us)")

	// Execution control
	f.StringP("connector", "c", "", "Override LLM connector")
	f.StringP("scripts", "s", "", "Script module name (enables script test mode)")
	f.Duration("timeout", 5*time.Minute, "Timeout per test case")
	f.String("run", "", "Regex filter for test case IDs or function names")
	f.Bool("fail-fast", false, "Stop on first failure")
	f.Int("parallel", 1, "Number of parallel test workers")
	f.Int("runs", 0, "Stability test rounds (0 = disabled)")
	f.Bool("dry-run", false, "List test cases without executing")
	f.String("before-all", "", "Global before script (e.g. scripts:tests.env.BeforeAll)")
	f.String("after-all", "", "Global after script (e.g. scripts:tests.env.AfterAll)")

	// Output
	f.Bool("json", false, "NDJSON output for AI agents")
	f.BoolP("verbose", "v", false, "Verbose terminal output")
	f.StringP("output", "o", "", "Write results to file (.json/.html/.md)")
	f.StringP("reporter", "r", "", "AI Reporter agent ID")

	// Environment
	f.String("tai", "", "Path to Tai binary (default: $PATH lookup)")
	f.Bool("remote", false, "Execute via remote gRPC server")
	f.String("auth", "", "Path to credentials file (default: ~/.yao/credentials)")
}

// ---------------------------------------------------------------------------
// runEval — main entry point
// ---------------------------------------------------------------------------

func runEval(cmd *cobra.Command, args []string) {
	defer share.SessionStop()
	defer plugin.KillAll()

	config.Silent = true

	// 1. Signal handling: ctx cancel on first signal, force exit on second
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		<-sigCh
		os.Exit(1)
	}()

	// 2. Parse positional args
	agentID := args[0]
	var rawInput string
	if len(args) > 1 {
		rawInput = args[1]
	}

	// 3. Read flags
	flags := cmd.Flags()
	userID, _ := flags.GetString("user")
	teamID, _ := flags.GetString("team")
	ctxFile, _ := flags.GetString("ctx")
	locale, _ := flags.GetString("locale")
	connector, _ := flags.GetString("connector")
	scripts, _ := flags.GetString("scripts")
	timeout, _ := flags.GetDuration("timeout")
	run, _ := flags.GetString("run")
	failFast, _ := flags.GetBool("fail-fast")
	parallel, _ := flags.GetInt("parallel")
	runs, _ := flags.GetInt("runs")
	dryRun, _ := flags.GetBool("dry-run")
	beforeAll, _ := flags.GetString("before-all")
	afterAll, _ := flags.GetString("after-all")
	jsonOutput, _ := flags.GetBool("json")
	verbose, _ := flags.GetBool("verbose")
	outputFile, _ := flags.GetString("output")
	reporterID, _ := flags.GetString("reporter")
	taiBin, _ := flags.GetString("tai")
	remote, _ := flags.GetBool("remote")
	authPath, _ := flags.GetString("auth")

	// 4. Build input and determine InputMode
	var inputValue string
	var inputMode eval.InputMode

	switch {
	case scripts != "":
		inputValue = "scripts." + agentID + "." + scripts
		inputMode = eval.InputModeScript
	case rawInput != "":
		resolved, isFile, err := resolveInput(rawInput)
		if err != nil {
			color.Red("Error: %s\n", err)
			os.Exit(1)
		}
		inputValue = resolved
		if isFile {
			inputMode = eval.InputModeFile
		} else {
			inputMode = eval.InputModeMessage
		}
	default:
		color.Red("Error: input argument or --scripts flag is required\n\n")
		cmd.Help()
		os.Exit(1)
	}

	// 5. Remote mode
	if remote {
		exitCode := remoteEval(ctx, authPath, &eval.Options{
			Input:       inputValue,
			InputMode:   inputMode,
			AgentID:     agentID,
			Connector:   connector,
			UserID:      userID,
			TeamID:      teamID,
			ContextFile: ctxFile,
			Locale:      locale,
			Timeout:     timeout,
			Run:         run,
			FailFast:    failFast,
			Parallel:    parallel,
			Runs:        runs,
			DryRun:      dryRun,
			BeforeAll:   beforeAll,
			AfterAll:    afterAll,
			JSONOutput:  jsonOutput,
			Verbose:     verbose,
			OutputFile:  outputFile,
			ReporterID:  reporterID,
		})
		os.Exit(exitCode)
	}

	// 6. Local mode — boot application
	if appPath == "" {
		if yaoRoot := os.Getenv("YAO_ROOT"); yaoRoot != "" {
			appPath = yaoRoot
		}
	}
	if appPath == "" {
		var err error
		cwd, _ := os.Getwd()
		appPath, err = findAppRoot(cwd)
		if err != nil {
			color.Red("Error: %s\n", err)
			color.Yellow("Hint: Run from a Yao application directory or set YAO_ROOT\n")
			os.Exit(1)
		}
	}

	Boot()

	config.Conf.Runtime.Mode = "standard"
	cfg := config.Conf
	cfg.Session.IsCLI = true

	if _, err := engine.Load(cfg, engine.LoadOption{Action: "agent-eval"}); err != nil {
		color.Red("Engine: %s\n", err)
		os.Exit(1)
	}
	if _, err := kb.Load(cfg); err != nil {
		color.Red("KB: %s\n", err)
		os.Exit(1)
	}
	if err := agent.Load(cfg); err != nil {
		color.Red("Agent: %s\n", err)
		os.Exit(1)
	}

	// 7. Default user lookup (from member table)
	if userID == "" || teamID == "" {
		u, t := lookupDefaultUser()
		if userID == "" {
			userID = u
		}
		if teamID == "" {
			teamID = t
		}
	}

	// 8. Bootstrap runtime (gRPC + Tai)
	env, err := Bootstrap(ctx, BootstrapOpts{TaiBin: taiBin, Verbose: verbose})
	defer env.Shutdown()
	if err != nil && verbose {
		color.Yellow("Bootstrap warning: %s\n", err)
	}

	// 9. Build Options and run
	opts := &eval.Options{
		Input:       inputValue,
		InputMode:   inputMode,
		AgentID:     agentID,
		Connector:   connector,
		UserID:      userID,
		TeamID:      teamID,
		ContextFile: ctxFile,
		Locale:      locale,
		Timeout:     timeout,
		Run:         run,
		FailFast:    failFast,
		Parallel:    parallel,
		Runs:        runs,
		DryRun:      dryRun,
		BeforeAll:   beforeAll,
		AfterAll:    afterAll,
		JSONOutput:  jsonOutput,
		Verbose:     verbose,
		OutputFile:  outputFile,
		ReporterID:  reporterID,
		Scripts:     scripts,
		Remote:      remote,
		TaiBin:      taiBin,
		AuthFile:    authPath,
	}

	if jsonOutput {
		opts.EventWriter = &stdoutEventWriter{}
		opts.Writer = io.Discard
	}

	if inputMode == eval.InputModeFile {
		opts.OutputFile = eval.ResolveOutputPath(opts)
	}

	runner := eval.NewRunner(opts)
	report, err := runner.Run()
	if err != nil {
		color.Red("Error: %s\n", err)
		os.Exit(1)
	}

	if report.HasFailures() {
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// Input resolution
// ---------------------------------------------------------------------------

func resolveInput(raw string) (path string, isFile bool, err error) {
	if !strings.HasPrefix(raw, "@") {
		return raw, false, nil
	}
	p := raw[1:]
	if p == "" {
		return "", false, fmt.Errorf("missing file path after @")
	}

	abs, err := filepath.Abs(p)
	if err != nil {
		return "", false, fmt.Errorf("invalid file path %q: %w", p, err)
	}

	if _, err := os.Stat(abs); err != nil {
		return "", false, fmt.Errorf("file not found: %s (resolved from %q)", abs, p)
	}
	return abs, true, nil
}

// ---------------------------------------------------------------------------
// Default user lookup
// ---------------------------------------------------------------------------

func lookupDefaultUser() (userID, teamID string) {
	defer func() {
		if userID == "" {
			userID = "test-user"
			teamID = "test-team"
			color.Yellow("Warning: no active owner found in member table, using test-user/test-team\n")
		}
	}()

	mod := model.Select("__yao.member")
	if mod == nil {
		return "", ""
	}
	tableName := mod.MetaData.Table.Name
	qb := capsule.Query()
	rows, err := qb.Table(tableName).
		Select("user_id", "team_id").
		Where("member_type", "=", "user").
		Where("is_owner", "=", true).
		Where("status", "=", "active").
		OrderBy("id").
		Limit(1).
		Get()
	if err != nil || len(rows) == 0 {
		return "", ""
	}

	row := map[string]interface{}(rows[0])
	if uid, ok := row["user_id"].(string); ok {
		userID = uid
	}
	if tid, ok := row["team_id"].(string); ok {
		teamID = tid
	}
	return userID, teamID
}

// ---------------------------------------------------------------------------
// Remote eval (gRPC delegation)
// ---------------------------------------------------------------------------

func remoteEval(ctx context.Context, authPath string, opts *eval.Options) int {
	cred := resolveCredential(authPath)
	if cred == nil {
		color.Red("Error: no credentials found. Use --auth or yao token make --save\n")
		return 1
	}
	if cred.GRPCAddr == "" {
		color.Red("Error: no gRPC address in credentials. Please re-login.\n")
		return 1
	}

	if !opts.JSONOutput {
		color.Green("Remote eval via gRPC: %s\n", cred.GRPCAddr)
	}

	optsJSON, err := jsoniter.Marshal(opts)
	if err != nil {
		color.Red("Error: failed to marshal options: %s\n", err)
		return 1
	}

	tm := grpcclient.NewTokenManager(cred.AccessToken, cred.RefreshToken, "")
	client, err := grpcclient.Dial(cred.GRPCAddr, tm)
	if err != nil {
		color.Red("Error: gRPC connect failed: %s\n", err)
		return 1
	}
	defer client.Close()

	var reportJSON []byte
	err = client.Stream(ctx, "agent.eval.Run", optsJSON, 0, func(data []byte, done bool) error {
		if done {
			reportJSON = data
			return nil
		}
		os.Stdout.Write(data)
		return nil
	})
	if err != nil {
		color.Red("Error: eval execution failed: %s\n", err)
		return 1
	}

	if len(reportJSON) > 0 {
		var report eval.Report
		if json.Unmarshal(reportJSON, &report) == nil {
			if report.Error != "" {
				color.Red("Error: %s\n", report.Error)
				return 1
			}
			if report.HasFailures() {
				return 1
			}
		}
	}
	return 0
}

// ---------------------------------------------------------------------------
// Credential helpers
// ---------------------------------------------------------------------------

type evalCredential struct {
	Server       string `json:"server"`
	GRPCAddr     string `json:"grpc_addr,omitempty"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func resolveCredential(authPath string) *evalCredential {
	if authPath != "" {
		cred, err := loadCredential(authPath)
		if err != nil {
			color.Red("Error: failed to load credentials: %s\n", err)
			os.Exit(1)
		}
		return cred
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(home, ".yao", "credentials")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	cred, _ := loadCredential(path)
	return cred
}

func loadCredential(path string) (*evalCredential, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("decode credentials: %w", err)
	}
	var cred evalCredential
	if err := json.Unmarshal(decoded, &cred); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}
	return &cred, nil
}

// ---------------------------------------------------------------------------
// App root discovery (reused from original test.go)
// ---------------------------------------------------------------------------

func findAppRoot(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("path not found: %s", absPath)
	}

	var dir string
	if info.IsDir() {
		dir = absPath
	} else {
		dir = filepath.Dir(absPath)
	}

	for {
		for _, appFile := range []string{"app.yao", "app.json", "app.jsonc"} {
			appFilePath := filepath.Join(dir, appFile)
			if _, err := os.Stat(appFilePath); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no app.yao found in path hierarchy of %s", startPath)
}

// stdoutEventWriter writes NDJSON event lines to stdout for local --json mode.
type stdoutEventWriter struct{}

func (w *stdoutEventWriter) WriteEvent(data []byte) error {
	line := make([]byte, len(data)+1)
	copy(line, data)
	line[len(data)] = '\n'
	_, err := os.Stdout.Write(line)
	return err
}
