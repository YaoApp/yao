package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/plugin"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/test"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	grpcclient "github.com/yaoapp/yao/grpc/client"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/share"
)

// Test command flags
var (
	testInput     string
	testOutput    string
	testAgent     string
	testConnector string
	testUser      string
	testTeam      string
	testContext   string // --ctx flag for custom context JSON file
	testReporter  string
	testRuns      int
	testRun       string // --run flag for test filtering (regex pattern)
	testTimeout   string
	testParallel  int
	testVerbose   bool
	testFailFast  bool
	testBefore    string // --before flag for global BeforeAll hook
	testAfter     string // --after flag for global AfterAll hook
	testDryRun    bool   // --dry-run flag for generating tests without running
	testSimulator string // --simulator flag for default simulator agent in dynamic mode
	testJSON      bool   // --json flag for machine-readable JSON output
	testScripts   string // --scripts flag for script unit test module
	testAuthPath  string // --auth flag for credential file
)

// TestCmd is the agent test command
var TestCmd = &cobra.Command{
	Use:   "test",
	Short: L("Test an agent with input cases"),
	Long: L(`Test an agent with input cases from JSONL file or direct message.

IMPORTANT: Run this command from the Yao application root directory (where app.yao is located).
  Always use -n <agent_id> to specify the agent explicitly, to avoid ambiguity.

Modes:
  Direct message (-n <agent> -i 'message'):
    Send a single message to the agent and print the response.
    The -n flag specifies the agent ID (e.g. myagent, folder.subagent).

  File mode (-n <agent> -i file.jsonl):
    Run test cases from a JSONL file with assertions and reporting.

  Script test (-n <agent> --scripts <module>):
    Run unit tests defined in the agent's src/*_test.ts files.
    -n specifies the agent, --scripts specifies the module name.
    This discovers and executes all Test* functions in the corresponding _test.ts file.
    Each test function receives (t: testing.T, ctx: agent.Context).
    Use --run <regex> to filter which Test* functions to run.

    Examples:
      -n myagent --scripts tools           -> assistants/myagent/src/tools_test.ts
      -n myagent.sub --scripts seed        -> assistants/myagent/sub/src/seed_test.ts

    Legacy syntax (also supported):
      -i scripts.myagent.tools             -> same as -n myagent --scripts tools

Common flags:
  -c, --connector <id>  Override the LLM connector for this test run. The connector ID
            corresponds to a connector defined in the application (e.g. gpt4o, claude, deepseek).
            If not specified, the agent's default connector is used.
            This is useful for testing the same agent against different models.

  -u, --user <id>       Set the user ID for the test context (default: "test-user").
            The agent sees this as the current user identity. Useful for testing
            permission-related logic or user-specific behavior.

  -t, --team <id>       Set the team ID for the test context (default: "test-team").
            The agent sees this as the current team. Useful for testing
            team-scoped data access or multi-tenant logic.

  --ctx <file.json>     Provide a full context JSON file for fine-grained control over the
            test session. Allows setting authorization details (sub, scope, client_id,
            session_id, constraints), metadata, client info, and locale.
            -u/-t are convenient shortcuts for user_id/team_id only.
            When both are provided, --ctx authorized.user_id/team_id take precedence
            over -u/-t for the authorization layer.
            JSON structure:
              {
                "chat_id": "session-1",
                "authorized": {
                  "user_id": "admin", "team_id": "ops",
                  "sub": "jwt-sub", "client_id": "app-1",
                  "scope": "full", "session_id": "sess-1",
                  "constraints": { "owner_only": true, "team_only": true }
                },
                "metadata": { "key": "value" },
                "locale": "en-us"
              }

Remote execution:
  When credentials are available (via --auth <path> or ~/.yao/credentials),
  tests are executed on the remote Yao server via gRPC with real-time progress
  streaming. This matches the 'yao run' gRPC mode.

AI Integration (recommended flags):
  --json    Output full JSON with trace diagnostics (completion details, all MCP tool calls
            with server/arguments/results/errors, and Next hook data). Use this when an AI
            agent needs to analyze test results programmatically.

  Console output is automatically silenced ([robot:*] suppressed) in test mode.

  Example (AI debugging a single agent call):
    yao agent test -n myagent -i 'what is the weather in Shanghai' --json

  Example (AI with specific connector):
    yao agent test -n myagent -i 'hello' -c gpt4o --json

  Example (AI running test suite):
    yao agent test -n myagent -i tests/myagent.jsonl -o results.json --json

  Example (AI running script unit tests):
    yao agent test -n myagent --scripts tools --json
    yao agent test -n myagent --scripts tools --run TestRecognize --json

  Example (human: E2E):
    yao agent test -n myagent -i 'hello' -v

  Example (human: script test):
    yao agent test -n myagent --scripts tools
    yao agent test -n myagent --scripts tools --run TestRecognize

Output formats for file mode (-o flag extension):
  .jsonl    JSONL streaming events (default, includes trace on each result)
  .json     Full JSON report (includes trace on each result)
  .md       Markdown report (failed cases expand trace: tool call table + completion summary)
  .html     HTML report (failed cases have collapsible trace details)`),
	Run: func(cmd *cobra.Command, args []string) {
		defer share.SessionStop()
		defer plugin.KillAll()

		// Suppress [robot:*] and other noisy console output during tests
		config.Silent = true

		// --scripts mode: combine -n <agent> --scripts <module> into scripts.<agent>.<module>
		if testScripts != "" {
			if testAgent == "" {
				color.Red(L("Error: -n <agent> is required when using --scripts") + "\n\n")
				cmd.Help()
				os.Exit(1)
			}
			testInput = "scripts." + testAgent + "." + testScripts
			testAgent = ""
		}

		// Validate input
		if testInput == "" {
			color.Red(L("Error: input (-i) or --scripts flag is required") + "\n\n")
			cmd.Help()
			os.Exit(1)
		}

		// Detect input mode
		inputMode := test.DetectInputMode(testInput)

		// For message mode, agent must be specified or resolvable from cwd
		if inputMode == test.InputModeMessage && testAgent == "" {
			cwd, err := os.Getwd()
			if err != nil {
				color.Red(L("Error: failed to get current directory")+": %s\n", err.Error())
				os.Exit(1)
			}
			resolver := test.NewResolver()
			_, err = resolver.ResolveFromPath(cwd)
			if err != nil {
				color.Red(L("Error: agent (-n) is required when using direct message input and not in an agent directory") + "\n")
				os.Exit(1)
			}
		}

		// gRPC delegation: when credentials are available, delegate to a running Yao server.
		cred := resolveTestCredential()
		if cred != nil {
			exitCode := testGRPC(cred, inputMode)
			os.Exit(exitCode)
		}

		// No credentials: local execution only.
		if testVerbose {
			color.Yellow("No credentials found — running in local mode. Remote runners (tai/claude/opencode) require:\n")
			color.Yellow("  1. A running Yao server: yao start\n")
			color.Yellow("  2. Login or generate credentials: yao token make --member <user_id> --team <team_id> --save\n\n")
		}

		// Find app root directory
		var err error

		if appPath == "" {
			if yaoRoot := os.Getenv("YAO_ROOT"); yaoRoot != "" {
				appPath = yaoRoot
			}
		}

		if appPath == "" {
			if inputMode == test.InputModeFile {
				appPath, err = findAppRoot(testInput)
			} else {
				cwd, _ := os.Getwd()
				appPath, err = findAppRoot(cwd)
			}
			if err != nil {
				color.Red("Error: %s\n", err.Error())
				color.Yellow(L("Hint: Make sure you're in a Yao application directory or specify --app flag") + "\n")
				os.Exit(1)
			}
		}

		// Boot the application
		Boot()

		// Set Runtime Mode
		config.Conf.Runtime.Mode = "standard"
		cfg := config.Conf
		cfg.Session.IsCLI = true

		// Load engine
		_, err = engine.Load(cfg, engine.LoadOption{Action: "agent-test"})
		if err != nil {
			color.Red("Engine: %s\n", err.Error())
			os.Exit(1)
		}

		// Load KB (required for agent KB features)
		_, err = kb.Load(cfg)
		if err != nil {
			color.Red("KB: %s\n", err.Error())
			os.Exit(1)
		}

		// Load agent
		err = agent.Load(cfg)
		if err != nil {
			color.Red("Agent: %s\n", err.Error())
			os.Exit(1)
		}

		// Parse timeout
		timeout := 5 * time.Minute
		if testTimeout != "" {
			d, err := time.ParseDuration(testTimeout)
			if err != nil {
				color.Red(L("Error: invalid timeout format")+": %s\n", testTimeout)
				os.Exit(1)
			}
			timeout = d
		}

		// Build test options
		opts := &test.Options{
			Input:       testInput,
			InputMode:   inputMode,
			OutputFile:  testOutput,
			AgentID:     testAgent,
			Connector:   testConnector,
			UserID:      testUser,
			TeamID:      testTeam,
			ContextFile: testContext,
			ReporterID:  testReporter,
			Runs:        testRuns,
			Run:         testRun,
			Timeout:     timeout,
			Parallel:    testParallel,
			Verbose:     testVerbose,
			FailFast:    testFailFast,
			BeforeAll:   testBefore,
			AfterAll:    testAfter,
			DryRun:      testDryRun,
			Simulator:   testSimulator,
			JSONOutput:  testJSON,
		}

		// Merge with defaults
		opts = test.MergeOptions(opts, test.DefaultOptions())

		// Resolve output path (only for file mode, direct message mode outputs to stdout)
		if inputMode == test.InputModeFile {
			opts.OutputFile = test.ResolveOutputPath(opts)
		}

		// Run tests
		runner := test.NewRunner(opts)
		report, err := runner.Run()
		if err != nil {
			color.Red("Error: %s\n", err.Error())
			os.Exit(1)
		}

		// Exit with appropriate code
		if report.HasFailures() {
			os.Exit(1)
		}
	},
}

