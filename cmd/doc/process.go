package doc

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/doc"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/schedule"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/gou/task"
)

// ProcessCmd is the parent command for process documentation.
var ProcessCmd = &cobra.Command{
	Use:   "process",
	Short: "Process documentation",
	Long:  "List, inspect and validate Yao process documentation",
	Run:   func(cmd *cobra.Command, args []string) { cmd.Help() },
}

var processListCmd = &cobra.Command{
	Use:   "list",
	Short: "List documented processes",
	Long: `List all documented processes in flat format (one per line).
Designed for grep/pipe usage:
  yao doc process list | grep model
  yao doc process list --group http`,
	Run: func(cmd *cobra.Command, args []string) {
		opts := doc.ListOption{Group: groupFlag, Search: searchFlag}
		entries := doc.List(doc.TypeProcess, opts)

		if jsonOutput {
			printJSON(entries)
			return
		}

		for _, e := range entries {
			fmt.Printf("%-45s %s\n", doc.CallableName(e), e.Desc)
		}

		if showAll {
			undoc := doc.AutoDiscover()
			if len(undoc) > 0 {
				fmt.Printf("\nUNDOCUMENTED: %d processes\n", len(undoc))
				for _, name := range undoc {
					fmt.Printf("%-45s (no documentation)\n", name)
				}
			}
		}
	},
}

var processInspectCmd = &cobra.Command{
	Use:   "inspect [process-name]",
	Short: "Show detailed info for a process",
	Long: `Show full documentation for a single process, including arguments,
return type, field descriptions, and examples.
  yao doc process inspect http.get
  yao doc process inspect models.user.Find`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		e, ok := doc.Get(doc.TypeProcess, name)
		if !ok {
			fmt.Fprintf(cmd.ErrOrStderr(), "Process %q not found.\n", name)
			result := doc.Validate(doc.TypeProcess, name)
			if len(result.Suggestion) > 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "Did you mean:")
				for _, s := range result.Suggestion {
					fmt.Fprintf(cmd.ErrOrStderr(), "  - %s\n", s)
				}
			}
			return
		}

		if jsonOutput {
			printJSON(e)
			return
		}

		printProcessDetail(e)
	},
}

var processValidateCmd = &cobra.Command{
	Use:   "validate [process-name]",
	Short: "Validate a process name",
	Long: `Validate a process call using the engine's addressing logic,
check that the referenced resource exists, and show documentation.

Examples:
  yao doc process validate models.user.Find
  yao doc process validate http.get
  yao doc process validate stores.cache.set`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		p, err := process.Of(name)
		if err != nil {
			printError(cmd, name, err.Error())
			return
		}

		if process.Handlers[p.Handler] == nil {
			printError(cmd, name, fmt.Sprintf("handler %q not registered", p.Handler))
			return
		}

		if reason := checkID(p); reason != "" {
			printError(cmd, name, reason)
			return
		}

		docResult := doc.Validate(doc.TypeProcess, name)

		if jsonOutput {
			out := map[string]interface{}{
				"name":       name,
				"valid":      true,
				"handler":    p.Handler,
				"group":      p.Group,
				"id":         p.ID,
				"method":     p.Method,
				"documented": docResult.Valid,
			}
			if docResult.Entry != nil {
				out["entry"] = docResult.Entry
			}
			printJSON(out)
			return
		}

		fmt.Printf("✓ %s — valid\n", name)
		fmt.Printf("  Handler: %s\n", p.Handler)
		fmt.Printf("  Group:   %s\n", p.Group)
		if p.ID != "" {
			fmt.Printf("  ID:      %s\n", p.ID)
		}
		fmt.Printf("  Method:  %s\n", p.Method)
		fmt.Println()

		if docResult.Valid && docResult.Entry != nil {
			printProcessDetail(docResult.Entry)
		} else {
			fmt.Printf("⚠ no documentation found for %s\n", name)
		}
	},
}

