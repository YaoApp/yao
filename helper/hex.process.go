package helper

import (
	"encoding/hex"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
)

// ProcessHexToString xiang.helper.HexToString
func ProcessHexToString(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	switch process.Args[0].(type) {
	case string:
		return hex.EncodeToString([]byte(process.Args[0].(string)))
	case []byte:
		return hex.EncodeToString(process.Args[0].([]byte))
	}
	log.With(log.F{"input": process.Args[0]}).Error("HexToString: type does not support")
	return nil
}
