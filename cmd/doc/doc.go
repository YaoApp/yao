package doc

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/yaoapp/gou/doc"
)

var jsonOutput bool

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func formatArgs(args []doc.TypeValue) string {
	if len(args) == 0 {
		return "()"
	}
	parts := make([]string, 0, len(args))
	for _, a := range args {
		s := a.Name + " " + a.Type
		if !a.Required {
			s += "?"
		}
		parts = append(parts, s)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func formatReturn(ret *doc.TypeValue) string {
	if ret == nil {
		return "void"
	}
	return ret.Type
}

func init() {
	ProcessCmd.AddCommand(processListCmd)
	ProcessCmd.AddCommand(processValidateCmd)
	ProcessCmd.AddCommand(processInspectCmd)
	RuntimeCmd.AddCommand(runtimeListCmd)
	RuntimeCmd.AddCommand(runtimeValidateCmd)
	RuntimeCmd.AddCommand(runtimeInspectCmd)

	processListCmd.Flags().StringVarP(&groupFlag, "group", "g", "", "Filter by group")
	processListCmd.Flags().StringVarP(&searchFlag, "search", "s", "", "Search keyword")
	processListCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	processListCmd.Flags().BoolVar(&showAll, "all", false, "Include undocumented processes")

	processValidateCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	processInspectCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	runtimeListCmd.Flags().StringVarP(&typeFlag, "type", "t", "", "Filter by type: object, function, class")
	runtimeListCmd.Flags().StringVarP(&searchFlag, "search", "s", "", "Search keyword")
	runtimeListCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	runtimeValidateCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	runtimeInspectCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
}

var groupFlag string
var searchFlag string
var typeFlag string
var showAll bool

func errExit(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
