package sui

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	suitest "github.com/yaoapp/yao/sui/test"
)

var (
	testPage     string
	testRun      string
	testVerbose  bool
	testJSON     bool
	testFailFast bool
	testTimeout  string
)

// TestCmd runs SUI backend tests
var TestCmd = &cobra.Command{
	Use:   "test",
	Short: L("Test SUI backend scripts"),
	Long:  L("Run unit tests for SUI backend scripts (*.backend_test.ts)"),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, color.RedString(L("Usage: yao sui test <sui> [template]")))
			os.Exit(1)
		}

		Boot()

		cfg := config.Conf
		_, err := engine.Load(cfg, engine.LoadOption{Action: "sui.test"})
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			os.Exit(1)
		}

		suiID := args[0]
		template := "default"
		if len(args) >= 2 {
			template = args[1]
		}
		if suiID == "agent" && template == "default" {
			template = "agent"
		}

		timeout := 30 * time.Second
		if testTimeout != "" {
			d, err := time.ParseDuration(testTimeout)
			if err != nil {
				fmt.Fprintln(os.Stderr, color.RedString("Invalid --timeout: %s", testTimeout))
				os.Exit(1)
			}
			timeout = d
		}

		opts := &suitest.Options{
			SUIID:    suiID,
			Template: template,
			Page:     testPage,
			Run:      testRun,
			Data:     data,
			Verbose:  testVerbose,
			JSON:     testJSON,
			FailFast: testFailFast,
			Timeout:  timeout,
		}

		runner, err := suitest.NewRunner(opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			os.Exit(1)
		}

		report, err := runner.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString("Error: %s", err.Error()))
			os.Exit(1)
		}

		if report.HasFailures() {
			os.Exit(1)
		}
	},
}
