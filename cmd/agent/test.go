package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/plugin"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/test"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
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
)

// TestCmd is the agent test command
var TestCmd = &cobra.Command{
	Use:   "test",
	Short: L("Test an agent with input cases"),
	Long:  L("Test an agent with input cases from JSONL file or direct message"),
	Run: func(cmd *cobra.Command, args []string) {
		defer share.SessionStop()
		defer plugin.KillAll()

		// Validate input
		if testInput == "" {
			color.Red(L("Error: input is required (-i flag)") + "\n")
			os.Exit(1)
		}

		// Detect input mode
		inputMode := test.DetectInputMode(testInput)

		// For message mode, agent must be specified or resolvable from cwd
		if inputMode == test.InputModeMessage && testAgent == "" {
			// Try to find app root from current directory
			cwd, err := os.Getwd()
			if err != nil {
				color.Red(L("Error: failed to get current directory")+": %s\n", err.Error())
				os.Exit(1)
			}

			// Try to find package.yao from cwd
			resolver := test.NewResolver()
			_, err = resolver.ResolveFromPath(cwd)
			if err != nil {
				color.Red(L("Error: agent (-n) is required when using direct message input and not in an agent directory") + "\n")
				os.Exit(1)
			}
		}

		// Find app root directory
		// Priority: -a flag > YAO_ROOT env > auto-detect from path
		var err error

		if appPath == "" {
			// Check YAO_ROOT environment variable
			if yaoRoot := os.Getenv("YAO_ROOT"); yaoRoot != "" {
				appPath = yaoRoot
			}
		}

		if appPath == "" {
			// Auto-detect from path
			if inputMode == test.InputModeFile {
				// For file mode, find app root from input file path
				appPath, err = findAppRoot(testInput)
			} else {
				// For message mode, find app root from current directory
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

// findAppRoot finds the Yao application root directory by looking for app.yao
// It traverses up from the given path until it finds app.yao or reaches the filesystem root
func findAppRoot(startPath string) (string, error) {
	// Get absolute path
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// If it's a file, start from its directory
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

	// Traverse up to find app.yao
	for {
		// Check for app.yao, app.json, or app.jsonc
		for _, appFile := range []string{"app.yao", "app.json", "app.jsonc"} {
			appFilePath := filepath.Join(dir, appFile)
			if _, err := os.Stat(appFilePath); err == nil {
				return dir, nil
			}
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, no app.yao found
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

	// Mark input as required
	TestCmd.MarkFlagRequired("input")
}
