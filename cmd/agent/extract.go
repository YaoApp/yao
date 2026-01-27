package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/agent/test"
)

// Extract command flags
var (
	extractOutput string
	extractFormat string
)

// ExtractCmd is the agent extract command
var ExtractCmd = &cobra.Command{
	Use:   "extract <output-file.jsonl>",
	Short: L("Extract test results to individual files for review"),
	Long:  L("Extract test results from output JSONL file to individual Markdown or JSON files"),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]

		// Resolve absolute path
		absPath, err := filepath.Abs(inputFile)
		if err != nil {
			color.Red("Error: %s\n", err.Error())
			os.Exit(1)
		}

		// Check if file exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			color.Red("Error: file not found: %s\n", absPath)
			os.Exit(1)
		}

		// Build extract options
		opts := &test.ExtractOptions{
			InputFile: absPath,
			OutputDir: extractOutput,
			Format:    extractFormat,
		}

		// Create extractor and run
		extractor := test.NewExtractor(opts)
		files, err := extractor.Extract()
		if err != nil {
			color.Red("Error: %s\n", err.Error())
			os.Exit(1)
		}

		// Print results
		fmt.Println()
		color.New(color.FgGreen, color.Bold).Println("═══════════════════════════════════════════════════════════════")
		color.New(color.FgGreen, color.Bold).Println("  Extract Complete")
		color.New(color.FgGreen, color.Bold).Println("═══════════════════════════════════════════════════════════════")
		fmt.Println()

		for _, file := range files {
			color.New(color.FgGreen).Printf("✓ ")
			fmt.Printf("Written: %s\n", filepath.Base(file))
		}

		fmt.Println()
		color.New(color.FgWhite).Printf("  Total: ")
		color.New(color.FgCyan).Printf("%d files\n", len(files))

		if extractOutput != "" {
			color.New(color.FgWhite).Printf("  Output: ")
			color.New(color.FgCyan).Printf("%s\n", extractOutput)
		} else {
			color.New(color.FgWhite).Printf("  Output: ")
			color.New(color.FgCyan).Printf("%s\n", filepath.Dir(absPath))
		}
		fmt.Println()
	},
}

func init() {
	// Extract command flags
	ExtractCmd.Flags().StringVarP(&extractOutput, "output", "o", "", L("Output directory (default: same as input file)"))
	ExtractCmd.Flags().StringVar(&extractFormat, "format", "markdown", L("Output format: markdown, json"))
}
