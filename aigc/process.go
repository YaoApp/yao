package aigc

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.Register("aigcs", processAigcs)
}

// processScripts
func processAigcs(process *process.Process) interface{} {

	process.ValidateArgNums(1)
	aigc, err := Select(process.ID)
	if err != nil {
		exception.New("aigcs.%s not loaded", 404, process.ID).Throw()
		return nil
	}

	content := process.ArgsString(0)
	user := ""

	var option map[string]interface{} = nil
	if process.NumOfArgs() > 1 {
		user = process.ArgsString(1)
	}

	if process.NumOfArgs() > 2 {
		option = process.ArgsMap(2)
	}

	res, ex := aigc.Call(content, user, option)
	if ex != nil {
		ex.Throw()
	}

	return res
}
