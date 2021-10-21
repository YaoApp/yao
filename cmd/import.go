package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/engine"
)

var importRenew bool
var importChunk int
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "导入数据(alpha)",
	Long:  `导入CSV/Excel数据`,
	Run: func(cmd *cobra.Command, args []string) {
		Boot()

		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "参数错误", args)
			fmt.Println("xiang import <数据模型> <文件名称> <导入配置>")
			os.Exit(1)
		}

		name := args[0]
		filename := args[1]
		configfile := args[2]

		// 检查数据文件
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			fmt.Println(color.RedString("数据文件不存在: %s", filename))
			os.Exit(1)
		}

		if _, err := os.Stat(configfile); os.IsNotExist(err) {
			fmt.Println(color.RedString("配置文件不存在: %s", configfile))
			os.Exit(1)
		}

		content, err := ioutil.ReadFile(configfile)
		if err != nil {
			fmt.Println(color.RedString("读取配置文件失败: %s", err))
			os.Exit(1)
		}

		impt := gou.NewImport(importChunk)
		err = jsoniter.Unmarshal(content, impt)
		if err != nil {
			fmt.Println(color.RedString("解析配置文件=失败: %s", err))
			os.Exit(1)
		}

		// 加载数据模型
		engine.Load(config.Conf)
		if importRenew {
			gou.Select(name).Migrate(true)
		}

		resp := impt.InsertCSV(filename, name)
		for _, err := range *resp.Errors {
			fmt.Println(color.RedString("%s line: %d", err.Message, err.Line))
		}
		fmt.Println(color.WhiteString("成功: %d, 失败: %d, 总共: %d", resp.Success, resp.Failure, resp.Total))

	},
}

func init() {
	importCmd.PersistentFlags().BoolVarP(&importRenew, "renew", "r", false, "清空数据模型原有数据")
	importCmd.PersistentFlags().IntVarP(&importChunk, "chunk", "c", 1000, "每次处理数量(数值越大,处理越快，内存占用越大)")
}
