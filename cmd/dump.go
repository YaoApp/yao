package cmd

import (
	"archive/zip"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
)

var dumpModel string
var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: L("Dump the application data"),
	Long:  L("Dump the application data"),
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			err := exception.Catch(recover())
			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			}
		}()

		Boot()

		path, err := filepath.Abs(".")
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		output := filepath.Join(fmt.Sprintf("%s-%s.zip", filepath.Base(path), time.Now().Format("20060102150405")))
		if len(args) > 0 {
			output = args[0]
		}

		output, err = filepath.Abs(output)
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		_, err = os.Stat(output)
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Println(color.RedString("%s exists", output))
			os.Exit(1)
		}

		// Load model
		err = engine.Load(config.Conf, engine.LoadOption{Action: "dump"})
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		if dumpModel != "" {
			fmt.Println(color.YellowString(L("Not supported yet")))
			os.Exit(1)
			return
		}

		// Export models
		files := []string{}
		for _, mod := range model.Models {

			fmt.Printf("\r%s", color.GreenString(L("Export the models: %s (%s)"), mod.Name, mod.MetaData.Table.Name))
			jsonfiles, err := mod.Export(5000, func(curr, total int) {
				fmt.Printf("\r%s", strings.Repeat(" ", 80))
				fmt.Printf("\r%s", color.GreenString(L("Export the models: %s (%s) %d/%d"), mod.Name, mod.MetaData.Table.Name, curr, total))
			})

			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
				os.Exit(1)
			}
			files = append(files, jsonfiles...)
		}
		fmt.Printf("\r%s", strings.Repeat(" ", 80))
		fmt.Printf("\r%s\n", color.GreenString(L("Export the models: ✨DONE✨")))

		// Compress files
		err = zipfiles(files, output, func(file string) {
			fmt.Printf("\r%s", strings.Repeat(" ", 80))
			fmt.Printf("\r%s", color.GreenString(L("Compress the files: %s"), file))
		})
		fmt.Printf("\r%s", strings.Repeat(" ", 80))
		fmt.Printf("\r%s\n", color.GreenString(L("Compress the files: ✨DONE✨")))

		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		fmt.Println(color.GreenString("File: %s", output))
	},
}

// func init() {
// 	// dumpCmd.PersistentFlags().StringVarP(&dumpModel, "name", "n", "", L("Model name"))
// }

// gzipfiles
func zipfiles(files []string, output string, process func(file string)) error {
	outpath := filepath.Dir(output)
	os.MkdirAll(outpath, 0755)

	outfile, err := os.Create(output)
	if err != nil {
		fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		os.Exit(1)
	}
	defer outfile.Close()

	w := zip.NewWriter(outfile)
	defer func() {
		w.Close()
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}
	}()

	for _, file := range files {
		addFile(w, file, "model", process)
	}

	// Add data path
	dataPath := filepath.Join(config.Conf.Root, "data")
	_, err = os.Stat(dataPath)
	if err == nil {
		addFolder(w, dataPath, "data", process)
	}

	return nil
}

func addFile(w *zip.Writer, file, baseInZip string, process func(file string)) {
	process(filepath.Join(baseInZip, filepath.Base(file)))
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		os.Exit(1)
	}

	// Add some files to the archive.
	f, err := w.Create(filepath.Join(baseInZip, filepath.Base(file)))
	if err != nil {
		fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		os.Exit(1)
	}
	_, err = f.Write(dat)
	if err != nil {
		fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		os.Exit(1)
	}

	os.Remove(file)
}

func addFolder(w *zip.Writer, basePath, baseInZip string, process func(file string)) {

	// Open the Directory
	files, err := ioutil.ReadDir(basePath)
	if err != nil {
		fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		os.Exit(1)
	}

	for _, file := range files {
		process(filepath.Join(baseInZip, file.Name()))
		if !file.IsDir() {
			dat, err := ioutil.ReadFile(filepath.Join(basePath, file.Name()))
			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
				os.Exit(1)
			}

			// Add some files to the archive.
			f, err := w.Create(filepath.Join(baseInZip, file.Name()))
			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
				os.Exit(1)
			}
			_, err = f.Write(dat)
			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
				os.Exit(1)
			}
		} else if file.IsDir() {
			// Recurse
			newBase := filepath.Join(basePath, file.Name())
			addFolder(w, newBase, filepath.Join(baseInZip, file.Name()), process)
		}
	}
}
