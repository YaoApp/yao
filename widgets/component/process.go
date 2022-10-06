package component

import (
	"github.com/yaoapp/gou"
)

// Export process
func exportProcess() {
	gou.RegisterProcessHandler("yao.component.tagview", processTagView)
	gou.RegisterProcessHandler("yao.component.tagedit", processTagEdit)
}

func processTagView(process *gou.Process) interface{} {
	process.ValidateArgNums(3)
	value := process.Args[1]
	switch value.(type) {
	case string:
		return map[string]interface{}{"label": value, "color": "#FF0000"}
	}
	return map[string]interface{}{}
}

func processTagEdit(process *gou.Process) interface{} {
	process.ValidateArgNums(3)
	value := process.Args[1]
	switch value.(type) {
	case map[string]interface{}:
		if val, has := value.(map[string]interface{})["label"]; has {
			return val
		}
		return nil
	}
	return value
}
