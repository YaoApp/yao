package pipe

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.Register("pipes", processPipes)
}

// processScripts
func processPipes(process *process.Process) interface{} {

	pipe, err := Get(process.ID)
	if err != nil {
		exception.New("pipes.%s not loaded", 404, process.ID).Throw()
		return nil
	}

	ctx := pipe.Create().WithGlobal(process.Global).WithSid(process.Sid)
	res, err := ctx.Exec(process.Args...)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return res
}
