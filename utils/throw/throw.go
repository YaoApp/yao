package throw

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// Unauthorized throw a unauthorized exception
func Unauthorized(process *process.Process) interface{} {
	message := process.ArgsString(0, "Authentication required")
	exception.New(message, 401).Throw()
	return nil
}

// Forbidden throw a forbidden exception
func Forbidden(process *process.Process) interface{} {
	message := process.ArgsString(0, "Access denied")
	exception.New(message, 403).Throw()
	return nil
}

// NotFound throw a not found exception
func NotFound(process *process.Process) interface{} {
	message := process.ArgsString(0, "Resource not found")
	exception.New(message, 404).Throw()
	return nil
}

// BadRequest throw a bad request exception
func BadRequest(process *process.Process) interface{} {
	message := process.ArgsString(0, "Bad Request")
	exception.New(message, 400).Throw()
	return nil
}

// InternalError throw a internal error exception
func InternalError(process *process.Process) interface{} {
	message := process.ArgsString(0, "Internal Error")
	exception.New(message, 500).Throw()
	return nil
}

// Exception throw a exception
func Exception(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	message := process.ArgsString(0)
	code := process.ArgsInt(1)
	exception.New(message, code).Throw()
	return nil
}
