package sui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/sui/core"
)

// BuildCmd command
var BuildCmd = &cobra.Command{
	Use:   "build",
	Short: L("Build the template"),
	Long:  L("Build the template"),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, color.RedString(L("yao cui build <sui> <template> [data]")))
			return
		}

		Boot()

		cfg := config.Conf
		err := engine.Load(cfg, engine.LoadOption{Action: "sui.build"})
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			return
		}

		id := args[0]
		template := args[1]

		var sessionData map[string]interface{}
		err = jsoniter.UnmarshalFromString(strings.TrimPrefix(data, "::"), &sessionData)
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			return
		}

		sid := uuid.New().String()
		if sessionData != nil && len(sessionData) > 0 {
			session.Global().ID(sid).SetMany(sessionData)
		}

		sui, has := core.SUIs[id]
		if !has {
			fmt.Fprintf(os.Stderr, color.RedString(("the sui " + id + " does not exist")))
			return
		}
		sui.WithSid(sid)

		tmpl, err := sui.GetTemplate(template)
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			return
		}

		// -
		publicRoot, err := sui.PublicRootWithSid(sid)
		assetRoot := filepath.Join(publicRoot, "assets")
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			return
		}

		fmt.Println(color.WhiteString("-----------------------"))
		fmt.Println(color.WhiteString("Public Root: /public%s", publicRoot))
		fmt.Println(color.WhiteString("   Template: %s", tmpl.GetRoot()))
		fmt.Println(color.WhiteString("    Session: %s", strings.TrimLeft(data, "::")))
		fmt.Println(color.WhiteString("-----------------------"))

		err = tmpl.Build(&core.BuildOption{SSR: true, AssetRoot: assetRoot})
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			return
		}

		fmt.Print(color.GreenString("Build Success\n"))
	},
}
