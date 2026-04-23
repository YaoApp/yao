package doc

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/doc"
)

// RuntimeCmd is the parent command for runtime API documentation.
var RuntimeCmd = &cobra.Command{
	Use:   "runtime",
	Short: "JS runtime API documentation",
	Long:  "List, inspect and validate JS runtime global objects, functions, and classes",
	Run:   func(cmd *cobra.Command, args []string) { cmd.Help() },
}

var runtimeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List JS runtime APIs",
	Long: `List all documented JS global objects, functions, and classes.
Designed for grep/pipe usage:
  yao doc runtime list | grep FS
  yao doc runtime list --type class`,
	Run: func(cmd *cobra.Command, args []string) {
		types := runtimeTypes()
		var allEntries []*doc.Entry
		for _, t := range types {
			opts := doc.ListOption{Search: searchFlag}
			allEntries = append(allEntries, doc.List(t, opts)...)
		}

		if jsonOutput {
			printJSON(allEntries)
			return
		}

		for _, e := range allEntries {
			tag := string(e.Type)
			extra := ""
			if len(e.Methods) > 0 {
				extra = fmt.Sprintf("  (%d methods)", len(e.Methods))
			}
			fmt.Printf("%-20s [%-11s] %s%s\n", e.Name, tag, e.Desc, extra)
		}
	},
}

var runtimeInspectCmd = &cobra.Command{
	Use:   "inspect [name]",
	Short: "Show detailed info for a JS runtime API",
	Long: `Show full documentation for a JS global object, function, or class.
  yao doc runtime inspect FS
  yao doc runtime inspect log
  yao doc runtime inspect Process`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		entry := findRuntime(name)
		if entry == nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Runtime API %q not found.\n", name)
			for _, t := range []doc.EntryType{doc.TypeJSObject, doc.TypeJSClass, doc.TypeJSFunction} {
				r := doc.Validate(t, name)
				if len(r.Suggestion) > 0 {
					fmt.Fprintln(cmd.ErrOrStderr(), "Did you mean:")
					for _, s := range r.Suggestion {
						fmt.Fprintf(cmd.ErrOrStderr(), "  - %s\n", s)
					}
					break
				}
			}
			return
		}

		if jsonOutput {
			printJSON(entry)
			return
		}

		printRuntimeDetail(entry)
	},
}

var runtimeValidateCmd = &cobra.Command{
	Use:   "validate [name]",
	Short: "Validate a JS runtime API name",
	Long:  "Check if a JS global name is documented (e.g. FS, log, Process)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		var result *doc.ValidationResult
		for _, t := range []doc.EntryType{doc.TypeJSObject, doc.TypeJSClass, doc.TypeJSFunction} {
			r := doc.Validate(t, name)
			if r.Valid {
				result = r
				break
			}
			if result == nil || len(r.Suggestion) > len(result.Suggestion) {
				result = r
			}
		}

		if jsonOutput {
			printJSON(result)
			return
		}

		if result.Valid {
			fmt.Printf("✓ %s — %s (type: %s)\n", result.Name, result.Message, result.Entry.Type)
		} else {
			fmt.Printf("✗ %s — %s (status: %s)\n", result.Name, result.Message, result.Status)
			if len(result.Suggestion) > 0 {
				fmt.Println("  Did you mean:")
				for _, s := range result.Suggestion {
					fmt.Printf("    - %s\n", s)
				}
			}
		}
	},
}

func findRuntime(name string) *doc.Entry {
	for _, t := range []doc.EntryType{doc.TypeJSObject, doc.TypeJSClass, doc.TypeJSFunction} {
		e, ok := doc.Get(t, name)
		if ok {
			return e
		}
	}
	return nil
}

func printRuntimeDetail(e *doc.Entry) {
	fmt.Printf("Name: %s\n", e.Name)
	fmt.Printf("Type: %s\n", e.Type)
	fmt.Printf("Desc: %s\n", e.Desc)

	if len(e.Args) > 0 {
		fmt.Println()
		fmt.Println("Constructor Arguments:")
		for i, a := range e.Args {
			req := "required"
			if !a.Required {
				req = "optional"
			}
			fmt.Printf("  [%d] %s  (%s, %s)\n", i, a.Name, a.Type, req)
			if a.Desc != "" {
				fmt.Printf("      %s\n", a.Desc)
			}
		}
	}

	if e.Return != nil {
		fmt.Println()
		fmt.Println("Returns:")
		printTypeValue(e.Return, 2)
	}

	if len(e.Methods) > 0 {
		fmt.Println()
		fmt.Printf("Methods (%d):\n", len(e.Methods))
		for _, m := range e.Methods {
			ret := formatReturn(m.Return)
			fmt.Printf("  .%-25s %s → %s\n", m.Name+formatArgs(m.Args), m.Desc, ret)
			if m.Return != nil && len(m.Return.Fields) > 0 {
				printTypeFields(m.Return.Fields, 4)
			}
		}
	}
}

func runtimeTypes() []doc.EntryType {
	switch typeFlag {
	case "object":
		return []doc.EntryType{doc.TypeJSObject}
	case "function":
		return []doc.EntryType{doc.TypeJSFunction}
	case "class":
		return []doc.EntryType{doc.TypeJSClass}
	default:
		return []doc.EntryType{doc.TypeJSObject, doc.TypeJSClass, doc.TypeJSFunction}
	}
}
