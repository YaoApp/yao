package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/application/yaz"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/pack"
)

var packOutput = ""
var packLicense = ""

var packCmd = &cobra.Command{
	Use:   "pack",
	Short: L("Package the application"),
	Long:  L("Package the application into a single file"),
	Run: func(cmd *cobra.Command, args []string) {

		cfg := config.Conf
		output, err := filepath.Abs(filepath.Join(cfg.Root, "dist"))
		if err != nil {
			color.Red(err.Error())
		}

		if packOutput != "" {
			output, err = filepath.Abs(packOutput)
			if err != nil {
				color.Red(err.Error())
			}
		}

		stat, err := os.Stat(output)
		if err != nil && os.IsNotExist(err) {
			color.Green("Creating directory %s", output)
			err = os.MkdirAll(output, 0755)
			if err != nil {
				color.Red(err.Error())
				os.Exit(1)
			}
		} else if err != nil {
			color.Red(err.Error())
			os.Exit(1)

		} else if !stat.IsDir() {
			color.Red("Output directory %s is not a directory.\n", output)
			os.Exit(1)
		}

		outputFile := filepath.Join(output, "app.yaz")
		_, err = os.Stat(outputFile)
		if !os.IsNotExist(err) {
			color.Yellow("%s already exists", outputFile)
			fmt.Printf("%s", color.RedString("Do you want to overwrite it? (y/n): "))
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				if strings.ToLower(scanner.Text()) != "y" {
					os.Exit(0)
					return
				}
			}
		}

		os.Remove(outputFile)
		if packLicense != "" {
			pack.SetCipher(packLicense)
			err = yaz.PackTo(cfg.Root, outputFile, pack.Cipher)

		} else {
			err = yaz.CompressTo(cfg.Root, outputFile)
		}

		if err != nil {
			color.Red(err.Error())
			os.Exit(1)
		}

		color.Green("Packaged to %s", outputFile)
	},
}

func init() {
	packCmd.PersistentFlags().StringVarP(&packOutput, "output", "o", "", L("Output Directory"))
	packCmd.PersistentFlags().StringVarP(&packLicense, "license", "l", "", L("Pack with the license"))
}
