package wework

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.RegisterGroup("yao.wework", map[string]process.Handler{
		"decrypt": processDecrypt,
	})
}

func processDecrypt(process *process.Process) interface{} {

	process.ValidateArgNums(2)
	encodingAESKey := process.ArgsString(0)
	msgEncrypt := process.ArgsString(1)
	parseXML := false

	if process.NumOfArgsIs(3) {
		parseXML = process.ArgsBool(2)
	}

	res, err := Decrypt(encodingAESKey, msgEncrypt, parseXML)
	if err != nil {
		exception.New("error: %s msgEncrypt: %s", 400, err, msgEncrypt).Throw()
	}

	return res
}
