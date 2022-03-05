package server

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

func init() {
	gou.RegisterProcessHandler("xiang.server.Start", ProcessStart)
	gou.RegisterProcessHandler("xiang.server.Connect", ProcessConnect)
}

// ProcessStart  xiang.server.Start
func ProcessStart(process *gou.Process) interface{} {
	process.ValidateArgNums(1)

	name := process.ArgsString(0)
	serv, has := gou.Sockets[name]
	if !has {
		exception.New("%s does not load", 400, name).Throw()
	}

	args := []interface{}{}
	if process.NumOfArgs() > 1 {
		args = process.Args[1:]
	}

	if serv.Mode != "server" {
		exception.New("%s mode [%s] not server", 400, name, serv.Mode).Throw()
	}

	serv.Start(args...)
	return nil
}

// ProcessConnect  xiang.server.Connect
func ProcessConnect(process *gou.Process) interface{} {
	process.ValidateArgNums(1)

	name := process.ArgsString(0)
	serv, has := gou.Sockets[name]
	if !has {
		exception.New("%s does not load", 400, name).Throw()
	}

	args := []interface{}{}
	if process.NumOfArgs() > 1 {
		args = process.Args[1:]
	}

	if serv.Mode != "client" {
		exception.New("%s mode [%s] not server", 400, name, serv.Mode).Throw()
	}

	serv.Connect(args...)
	return nil
}