// checkID validates the <id> segment for dynamic-ID groups by looking up the
// actual runtime registry. Returns an error message or "" if OK.
func checkID(p *process.Process) string {
	switch p.Group {
	case "models":
		if p.ID == "" {
			return fmt.Sprintf("missing <id>, correct format: models.<id>.%s", p.Method)
		}
		if !model.Exists(p.ID) {
			return fmt.Sprintf("model %q not loaded", p.ID)
		}

	case "schemas":
		if p.ID == "" {
			return fmt.Sprintf("missing <id>, correct format: schemas.<id>.%s", p.Method)
		}

	case "stores":
		if p.ID == "" {
			return fmt.Sprintf("missing <id>, correct format: stores.<id>.%s", p.Method)
		}
		if _, has := store.Pools[p.ID]; !has {
			return fmt.Sprintf("store %q not loaded", p.ID)
		}

	case "fs":
		if p.ID == "" {
			return fmt.Sprintf("missing <id>, correct format: fs.<id>.%s", p.Method)
		}
		if _, has := fs.FileSystems[p.ID]; !has {
			return fmt.Sprintf("filesystem %q not registered", p.ID)
		}

	case "tasks":
		if p.ID == "" {
			return fmt.Sprintf("missing <id>, correct format: tasks.<id>.%s", p.Method)
		}
		if _, has := task.Tasks[p.ID]; !has {
			return fmt.Sprintf("task %q not loaded", p.ID)
		}

	case "schedules":
		if p.ID == "" {
			return fmt.Sprintf("missing <id>, correct format: schedules.<id>.%s", p.Method)
		}
		if _, has := schedule.Schedules[p.ID]; !has {
			return fmt.Sprintf("schedule %q not loaded", p.ID)
		}
	}
	return ""
}

func printError(cmd *cobra.Command, name, msg string) {
	if jsonOutput {
		printJSON(map[string]interface{}{
			"name":  name,
			"valid": false,
			"error": msg,
		})
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "✗ %s — %s\n", name, msg)
}

func printProcessDetail(e *doc.Entry) {
	fmt.Printf("Process: %s\n", doc.CallableName(e))
	fmt.Printf("Group:   %s\n", e.Group)
	fmt.Printf("Desc:    %s\n", e.Desc)
	fmt.Println()

	if len(e.Args) > 0 {
		fmt.Println("Arguments:")
		for i, a := range e.Args {
			req := "required"
			if !a.Required {
				req = "optional"
			}
			fmt.Printf("  [%d] %s  (%s, %s)\n", i, a.Name, a.Type, req)
			if a.Desc != "" {
				fmt.Printf("      %s\n", a.Desc)
			}
			if a.Example != nil {
				fmt.Printf("      Example: %v\n", a.Example)
			}
			printTypeFields(a.Fields, 6)
		}
		fmt.Println()
	}

	fmt.Println("Returns:")
	if e.Return == nil {
		fmt.Println("  void")
	} else {
		printTypeValue(e.Return, 2)
	}
}

func printTypeValue(tv *doc.TypeValue, indent int) {
	pad := strings.Repeat(" ", indent)
	fmt.Printf("%s%s", pad, tv.Type)
	if tv.Desc != "" {
		fmt.Printf(" — %s", tv.Desc)
	}
	fmt.Println()
	if tv.Example != nil {
		fmt.Printf("%sExample: %v\n", pad, tv.Example)
	}

	if len(tv.Fields) > 0 {
		printTypeFields(tv.Fields, indent)
	}
	if tv.Items != nil {
		fmt.Printf("%sItems:\n", pad)
		printTypeValue(tv.Items, indent+2)
	}
	if len(tv.Variants) > 0 {
		for i, v := range tv.Variants {
			fmt.Printf("%sVariant %d: ", pad, i+1)
			desc := v.Type
			if v.Desc != "" {
				desc += " — " + v.Desc
			}
			fmt.Println(desc)
		}
	}
}

func printTypeFields(fields []doc.TypeValue, indent int) {
	if len(fields) == 0 {
		return
	}
	pad := strings.Repeat(" ", indent)
	fmt.Printf("%sFields:\n", pad)
	for _, f := range fields {
		req := ""
		if f.Required {
			req = ", required"
		}
		fmt.Printf("%s  .%-20s %s%s\n", pad, f.Name, f.Type, req)
		if f.Desc != "" {
			fmt.Printf("%s    %s\n", pad, f.Desc)
		}
	}
}
