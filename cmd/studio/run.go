package studio

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/plugin"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/studio"
)

// RunCmd command
var RunCmd = &cobra.Command{
	Use:   "run",
	Short: L("Execute Yao Studio Script"),
	Long:  L("Execute Yao Studio Script"),
	Run: func(cmd *cobra.Command, args []string) {
		defer share.SessionStop()
		defer plugin.KillAll()
		defer func() {
			err := exception.Catch(recover())
			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			}
		}()

		Boot()
		cfg := config.Conf
		cfg.Session.IsCLI = true

		if len(args) < 1 {
			fmt.Println(color.RedString(L("Not enough arguments")))
			fmt.Println(color.WhiteString(share.BUILDNAME + " help"))
			return
		}

		err := engine.Load(cfg, engine.LoadOption{Action: "studio.run"})
		if err != nil {
			fmt.Println(color.RedString(L("Engine: %s"), err.Error()))
		}

		err = studio.Load(cfg)
		if err != nil {
			fmt.Println(color.RedString(L("Studio: %s"), err.Error()))
		}

		name := strings.Split(args[0], ".")
		service := strings.Join(name[0:len(name)-1], ".")
		method := name[len(name)-1]

		fmt.Println(color.GreenString(L("Studio Run: %s"), args[0]))
		pargs := []interface{}{}
		for i, arg := range args {
			if i == 0 {
				continue
			}

			if strings.HasPrefix(arg, "::") {
				arg := strings.TrimPrefix(arg, "::")
				var v interface{}
				err := jsoniter.Unmarshal([]byte(arg), &v)
				if err != nil {
					fmt.Println(color.RedString(L("Arguments: %s"), err.Error()))
					return
				}
				pargs = append(pargs, v)
				fmt.Println(color.WhiteString("args[%d]: %s", i-1, arg))
			} else if strings.HasPrefix(arg, "\\::") {
				arg := "::" + strings.TrimPrefix(arg, "\\::")
				pargs = append(pargs, arg)
				fmt.Println(color.WhiteString("args[%d]: %s", i-1, arg))
			} else {
				pargs = append(pargs, arg)
				fmt.Println(color.WhiteString("args[%d]: %s", i-1, arg))
			}
		}

		script, err := v8.SelectRoot(service)
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		}

		sid := uuid.New().String()
		global := map[string]interface{}{}
		ctx, err := script.NewContext(sid, global)
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
		}
		defer ctx.Close()

		res, err := ctx.Call(method, pargs...)
		if err != nil {
			fmt.Println(color.RedString("--------------------------------------"))
			fmt.Println(color.RedString(L("%s Error"), args[0]))
			fmt.Println(color.RedString("--------------------------------------"))
			utils.Dump(err)
			fmt.Println(color.RedString("--------------------------------------"))
			fmt.Println(color.GreenString(L("✨DONE✨")))
			return
		}

		fmt.Println(color.WhiteString("--------------------------------------"))
		fmt.Println(color.WhiteString(L("%s Response"), args[0]))
		fmt.Println(color.WhiteString("--------------------------------------"))
		utils.Dump(res)
		fmt.Println(color.WhiteString("--------------------------------------"))
		fmt.Println(color.GreenString(L("✨DONE✨")))
	},
}
