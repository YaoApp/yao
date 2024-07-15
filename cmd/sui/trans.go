package sui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/sui/core"
	"golang.org/x/text/language"
)

// TransCmd command
var TransCmd = &cobra.Command{
	Use:   "trans",
	Short: L("Translate the template"),
	Long:  L("Translate the template"),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, color.RedString(L("yao sui trans <sui> <template> [data]")))
			return
		}

		Boot()

		cfg := config.Conf
		err := engine.Load(cfg, engine.LoadOption{Action: "sui.trans"})
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
		localeRoot := filepath.Join(tmpl.GetRoot(), "__locales")
		definedLocales := tmpl.Locales()
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

		fmt.Println("")
		fmt.Println(color.GreenString("Language packs:"))
		fmt.Println(color.WhiteString("-----------------------"))
		for _, locale := range definedLocales {
			if locale.Default {
				continue
			}
			path := filepath.Join(localeRoot, locale.Value)
			fmt.Println(color.WhiteString("  %s:\t%s", locale.Label, path))
		}
		fmt.Println(color.WhiteString("-----------------------"))
		fmt.Println("")

		// Timecost
		start := time.Now()
		minify := true
		mode := "production"
		if debug {
			minify = false
			mode = "development"
		}

		option := core.BuildOption{SSR: true, AssetRoot: assetRoot, ExecScripts: true, ScriptMinify: minify, StyleMinify: minify}

		// locales filter
		if locales != "" {
			fmt.Println("")
			fmt.Println(color.GreenString("Translate locales:"))
			fmt.Println(color.WhiteString("-----------------------"))
			localeList := strings.Split(locales, ",")
			option.Locales = []string{}
			for _, locale := range localeList {
				locale = strings.ToLower(strings.TrimSpace(locale))
				label := language.Make(locale).String()
				option.Locales = append(option.Locales, locale)
				path := filepath.Join(localeRoot, locale)
				fmt.Println(color.WhiteString("  %s:\t%s", label, path))
			}
			fmt.Println(color.WhiteString("-----------------------"))
			fmt.Println("")
		}

		warnings, err := tmpl.Trans(&option)
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			return
		}
		end := time.Now()
		timecost := end.Sub(start).Truncate(time.Millisecond)
		if debug {
			fmt.Println(color.YellowString("Translate succeeded for %s in %s", mode, timecost))
			return
		}
		if len(warnings) > 0 {
			for _, warning := range warnings {
				fmt.Println(color.YellowString("Warning: %s", warning))
			}
		}

		fmt.Println(color.GreenString("Translate succeeded for %s in %s", mode, timecost))

		// build the template
		fmt.Println("Start building the template")
		start = time.Now()
		warnings, err = tmpl.Build(&option)
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString(err.Error()))
			return
		}

		end = time.Now()
		timecost = end.Sub(start).Truncate(time.Millisecond)
		if debug {
			fmt.Println(color.YellowString("Build succeeded for %s in %s", mode, timecost))
			return
		}
		if len(warnings) > 0 {
			for _, warning := range warnings {
				fmt.Println(color.YellowString("Warning: %s", warning))
			}
		}
		fmt.Println(color.GreenString("Build succeeded for %s in %s", mode, timecost))
	},
}
