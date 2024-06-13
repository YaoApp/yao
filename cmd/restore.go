package cmd

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
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
	"github.com/yaoapp/yao/share"
)

var restoreForce bool = false
var migrateNoInsert bool = false
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: L("Restore the application data"),
	Long:  L("Restore the application data"),
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			err := exception.Catch(recover())
			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			}
		}()

		if len(args) < 1 {
			fmt.Println(color.RedString(L("Not enough arguments")))
			fmt.Println(color.WhiteString(share.BUILDNAME + " help"))
			os.Exit(1)
		}

		zipfile, err := filepath.Abs(args[0])
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		Boot()

		if !restoreForce && config.Conf.Mode == "production" {
			fmt.Println(color.WhiteString(L("TRY:")), color.GreenString("%s restore --force", share.BUILDNAME))
			exception.New(L("Retore is not allowed on production mode."), 403).Throw()
		}

		// Unzip files
		dst := unzipFile(zipfile, func(file string) {
			fmt.Printf("\r%s", strings.Repeat(" ", 80))
			fmt.Printf("\r%s", color.GreenString(L("Unzip the file: %s"), file))
		})

		// 加载数据模型
		err = engine.Load(config.Conf, engine.LoadOption{Action: "restore"})
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		// Restore models
		restoreModels(filepath.Join(dst, "model"), []model.MigrateOption{
			model.WithDonotInsertValues(migrateNoInsert),
		})

		// Restore Data
		restoreData(filepath.Join(dst, "data"))

		// Clean
		os.RemoveAll(dst)

		fmt.Println(color.GreenString(L("✨DONE✨")))
	},
}

func init() {
	restoreCmd.PersistentFlags().BoolVarP(&restoreForce, "force", "", false, L("Force restore"))
	restoreCmd.PersistentFlags().BoolVarP(&migrateNoInsert, "migrate-no-insert", "", false, L("Do not insert values when migrating"))
}

func restoreData(basePath string) {

	_, err := os.Stat(basePath)
	if err != nil {
		fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		os.Exit(1)
	}

	// Clean Data
	dataPath := filepath.Join(config.Conf.Root, "data")
	_, err = os.Stat(dataPath)
	if err == nil {
		os.RemoveAll(dataPath)
	}

	err = os.Rename(basePath, dataPath)
	if err != nil {
		fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		os.Exit(1)
	}
}

func restoreModels(basePath string, migOpts []model.MigrateOption) {

	files, err := ioutil.ReadDir(basePath)
	if err != nil {
		fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		os.Exit(1)
	}

	// Migrate models
	for _, mod := range model.Models {
		fmt.Printf("\r%s", strings.Repeat(" ", 80))
		fmt.Printf(color.GreenString(L("\rUpdate schema model: %s (%s) "), mod.Name, mod.MetaData.Table.Name))
		err := mod.Migrate(true, migOpts...)
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}
	}

	fmt.Println("")

	for _, file := range files {
		namer := strings.Split(file.Name(), ".")
		name := strings.Join(namer[:len(namer)-2], ".")
		if mod, has := model.Models[name]; has {
			fmt.Printf("\r%s", strings.Repeat(" ", 80))
			fmt.Printf(color.GreenString(L("\rRestore model: %s (%s) "), mod.Name, mod.MetaData.Table.Name))
			err := mod.Import(filepath.Join(basePath, file.Name()))
			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
				os.Exit(1)
			}
		}
	}
}

func unzipFile(file string, process func(file string)) string {
	_, err := os.Stat(file)

	if errors.Is(err, os.ErrNotExist) {
		fmt.Println(color.RedString("%s not exists", file))
		os.Exit(1)
	}

	if err != nil {
		fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		os.Exit(1)
	}

	dst := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s", filepath.Base(file), time.Now().Format("20060102150405")))
	os.MkdirAll(dst, 0755)

	archive, err := zip.OpenReader(file)
	if err != nil {
		fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		os.Exit(1)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(dst, f.Name)
		process(f.Name)

		if !strings.HasPrefix(filePath, filepath.Clean(dst)+string(os.PathSeparator)) {
			fmt.Println(color.RedString(L("Fatal: invalid file path")))
			os.Exit(1)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		dstFile.Close()
		fileInArchive.Close()
	}

	return dst
}