// testCredential mirrors cmd.Credential for use within the cmd/agent package.
type testCredential struct {
	Server       string `json:"server"`
	GRPCAddr     string `json:"grpc_addr,omitempty"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// resolveTestCredential loads credentials from --auth flag or default path.
func resolveTestCredential() *testCredential {
	if testAuthPath != "" {
		cred, err := loadTestCredential(testAuthPath)
		if err != nil {
			color.Red("  %s %s\n", L("Failed to load credentials:"), err)
			os.Exit(1)
		}
		return cred
	}
	cred, _ := loadTestCredentialDefault()
	return cred
}

func loadTestCredentialDefault() (*testCredential, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".yao", "credentials")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	return loadTestCredential(path)
}

func loadTestCredential(path string) (*testCredential, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("decode credentials: %w", err)
	}
	var cred testCredential
	if err := json.Unmarshal(decoded, &cred); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}
	return &cred, nil
}

// testGRPC delegates the test execution to a remote Yao server via gRPC Stream.
// It streams real-time progress output and receives the final *test.Report.
func testGRPC(cred *testCredential, inputMode test.InputMode) int {
	if cred.GRPCAddr == "" {
		color.Red("  %s\n", L("No gRPC address in credentials. Please re-login."))
		return 1
	}

	if !testJSON {
		color.Green(L("Remote test via gRPC: %s\n"), cred.GRPCAddr)
	}

	// Parse timeout
	timeout := 5 * time.Minute
	if testTimeout != "" {
		if d, err := time.ParseDuration(testTimeout); err == nil {
			timeout = d
		}
	}

	// Build options to send to server
	opts := &test.Options{
		Input:       testInput,
		InputMode:   inputMode,
		OutputFile:  testOutput,
		AgentID:     testAgent,
		Connector:   testConnector,
		UserID:      testUser,
		TeamID:      testTeam,
		ContextFile: testContext,
		ReporterID:  testReporter,
		Runs:        testRuns,
		Run:         testRun,
		Timeout:     timeout,
		Parallel:    testParallel,
		Verbose:     testVerbose,
		FailFast:    testFailFast,
		BeforeAll:   testBefore,
		AfterAll:    testAfter,
		DryRun:      testDryRun,
		Simulator:   testSimulator,
		JSONOutput:  testJSON,
	}

	optsJSON, err := jsoniter.Marshal(opts)
	if err != nil {
		color.Red("  %s %s\n", L("Failed to marshal options:"), err.Error())
		return 1
	}

	tm := grpcclient.NewTokenManager(cred.AccessToken, cred.RefreshToken, "")
	client, err := grpcclient.Dial(cred.GRPCAddr, tm)
	if err != nil {
		color.Red("  %s %s\n", L("gRPC connect failed:"), err.Error())
		return 1
	}
	defer client.Close()

	var reportJSON []byte
	err = client.Stream(context.Background(), "agent.test.Run", optsJSON, 0, func(data []byte, done bool) error {
		if done {
			reportJSON = data
			return nil
		}
		// Real-time progress: write server output directly to stdout
		os.Stdout.Write(data)
		return nil
	})
	if err != nil {
		color.Red("  %s %s\n", L("Test execution failed:"), err.Error())
		return 1
	}

	// Parse report to determine exit code
	if len(reportJSON) > 0 {
		var report test.Report
		if json.Unmarshal(reportJSON, &report) == nil {
			if report.Error != "" {
				color.Red("  %s %s\n", L("Error:"), report.Error)
				return 1
			}
			if report.HasFailures() {
				return 1
			}
		}
	}

	return 0
}

// findAppRoot finds the Yao application root directory by looking for app.yao
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

func init() {
	// Test command flags
	TestCmd.Flags().StringVarP(&appPath, "app", "a", "", L("Application directory"))
	TestCmd.Flags().StringVarP(&envFile, "env", "e", "", L("Environment file"))
	TestCmd.Flags().StringVarP(&testInput, "input", "i", "", L("Input: JSONL file path or message (required)"))
	TestCmd.Flags().StringVarP(&testOutput, "output", "o", "", L("Path to output file (default: output-{timestamp}.jsonl)"))
	TestCmd.Flags().StringVarP(&testAgent, "name", "n", "", L("Explicit agent ID (default: auto-detect)"))
	TestCmd.Flags().StringVarP(&testConnector, "connector", "c", "", L("Override connector"))
	TestCmd.Flags().StringVarP(&testUser, "user", "u", "", L("Test user ID (default: test-user)"))
	TestCmd.Flags().StringVarP(&testTeam, "team", "t", "", L("Test team ID (default: test-team)"))
	TestCmd.Flags().StringVar(&testContext, "ctx", "", L("Path to context JSON file for custom authorization"))
	TestCmd.Flags().StringVarP(&testReporter, "reporter", "r", "", L("Reporter agent ID for custom report"))
	TestCmd.Flags().IntVar(&testRuns, "runs", 1, L("Number of runs for stability analysis"))
	TestCmd.Flags().StringVar(&testRun, "run", "", L("Regex pattern to filter which tests to run"))
	TestCmd.Flags().StringVar(&testTimeout, "timeout", "5m", L("Default timeout per test case"))
	TestCmd.Flags().IntVar(&testParallel, "parallel", 1, L("Number of parallel test cases"))
	TestCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, L("Verbose output"))
	TestCmd.Flags().BoolVar(&testFailFast, "fail-fast", false, L("Stop on first failure"))
	TestCmd.Flags().StringVar(&testBefore, "before", "", L("Global BeforeAll hook (e.g., env_test.BeforeAll)"))
	TestCmd.Flags().StringVar(&testAfter, "after", "", L("Global AfterAll hook (e.g., env_test.AfterAll)"))
	TestCmd.Flags().BoolVar(&testDryRun, "dry-run", false, L("Generate test cases without running them"))
	TestCmd.Flags().StringVar(&testSimulator, "simulator", "", L("Default simulator agent for dynamic mode (e.g., tests.simulator-agent)"))
	TestCmd.Flags().BoolVar(&testJSON, "json", false, L("Output full JSON with trace diagnostics: completion, MCP tool calls (server/args/result/error), Next hook (recommended for AI)"))
	TestCmd.Flags().StringVar(&testScripts, "scripts", "", L("Script module to test (use with -n): -n expense --scripts tools → runs assistants/expense/src/tools_test.ts"))
	TestCmd.Flags().StringVar(&testAuthPath, "auth", "", L("Path to credentials file for gRPC remote execution"))
}
