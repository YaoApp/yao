package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/setup"
	"github.com/yaoapp/yao/share"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: L("Initialize project"),
	Long:  L("Initialize a new Yao application in the current directory"),
	Run: func(cmd *cobra.Command, args []string) {
		Boot()

		// First check if we're inside an existing Yao app (including parent directories)
		if setup.InYaoApp(config.Conf.Root) {
			fmt.Println(color.YellowString(L("Directory is inside an existing Yao application")))
			fmt.Println(color.WhiteString("Please run 'yao init' outside of any Yao project"))
			os.Exit(1)
		}

		// Check if this is an empty directory
		if !setup.IsEmptyDir(config.Conf.Root) {
			// Directory is not empty
			fmt.Println(color.RedString(L("Directory is not empty")))
			fmt.Println(color.WhiteString("Please run 'yao init' in an empty directory"))
			os.Exit(1)
		}

		startTime := time.Now()
		fmt.Println(color.CyanString("Initializing Yao application..."))

		// Install the init app (copy embedded files)
		if err := setup.Install(config.Conf.Root); err != nil {
			fmt.Println(color.RedString(L("Install: %s"), err.Error()))
			os.Exit(1)
		}
		fmt.Printf("  %s %s\n", color.GreenString("✓"), "Copied application files")

		// Reload configuration after install
		Boot()

		// Load the application engine
		loadWarnings, err := engine.Load(config.Conf, engine.LoadOption{Action: "init"})
		if err != nil {
			fmt.Println(color.RedString(L("Load: %s"), err.Error()))
			os.Exit(1)
		}
		fmt.Printf("  %s %s\n", color.GreenString("✓"), "Loaded application engine")

		// Initialize (migrate + setup hook)
		if err := setup.Initialize(config.Conf.Root, config.Conf); err != nil {
			fmt.Println(color.RedString(L("Initialize: %s"), err.Error()))
			os.Exit(1)
		}
		fmt.Printf("  %s %s\n", color.GreenString("✓"), "Initialized database and data")

		initDuration := time.Since(startTime)

		// Print warnings if any
		if len(loadWarnings) > 0 {
			fmt.Println(color.YellowString("\n---------------------------------"))
			fmt.Println(color.YellowString(L("Warnings")))
			fmt.Println(color.YellowString("---------------------------------"))
			for _, warning := range loadWarnings {
				fmt.Println(color.YellowString("[%s] %s", warning.Widget, warning.Error))
			}
		}

		// Print success message
		fmt.Printf("\n%s Application initialized successfully in %s\n\n",
			color.GreenString("✓"),
			color.CyanString("%v", initDuration))

		// Print application info
		root, _ := filepath.Abs(config.Conf.Root)
		fmt.Println(color.WhiteString("---------------------------------"))
		fmt.Println(color.WhiteString(L("Application Info")))
		fmt.Println(color.WhiteString("---------------------------------"))
		fmt.Println(color.WhiteString(L("Name")), color.GreenString(" %s", share.App.Name))
		fmt.Println(color.WhiteString(L("Version")), color.GreenString(" %s", share.App.Version))
		fmt.Println(color.WhiteString(L("Root")), color.GreenString(" %s", root))

		// Print welcome message
		printInitWelcome()
	},
}

func printInitWelcome() {
	fmt.Println(color.CyanString("\n---------------------------------"))
	fmt.Println(color.CyanString(L("🎉 Application Ready 🎉")))
	fmt.Println(color.CyanString("---------------------------------"))
	fmt.Println(color.WhiteString("📚 Documentation:        "), color.CyanString("https://yaoapps.com/docs"))
	fmt.Println(color.WhiteString("🏡 Join Yao Community:   "), color.CyanString("https://yaoapps.com/community"))
	fmt.Println(color.WhiteString("🤖 Build Your Digital Workforce:"), color.CyanString("https://yaoagents.com"))
	fmt.Println("")
	fmt.Println(color.WhiteString(L("NEXT:")))
	fmt.Println(color.GreenString("  1. Edit .env to configure your application"))
	fmt.Println(color.GreenString("  2. Run 'yao start' to start the server"))
	fmt.Println("")
}

func init() {
	// Register init command
	rootCmd.AddCommand(initCmd)
}
