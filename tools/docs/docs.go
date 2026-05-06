package docs

import (
	_ "embed"

	goudoc "github.com/yaoapp/gou/doc"
	"github.com/yaoapp/gou/process"
)

//go:embed list.json
var ListSchemaJSON []byte

//go:embed inspect.json
var InspectSchemaJSON []byte

//go:embed validate.json
var ValidateSchemaJSON []byte

// ListHandler is the tools.doc_list process handler.
// Args[0]: keyword (string, optional — empty lists all)
// Args[1]: limit (int, default 20)
func ListHandler(proc *process.Process) interface{} {
	keyword := proc.ArgsString(0)
	limit := proc.ArgsInt(1, 20)

	var results []*goudoc.Entry
	if keyword != "" {
		results = goudoc.List(goudoc.TypeProcess, goudoc.ListOption{Search: keyword})
	} else {
		results = goudoc.List(goudoc.TypeProcess)
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

// InspectHandler is the tools.doc_inspect process handler.
// Args[0]: name (string — process name, e.g. "models.user.Find")
func InspectHandler(proc *process.Process) interface{} {
	name := proc.ArgsString(0)
	entry, ok := goudoc.Get(goudoc.TypeProcess, name)
	if !ok {
		return nil
	}
	return entry
}

// ValidateHandler is the tools.doc_validate process handler.
// Args[0]: name (string — process name)
func ValidateHandler(proc *process.Process) interface{} {
	name := proc.ArgsString(0)
	return goudoc.Validate(goudoc.TypeProcess, name)
}
