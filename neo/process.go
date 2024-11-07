package neo

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/neo/message"
)

func init() {
	process.RegisterGroup("neo", map[string]process.Handler{
		"write": ProcessWrite,
	})
}

// ProcessWrite process the write request
func ProcessWrite(process *process.Process) interface{} {

	process.ValidateArgNums(2)

	w, ok := process.Args[0].(gin.ResponseWriter)
	if !ok {
		exception.New("The first argument must be a io.Writer", 400).Throw()
		return nil
	}

	data, ok := process.Args[1].([]interface{})
	if !ok {
		exception.New("The second argument must be a Array", 400).Throw()
		return nil
	}

	for _, new := range data {
		if v, ok := new.(map[string]interface{}); ok {
			newMsg := message.New().Map(v)
			newMsg.Write(w)
		}
	}

	return nil
}
